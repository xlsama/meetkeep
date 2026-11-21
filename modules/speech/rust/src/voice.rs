use crate::mygo::{Context, Error, Span, Voiceprint};
use crate::{decode, fail, session};
use ort::session::Session;
use ort::value::Tensor;

const RATE: usize = 16_000;
/// The model's memory grows with its input: longer spans are embedded in
/// parts, which are averaged by their length.
const PART: usize = 20 * RATE;
const SHORTEST: usize = RATE / 20;

/// Embeds the voice heard in each span of an Ogg Opus file with ReDimNet2
/// (raw 16 kHz waveform in, a unit vector out). A span too short to tell a
/// voice from gets an empty vector.
pub fn embed(ctx: &Context, path: &str, model: &str, spans: &[Span]) -> Result<Vec<Voiceprint>, Error> {
    let samples: Vec<f32> =
        decode::OpusFile::open(path, RATE as u32)?.all()?.into_iter().map(|v| v as f32 / 32768.0).collect();
    let mut session = session(model)?;
    let mut voices = Vec::with_capacity(spans.len());
    for span in spans {
        ctx.check()?;
        let start = (span.start_ms.max(0) as usize * RATE / 1000).min(samples.len());
        let end = (span.end_ms.max(0) as usize * RATE / 1000).clamp(start, samples.len());
        voices.push(Voiceprint { vector: whole(&mut session, &samples[start..end])? });
    }
    Ok(voices)
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
