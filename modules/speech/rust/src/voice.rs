use crate::mygo::{Context, Error, Span, Voiceprint};
use crate::{builder, decode, fail};
use ort::session::Session;
use ort::value::Tensor;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex;

const RATE: usize = 16_000;
/// The model's memory grows with its input: longer spans are embedded in
/// parts, which are averaged by their length.
const PART: usize = 10 * RATE;
const SHORTEST: usize = RATE / 20;
/// Spans are embedded by this many sessions, side by side: each more
/// takes as much memory again.
const WORKERS: usize = 2;

/// Embeds the voice heard in each span of an Ogg Opus file with ReDimNet2
/// (raw 16 kHz waveform in, a unit vector out). A span too short to tell a
/// voice from gets an empty vector.
pub fn embed(ctx: &Context, path: &str, model: &str, spans: &[Span]) -> Result<Vec<Voiceprint>, Error> {
    let samples: Vec<f32> =
        decode::OpusFile::open(path, RATE as u32)?.all()?.into_iter().map(|v| v as f32 / 32768.0).collect();
    let cores = std::thread::available_parallelism().map_or(4, |n| n.get());
    let workers = WORKERS.min(cores).min(spans.len()).max(1);
    let next = AtomicUsize::new(0);
    let vectors: Mutex<Vec<Vec<f32>>> = Mutex::new(vec![Vec::new(); spans.len()]);
    let failure: Mutex<Option<Error>> = Mutex::new(None);

    std::thread::scope(|scope| {
        for _ in 0..workers {
            scope.spawn(|| {
                let work = || -> Result<(), Error> {
                    let mut session = builder((cores / workers).max(1))?.commit_from_file(model).map_err(fail)?;
                    loop {
                        let i = next.fetch_add(1, Ordering::Relaxed);
                        if i >= spans.len() || failure.lock().map_err(fail)?.is_some() {
                            return Ok(());
                        }
                        ctx.check()?;
                        let start = (spans[i].start_ms.max(0) as usize * RATE / 1000).min(samples.len());
                        let end = (spans[i].end_ms.max(0) as usize * RATE / 1000).clamp(start, samples.len());
                        let vector = whole(&mut session, &samples[start..end])?;
                        vectors.lock().map_err(fail)?[i] = vector;
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
    Ok(vectors.into_inner().map_err(fail)?.into_iter().map(|vector| Voiceprint { vector }).collect())
}

fn whole(session: &mut Session, samples: &[f32]) -> Result<Vec<f32>, Error> {
    if samples.len() < SHORTEST {
        return Ok(Vec::new());
    }
    let parts = samples.len().div_ceil(PART);
    let mut sum: Vec<f32> = Vec::new();
    for i in 0..parts {
        let part = &samples[samples.len() * i / parts..samples.len() * (i + 1) / parts];
        let vector = one(session, part)?;
        if sum.is_empty() {
            sum = vec![0.0; vector.len()];
        }
        for (total, v) in sum.iter_mut().zip(&vector) {
            *total += v * part.len() as f32;
        }
    }
    Ok(unit(sum))
}

fn one(session: &mut Session, samples: &[f32]) -> Result<Vec<f32>, Error> {
    let input = Tensor::from_array(([1, samples.len()], samples.to_vec())).map_err(fail)?;
    let outputs = session.run(ort::inputs!["wav" => input]).map_err(fail)?;
    let (_, values) = outputs["emb"].try_extract_tensor::<f32>().map_err(fail)?;
    Ok(unit(values.to_vec()))
}

fn unit(mut vector: Vec<f32>) -> Vec<f32> {
    let norm = vector.iter().map(|v| v * v).sum::<f32>().sqrt();
    if norm > 0.0 {
        vector.iter_mut().for_each(|v| *v /= norm);
    }
    vector
}
