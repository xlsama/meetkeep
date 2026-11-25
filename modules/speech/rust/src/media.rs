use crate::decode::OpusFile;
use crate::fail;
use crate::mygo::{Context, Error};
use crate::recording::OggOpus;
use ropus::{Channels, DecodeMode, Decoder};
use std::f64::consts::PI;
use std::fs::File;
use std::path::Path;
use symphonia::core::codecs::audio::well_known::CODEC_ID_OPUS;
use symphonia::core::codecs::audio::AudioDecoderOptions;
use symphonia::core::errors::Error as MediaError;
use symphonia::core::formats::probe::Hint;
use symphonia::core::formats::TrackType;
use symphonia::core::io::MediaSourceStream;

const RATE: u32 = 48_000;
const BLOCK: usize = 4800;

/// Writes the sum of the Ogg Opus files to another.
pub fn mix(ctx: &Context, inputs: &[String], output: &str) -> Result<(), Error> {
    let mut files = inputs.iter().map(|path| OpusFile::open(path, RATE)).collect::<Result<Vec<_>, _>>()?;
    let out = OggOpus::create(output)?;
    loop {
        ctx.check()?;
        let mut sum: Vec<i32> = Vec::new();
        for file in &mut files {
            let part = file.take(BLOCK)?;
            if part.len() > sum.len() {
                sum.resize(part.len(), 0);
            }
            for (total, sample) in sum.iter_mut().zip(&part) {
                *total += *sample as i32;
            }
        }
        if sum.is_empty() {
            break;
        }
        let mixed: Vec<i16> = sum.iter().map(|v| (*v).clamp(i16::MIN as i32, i16::MAX as i32) as i16).collect();
        out.push(&mixed)?;
    }
    out.finish()
}

/// Decodes the audio of a media file and writes it, mono at 48 kHz, as Ogg
/// Opus. It returns how long the audio lasts, in milliseconds.
pub fn import(ctx: &Context, input: &str, output: &str) -> Result<i64, Error> {
    let out = OggOpus::create(output)?;
    let written = decode(ctx, input, RATE, |samples| out.push(samples))?;
    out.finish()?;
    Ok((written * 1000 / RATE as u64) as i64)
}

/// Decodes the audio of a media file and shows emit its sound, mono at
/// rate, part after part. It returns how many samples the sound holds.
pub fn decode(
    ctx: &Context,
    input: &str,
    rate: u32,
    mut emit: impl FnMut(&[i16]) -> Result<(), Error>,
) -> Result<u64, Error> {
    let source = MediaSourceStream::new(Box::new(File::open(input).map_err(fail)?), Default::default());
    let mut hint = Hint::new();
    if let Some(extension) = Path::new(input).extension().and_then(|e| e.to_str()) {
        hint.with_extension(extension);
    }
    let mut format = symphonia::default::get_probe()
        .probe(&hint, source, Default::default(), Default::default())
        .map_err(|_| Error::other("the file's format is not supported"))?;
    let track = format.default_track(TrackType::Audio).ok_or_else(|| Error::other("the file has no audio"))?;
    let id = track.id;
    let params = track
        .codec_params
        .as_ref()
        .and_then(|p| p.audio())
        .ok_or_else(|| Error::other("the file has no audio"))?
        .clone();

    let mut resampler: Option<Resampler> = None;
    let mut written = 0u64;
    let mut heard = |mono: &[f32], from: u32| -> Result<(), Error> {
        let resampler = resampler.get_or_insert_with(|| Resampler::new(from, rate));
        let samples: Vec<i16> = resampler
            .push(mono)
            .iter()
            .map(|v| (v * 32768.0).round().clamp(i16::MIN as f32, i16::MAX as f32) as i16)
            .collect();
        written += samples.len() as u64;
        emit(&samples)
    };

    if params.codec == CODEC_ID_OPUS {
        let channels = match params.channels.as_ref().map(|c| c.count()).unwrap_or(1) {
            1 => Channels::Mono,
            2 => Channels::Stereo,
            n => return Err(Error::other(format!("Opus with {n} channels is not supported"))),
        };
        let count = channels.count();
        let mut decoder = Decoder::new(RATE, channels).map_err(fail)?;
        let mut pcm = vec![0f32; 5760 * count];
        while let Ok(Some(packet)) = format.next_packet() {
            ctx.check()?;
            if packet.track_id != id {
                continue;
            }
            if let Ok(n) = decoder.decode_float(&packet.data, &mut pcm, DecodeMode::Normal) {
                let mono: Vec<f32> = pcm[..n * count].chunks(count).map(|f| f.iter().sum::<f32>() / count as f32).collect();
                heard(&mono, RATE)?;
            }
        }
    } else {
        let mut decoder = symphonia::default::get_codecs()
            .make_audio_decoder(&params, &AudioDecoderOptions::default())
            .map_err(|_| Error::other("the file's audio codec is not supported"))?;
        let mut samples: Vec<f32> = Vec::new();
        while let Ok(Some(packet)) = format.next_packet() {
            ctx.check()?;
            if packet.track_id != id {
                continue;
            }
            match decoder.decode(&packet) {
                Ok(buffer) => {
                    let count = buffer.spec().channels().count();
                    let from = buffer.spec().rate();
                    buffer.copy_to_vec_interleaved(&mut samples);
                    let mono: Vec<f32> = samples.chunks(count).map(|f| f.iter().sum::<f32>() / count as f32).collect();
                    heard(&mono, from)?;
                }
                Err(MediaError::DecodeError(_)) => continue,
                Err(_) => break,
            }
        }
    }
    if written == 0 {
        return Err(Error::other("no audio could be decoded from the file"));
    }
    Ok(written)
}

const TAPS: usize = 32;
const PHASES: usize = 256;

/// Resampler changes the rate of a stream with a windowed sinc.
struct Resampler {
    step: f64,
    position: f64,
    kernel: Vec<f32>,
    buffer: Vec<f32>,
    same: bool,
}

impl Resampler {
    fn new(from: u32, to: u32) -> Self {
        let cutoff = (to as f64 / from as f64).min(1.0) * 0.95;
        let mut kernel = vec![0f32; (PHASES + 1) * TAPS];
        for phase in 0..=PHASES {
            let fraction = phase as f64 / PHASES as f64;
            let taps = &mut kernel[phase * TAPS..(phase + 1) * TAPS];
            let mut sum = 0.0;
            for (k, tap) in taps.iter_mut().enumerate() {
                let x = k as f64 - (TAPS / 2 - 1) as f64 - fraction;
                let sinc = if x.abs() < 1e-9 { cutoff } else { (PI * cutoff * x).sin() / (PI * x) };
                let window = 0.5 + 0.5 * (PI * x / (TAPS / 2) as f64).cos();
                *tap = (sinc * window.max(0.0)) as f32;
                sum += *tap as f64;
            }
            for tap in taps.iter_mut() {
                *tap = (*tap as f64 / sum) as f32;
            }
        }
        Resampler {
            step: from as f64 / to as f64,
            position: 0.0,
            kernel,
            buffer: vec![0.0; TAPS / 2 - 1],
            same: from == to,
        }
    }

    fn push(&mut self, input: &[f32]) -> Vec<f32> {
        if self.same {
            return input.to_vec();
        }
        self.buffer.extend_from_slice(input);
        let mut out = Vec::with_capacity((input.len() as f64 / self.step) as usize + 2);
        loop {
            let base = self.position as usize;
            if base + TAPS > self.buffer.len() {
                break;
            }
            let phase = ((self.position - base as f64) * PHASES as f64).round() as usize;
            let taps = &self.kernel[phase * TAPS..(phase + 1) * TAPS];
            out.push(dot(taps, &self.buffer[base..base + TAPS]));
            self.position += self.step;
        }
        let used = self.position as usize;
        self.buffer.drain(..used);
        self.position -= used as f64;
        out
    }
}

/// The sum of the products of taps and values, eight at a time.
fn dot(taps: &[f32], values: &[f32]) -> f32 {
    let mut sums = [0f32; 8];
    for (t, v) in taps.chunks_exact(8).zip(values.chunks_exact(8)) {
        for i in 0..8 {
            sums[i] += t[i] * v[i];
        }
    }
    sums.iter().sum()
}
