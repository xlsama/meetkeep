use crate::mygo::{Context, Error, Hearing};
use crate::{fail, pooled};
use ort::value::Tensor;

const CHUNK: usize = 160_000;
const CHUNK_SECONDS: f64 = 10.0;
const FRAME_STEP: f64 = 0.016875;
const MIN_ON: f64 = 0.25;
const MIN_OFF: f64 = 0.1;
const BATCH: usize = 4;
/// The model hears the sound with up to this many sessions, side by side.
const LISTENERS: usize = 8;
const RATE: f64 = 16000.0;
const WINDOW: usize = 512;
const SPEAKERS: usize = 3;
const TURN: usize = 15;

/// Runs the model over samples and shows visit, for each 10 s of them, what
/// the model tells of each frame: how likely nobody speaks, one of three
/// speakers does, or two of them do.
fn scan(ctx: &Context, model: &str, samples: &[i16], mut visit: impl FnMut(usize, &[f32], usize)) -> Result<(), Error> {
    let chunks = samples.len().div_ceil(CHUNK);
    let heard = pooled(ctx, model, chunks.div_ceil(BATCH), LISTENERS, |session, batch| {
        let first = batch * BATCH;
        let count = BATCH.min(chunks - first);
        let mut data = vec![0f32; count * CHUNK];
        for b in 0..count {
            let start = (first + b) * CHUNK;
            let end = samples.len().min(start + CHUNK);
            for (slot, sample) in data[b * CHUNK..].iter_mut().zip(&samples[start..end]) {
                *slot = *sample as f32 / 32768.0;
            }
        }
        let input = Tensor::from_array(([count, 1, CHUNK], data)).map_err(fail)?;
        let outputs = session.run(ort::inputs!["x" => input]).map_err(fail)?;
        let (shape, logits) = outputs["y"].try_extract_tensor::<f32>().map_err(fail)?;
        Ok((shape[1] as usize * shape[2] as usize, shape[2] as usize, logits.to_vec()))
    })?;
    for (batch, (size, classes, logits)) in heard.iter().enumerate() {
        for (b, part) in logits.chunks(*size).enumerate() {
            visit(batch * BATCH + b, part, *classes);
        }
    }
    Ok(())
}

fn best(row: &[f32]) -> usize {
    let mut best = 0;
    for c in 1..row.len() {
        if row[c] > row[best] {
            best = c;
        }
    }
    best
}

pub fn detect(ctx: &Context, model: &str, samples: &[i16]) -> Result<Vec<(f64, f64)>, Error> {
    let mut raw = Vec::new();
    scan(ctx, model, samples, |chunk, logits, classes| {
        let offset = chunk as f64 * CHUNK_SECONDS;
        let frames = logits.len() / classes;
        let mut began = None;
        for (f, row) in logits.chunks(classes).enumerate() {
            let time = offset + f as f64 * FRAME_STEP;
            match (best(row) > 0, began) {
                (true, None) => began = Some(time),
                (false, Some(start)) => {
                    raw.push((start, time));
                    began = None;
                }
                _ => {}
            }
        }
        if let Some(start) = began {
            raw.push((start, offset + frames as f64 * FRAME_STEP));
        }
    })?;
    Ok(postprocess(raw))
}

/// Tells how likely someone speaks in each window of samples, and the
/// windows at which another speaker starts to talk: one who goes on for
/// TURN frames with nobody else alone among them.
pub fn hear(ctx: &Context, model: &str, samples: &[i16]) -> Result<Hearing, Error> {
    let windows = samples.len() / WINDOW;
    let mut speech = vec![0f32; windows];
    let mut turns = Vec::new();
    scan(ctx, model, samples, |chunk, logits, classes| {
        let offset = chunk as f64 * CHUNK_SECONDS;
        let labels: Vec<usize> = logits.chunks(classes).map(best).collect();
        let alone = |label: &usize| (1..=SPEAKERS).contains(label) && classes > SPEAKERS;
        let mut current = None;
        for (f, row) in logits.chunks(classes).enumerate() {
            let window = ((offset + f as f64 * FRAME_STEP) * RATE / WINDOW as f64) as usize;
            if window >= windows {
                break;
            }
            let most = row.iter().copied().fold(f32::MIN, f32::max);
            let total: f32 = row.iter().map(|v| (v - most).exp()).sum();
            speech[window] = speech[window].max(1.0 - (row[0] - most).exp() / total);

            let label = labels[f];
            if !alone(&label) || current == Some(label) {
                continue;
            }
            if current.is_some() {
                let ahead = &labels[f..labels.len().min(f + TURN)];
                let same = ahead.iter().filter(|l| **l == label).count();
                if same < TURN / 2 || same < ahead.iter().filter(|l| alone(l)).count() {
                    continue;
                }
                turns.push(window as i64);
            }
            current = Some(label);
        }
    })?;
    turns.dedup();
    Ok(Hearing { speech, turns })
}

fn postprocess(mut segments: Vec<(f64, f64)>) -> Vec<(f64, f64)> {
    if segments.is_empty() {
        return segments;
    }
    segments.sort_by(|a, b| a.0.total_cmp(&b.0));
    let mut merged = Vec::new();
    let mut current = segments[0];
    for next in &segments[1..] {
        if next.0 - current.1 <= MIN_OFF {
            current.1 = current.1.max(next.1);
        } else {
            merged.push(current);
            current = *next;
        }
    }
    merged.push(current);
    merged.retain(|s| s.1 - s.0 >= MIN_ON);
    merged
}
