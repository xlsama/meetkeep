use crate::mygo::Error;
use crate::{fail, session};
use ort::value::Tensor;

const CHUNK: usize = 160_000;
const CHUNK_SECONDS: f64 = 10.0;
const FRAME_STEP: f64 = 0.016875;
const MIN_ON: f64 = 0.25;
const MIN_OFF: f64 = 0.1;
const BATCH: usize = 8;

pub fn detect(model: &str, samples: &[f32]) -> Result<Vec<(f64, f64)>, Error> {
    let mut session = session(model)?;
    let chunks = samples.len().div_ceil(CHUNK);
    let mut raw = Vec::new();
    for first in (0..chunks).step_by(BATCH) {
        let count = BATCH.min(chunks - first);
        let mut data = vec![0f32; count * CHUNK];
        for b in 0..count {
            let start = (first + b) * CHUNK;
            let end = samples.len().min(start + CHUNK);
            data[b * CHUNK..b * CHUNK + end - start].copy_from_slice(&samples[start..end]);
        }
        let input = Tensor::from_array(([count, 1, CHUNK], data)).map_err(fail)?;
        let outputs = session.run(ort::inputs!["x" => input]).map_err(fail)?;
        let (shape, logits) = outputs["y"].try_extract_tensor::<f32>().map_err(fail)?;
        let (frames, classes) = (shape[1] as usize, shape[2] as usize);

        for b in 0..count {
            let offset = (first + b) as f64 * CHUNK_SECONDS;
            let mut began = None;
            for f in 0..frames {
                let row = &logits[(b * frames + f) * classes..(b * frames + f + 1) * classes];
                let mut best = 0;
                for c in 1..classes {
                    if row[c] > row[best] {
                        best = c;
                    }
                }
                let time = offset + f as f64 * FRAME_STEP;
                match (best > 0, began) {
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
        }
    }
    Ok(postprocess(raw))
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
