mod decode;
mod detector;
mod fbank;
mod media;
mod mygo;
mod recording;
mod regions;
mod spectral;

use fbank::{Fbank, FIXED_FRAMES, MEL_BINS};
use mygo::*;
use ort::environment::Environment;
use ort::session::builder::SessionBuilder;
use ort::session::Session;
use ort::value::Tensor;
use rayon::prelude::*;
use std::collections::BTreeMap;
use std::fmt::Display;
use std::sync::{Arc, OnceLock};

const RATE: f64 = 16000.0;
const WINDOW: f64 = 1.5;
const HOP: f64 = 0.6;
const BATCH: usize = 16;
const GROUP: usize = 512;

pub fn make_engine() -> Result<Arc<dyn Engine>, Error> {
    Ok(Arc::new(NativeEngine))
}

struct NativeEngine;

impl Engine for NativeEngine {
    fn detector(&self) -> Result<Arc<dyn Detector>, Error> {
        Ok(Arc::new(detector::Silero::new()?))
    }

    fn record(&self, path: String) -> Result<Arc<dyn Recording>, Error> {
        Ok(Arc::new(recording::OggOpus::create(&path)?))
    }

    fn mix(&self, ctx: &Context, inputs: Vec<String>, output: String) -> Result<(), Error> {
        media::mix(ctx, &inputs, &output)
    }

    fn import(&self, ctx: &Context, input: String, output: String) -> Result<i64, Error> {
        media::import(ctx, &input, &output)
    }

    fn separate(&self, ctx: &Context, path: String, models: Models, speakers: i64) -> Result<Vec<Segment>, Error> {
        let samples = read_audio(&path)?;
        if samples.len() < RATE as usize {
            return Err(Error::other("the audio is too short"));
        }
        let regions = regions::detect(&models.segmentation, &samples)?;
        ctx.check()?;
        let windows: Vec<(f64, f64)> = windows(&regions)
            .into_iter()
            .filter(|w| slice(&samples, *w).len() >= fbank::FRAME_LENGTH)
            .collect();
        if windows.is_empty() {
            return Ok(Vec::new());
        }
        let vectors = embeddings(ctx, &models.embedding, &samples, &windows)?;
        let labels = spectral::labels(&vectors, (speakers > 0).then_some(speakers as usize));
        Ok(assemble(&windows, &labels))
    }
}

pub(crate) fn fail(e: impl Display) -> Error {
    Error::other(e.to_string())
}

pub(crate) fn builder(threads: usize) -> Result<SessionBuilder, Error> {
    static ENVIRONMENT: OnceLock<Result<Environment, String>> = OnceLock::new();
    let environment = ENVIRONMENT
        .get_or_init(|| ort::init().build().map_err(|e| e.to_string()))
        .as_ref()
        .map_err(fail)?;
    Session::builder(environment).map_err(fail)?.with_intra_threads(threads).map_err(fail)
}

pub(crate) fn session(model: &str) -> Result<Session, Error> {
    let threads = std::thread::available_parallelism().map_or(4, |n| n.get());
    builder(threads)?.commit_from_file(model).map_err(fail)
}

fn read_audio(path: &str) -> Result<Vec<f32>, Error> {
    if path.ends_with(".wav") {
        return read_wav(path);
    }
    let samples = decode::OpusFile::open(path, RATE as u32)?.all()?;
    Ok(samples.into_iter().map(|v| v as f32 / 32768.0).collect())
}

fn read_wav(path: &str) -> Result<Vec<f32>, Error> {
    let mut reader = hound::WavReader::open(path).map_err(fail)?;
    let spec = reader.spec();
    if spec.sample_rate != RATE as u32 || spec.channels != 1 || spec.bits_per_sample != 16 {
        return Err(Error::other("the audio is not 16 kHz mono 16-bit PCM"));
    }
    reader
        .samples::<i16>()
        .map(|s| s.map(|v| v as f32 / 32768.0).map_err(fail))
        .collect()
}

fn slice(samples: &[f32], window: (f64, f64)) -> &[f32] {
    let lo = ((window.0 * RATE) as isize).max(0) as usize;
    let hi = ((window.1 * RATE) as isize).max(0) as usize;
    let hi = hi.min(samples.len());
    &samples[lo.min(hi)..hi]
}

fn windows(regions: &[(f64, f64)]) -> Vec<(f64, f64)> {
    let mut out = Vec::new();
    for &(begin, end) in regions {
        let mut start = begin;
        while start + WINDOW < end {
            out.push((start, start + WINDOW));
            start += HOP;
        }
        if start < end {
            out.push(((end - WINDOW).min(start), end));
        }
    }
    out
}

fn embeddings(ctx: &Context, model: &str, samples: &[f32], windows: &[(f64, f64)]) -> Result<Vec<Vec<f32>>, Error> {
    let mut session = session(model)?;
    let mut vectors = Vec::with_capacity(windows.len());
    for group in windows.chunks(GROUP) {
        ctx.check()?;
        let features: Vec<Vec<f32>> = group
            .par_iter()
            .map_init(Fbank::new, |fbank, window| {
                let (frames, values) = fbank.features(slice(samples, *window));
                fbank::fixed_length(frames, values)
            })
            .collect();
        for batch in features.chunks(BATCH) {
            let data: Vec<f32> = batch.iter().flatten().copied().collect();
            let input = Tensor::from_array(([batch.len(), FIXED_FRAMES, MEL_BINS], data)).map_err(fail)?;
            let outputs = session.run(ort::inputs!["x" => input]).map_err(fail)?;
            let (shape, values) = outputs["embedding"].try_extract_tensor::<f32>().map_err(fail)?;
            let width = shape[1] as usize;
            vectors.extend(values.chunks(width).map(<[f32]>::to_vec));
        }
    }
    Ok(vectors)
}

fn assemble(windows: &[(f64, f64)], labels: &[usize]) -> Vec<Segment> {
    let mut flat: Vec<(usize, f64, f64)> = Vec::new();
    for (&(start, end), &label) in windows.iter().zip(labels) {
        let (start, end) = (start.max(0.0), end.max(0.0));
        if let Some(last) = flat.last_mut() {
            if label == last.0 {
                if start > last.2 {
                    flat.push((label, start, end));
                } else {
                    last.2 = end;
                }
                continue;
            }
            if start < last.2 {
                let middle = ((last.2 + start) / 2.0).max(0.0);
                last.2 = middle;
                flat.push((label, middle, end));
                continue;
            }
        }
        flat.push((label, start, end));
    }

    let mut clusters: BTreeMap<usize, Vec<(i64, i64)>> = BTreeMap::new();
    for (label, start, end) in flat {
        if end > start {
            clusters.entry(label).or_default().push(((start * 1000.0) as i64, (end * 1000.0) as i64));
        }
    }
    let mut ordered: Vec<Vec<(i64, i64)>> = clusters.into_values().collect();
    ordered.sort_by_key(|c| std::cmp::Reverse(c.iter().map(|s| s.1 - s.0).sum::<i64>()));

    let mut segments: Vec<Segment> = ordered
        .iter()
        .enumerate()
        .flat_map(|(speaker, cluster)| {
            cluster.iter().map(move |&(start_ms, end_ms)| Segment { start_ms, end_ms, speaker: speaker as i64 })
        })
        .collect();
    segments.sort_by_key(|s| s.start_ms);
    segments
}
