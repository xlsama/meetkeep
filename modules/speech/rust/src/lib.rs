mod decode;
mod detector;
mod fbank;
mod media;
mod mygo;
mod recording;
mod regions;
mod sound;
mod spectral;
mod voice;

use fbank::{Fbank, FIXED_FRAMES, MEL_BINS};
use mygo::*;
use ort::environment::Environment;
use ort::session::builder::{AutoDevicePolicy, SessionBuilder};
use ort::session::Session;
use ort::value::Tensor;
use std::collections::BTreeMap;
use std::fmt::Display;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex, OnceLock};

const RATE: f64 = 16000.0;
const WINDOW: f64 = 1.5;
const HOP: f64 = 0.6;
const BATCH: usize = 4;
/// Windows are embedded by up to this many sessions, side by side: a core
/// each embeds more of them than the cores do together in one.
const EMBEDDERS: usize = 8;

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

    fn open(&self, ctx: &Context, path: String) -> Result<Arc<dyn Sound>, Error> {
        Ok(Arc::new(sound::Held::open(ctx, &path)?))
    }
}

/// Tells who speaks when in samples.
pub(crate) fn separate(ctx: &Context, samples: &[i16], models: &Models, speakers: i64) -> Result<Vec<Segment>, Error> {
    if samples.len() < RATE as usize {
        return Err(Error::other("the audio is too short"));
    }
    let regions = regions::detect(ctx, &models.segmentation, samples)?;
    ctx.check()?;
    let windows: Vec<(f64, f64)> = windows(&regions)
        .into_iter()
        .filter(|w| slice(samples, *w).len() >= fbank::FRAME_LENGTH)
        .collect();
    if windows.is_empty() {
        return Ok(Vec::new());
    }
    let vectors = embeddings(ctx, &models.embedding, samples, &windows)?;
    let labels = spectral::labels(&vectors, (speakers > 0).then_some(speakers as usize));
    Ok(assemble(&windows, &labels))
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
    // ort's default hands every model to Core ML on macOS, which runs these
    // slower than the CPU and in several times the memory.
    Session::builder(environment)
        .map_err(fail)?
        .with_auto_device(AutoDevicePolicy::PreferCPU)
        .map_err(fail)?
        .with_intra_threads(threads)
        .map_err(fail)
}

/// Does jobs with sessions of a model, up to most of them side by side
/// that share the cores, and returns what each job made, in their order.
pub(crate) fn pooled<T: Send>(
    ctx: &Context,
    model: &str,
    jobs: usize,
    most: usize,
    run: impl Fn(&mut Session, usize) -> Result<T, Error> + Sync,
) -> Result<Vec<T>, Error> {
    if jobs == 0 {
        return Ok(Vec::new());
    }
    let cores = std::thread::available_parallelism().map_or(4, |n| n.get());
    let workers = most.min(cores).min(jobs).max(1);
    let next = AtomicUsize::new(0);
    let made: Mutex<Vec<Option<T>>> = Mutex::new((0..jobs).map(|_| None).collect());
    let failure: Mutex<Option<Error>> = Mutex::new(None);

    std::thread::scope(|scope| {
        for _ in 0..workers {
            scope.spawn(|| {
                let work = || -> Result<(), Error> {
                    let mut session = builder((cores / workers).max(1))?.commit_from_file(model).map_err(fail)?;
                    loop {
                        let i = next.fetch_add(1, Ordering::Relaxed);
                        if i >= jobs || failure.lock().map_err(fail)?.is_some() {
                            return Ok(());
                        }
                        ctx.check()?;
                        let value = run(&mut session, i)?;
                        made.lock().map_err(fail)?[i] = Some(value);
                    }
                };
                if let Err(e) = work() {
                    if let Ok(mut slot) = failure.lock() {
                        slot.get_or_insert(e);
                    }
                }
            });
        }
    });

    if let Some(e) = failure.into_inner().map_err(fail)? {
        return Err(e);
    }
    made.into_inner()
        .map_err(fail)?
        .into_iter()
        .map(|value| value.ok_or_else(|| Error::other("a part of the audio was not heard")))
        .collect()
}

fn slice(samples: &[i16], window: (f64, f64)) -> &[i16] {
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

fn embeddings(ctx: &Context, model: &str, samples: &[i16], windows: &[(f64, f64)]) -> Result<Vec<Vec<f32>>, Error> {
    let batches: Vec<&[(f64, f64)]> = windows.chunks(BATCH).collect();
    let embedded = pooled(ctx, model, batches.len(), EMBEDDERS, |session, i| {
        let fbank = Fbank::new();
        let mut data = Vec::with_capacity(batches[i].len() * FIXED_FRAMES * MEL_BINS);
        for window in batches[i] {
            let (frames, values) = fbank.features(&sound::floats(slice(samples, *window)));
            data.extend(fbank::fixed_length(frames, values));
        }
        let input = Tensor::from_array(([batches[i].len(), FIXED_FRAMES, MEL_BINS], data)).map_err(fail)?;
        let outputs = session.run(ort::inputs!["x" => input]).map_err(fail)?;
        let (shape, values) = outputs["embedding"].try_extract_tensor::<f32>().map_err(fail)?;
        Ok(values.chunks(shape[1] as usize).map(<[f32]>::to_vec).collect::<Vec<_>>())
    })?;
    Ok(embedded.into_iter().flatten().collect())
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
