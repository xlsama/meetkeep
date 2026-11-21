use rustfft::{num_complex::Complex, Fft, FftPlanner};
use std::sync::Arc;

pub const SAMPLE_RATE: usize = 16000;
pub const FRAME_LENGTH: usize = 400;
pub const FRAME_SHIFT: usize = 160;
pub const FFT_SIZE: usize = 512;
pub const MEL_BINS: usize = 80;
pub const FIXED_FRAMES: usize = 150;

pub struct Fbank {
    window: Vec<f32>,
    filters: Vec<(usize, Vec<f32>)>,
    fft: Arc<dyn Fft<f32>>,
}

impl Fbank {
    pub fn new() -> Self {
        let a = 2.0 * std::f64::consts::PI / (FRAME_LENGTH - 1) as f64;
        let window = (0..FRAME_LENGTH)
            .map(|i| (0.5 - 0.5 * (a * i as f64).cos()).powf(0.85) as f32)
            .collect();
        Fbank {
            window,
            filters: mel_filters(),
            fft: FftPlanner::new().plan_fft_forward(FFT_SIZE),
        }
    }

    pub fn features(&self, input: &[f32]) -> (usize, Vec<f32>) {
        let mut samples = input.to_vec();
        if samples.len() < FRAME_LENGTH {
            samples.resize(FRAME_LENGTH, 0.0);
        }
        let frames = 1 + (samples.len() - FRAME_LENGTH) / FRAME_SHIFT;
        let mut out = vec![0f32; frames * MEL_BINS];
        let mut frame = [0f32; FRAME_LENGTH];
        let mut spectrum = vec![Complex::new(0f32, 0f32); FFT_SIZE];
        let mut power = [0f32; FFT_SIZE / 2];

        for f in 0..frames {
            let base = f * FRAME_SHIFT;
            frame.copy_from_slice(&samples[base..base + FRAME_LENGTH]);
            let mean = frame.iter().sum::<f32>() / FRAME_LENGTH as f32;
            for v in frame.iter_mut() {
                *v -= mean;
            }
            for i in (1..FRAME_LENGTH).rev() {
                frame[i] -= 0.97 * frame[i - 1];
            }
            frame[0] -= 0.97 * frame[0];
            for (i, slot) in spectrum.iter_mut().enumerate() {
                let v = if i < FRAME_LENGTH { frame[i] * self.window[i] } else { 0.0 };
                *slot = Complex::new(v, 0.0);
            }
            self.fft.process(&mut spectrum);
            for (i, p) in power.iter_mut().enumerate() {
                *p = spectrum[i].norm_sqr();
            }
            for (m, (first, weights)) in self.filters.iter().enumerate() {
                let energy: f32 = weights.iter().zip(&power[*first..]).map(|(w, p)| w * p).sum();
                out[f * MEL_BINS + m] = energy.max(f32::EPSILON).ln();
            }
        }

        for m in 0..MEL_BINS {
            let mean = (0..frames).map(|f| out[f * MEL_BINS + m]).sum::<f32>() / frames as f32;
            for f in 0..frames {
                out[f * MEL_BINS + m] -= mean;
            }
        }
        (frames, out)
    }
}

pub fn fixed_length(frames: usize, mut values: Vec<f32>) -> Vec<f32> {
    let target = FIXED_FRAMES * MEL_BINS;
    if frames == FIXED_FRAMES {
        return values;
    }
    if frames < FIXED_FRAMES {
        values.resize(target, 0.0);
        return values;
    }
    let start = (frames - FIXED_FRAMES) / 2;
    values[start * MEL_BINS..(start + FIXED_FRAMES) * MEL_BINS].to_vec()
}

fn mel_filters() -> Vec<(usize, Vec<f32>)> {
    let mel = |freq: f32| 1127.0 * (1.0 + freq / 700.0).ln();
    let bins = FFT_SIZE / 2;
    let low = mel(20.0);
    let high = mel(SAMPLE_RATE as f32 / 2.0);
    let delta = (high - low) / (MEL_BINS + 1) as f32;
    let width = SAMPLE_RATE as f32 / FFT_SIZE as f32;

    (0..MEL_BINS)
        .map(|index| {
            let left = low + index as f32 * delta;
            let middle = left + delta;
            let right = middle + delta;
            let mut first = None;
            let mut weights = Vec::new();
            for i in 0..bins {
                let m = mel(width * i as f32);
                if !(m > left && m < right) {
                    if first.is_some() {
                        break;
                    }
                    continue;
                }
                first.get_or_insert(i);
                weights.push(if m <= middle {
                    (m - left) / (middle - left)
                } else {
                    (right - m) / (right - middle)
                });
            }
            (first.unwrap_or(0), weights)
        })
        .collect()
}
