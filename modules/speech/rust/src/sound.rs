use crate::decode::OpusFile;
use crate::mygo::{Context, Error, Hearing, Models, Segment, Sound, Span, Voiceprint};
use crate::recording::Stream;
use crate::{fail, media, regions, voice};
use std::sync::{Arc, RwLock};

pub const RATE: usize = 16_000;

/// The sound of a file, in memory until it is closed.
pub struct Held(RwLock<Option<Arc<Vec<i16>>>>);

impl Held {
    /// Reads the sound of an Ogg Opus file as it is: one cut short, as by a
    /// crash, ends where it was cut. Other files are decoded as media.
    pub fn open(ctx: &Context, path: &str) -> Result<Self, Error> {
        let samples = match OpusFile::open(path, RATE as u32) {
            Ok(file) if path.ends_with(".opus") => file.all()?,
            _ => {
                let mut all = Vec::new();
                media::decode(ctx, path, RATE as u32, |part| {
                    all.extend_from_slice(part);
                    Ok(())
                })?;
                all
            }
        };
        Ok(Held(RwLock::new(Some(Arc::new(samples)))))
    }

    fn samples(&self) -> Result<Arc<Vec<i16>>, Error> {
        self.0.read().map_err(fail)?.clone().ok_or(Error::Closed)
    }
}

/// The samples of a stretch of sound: what of it there is.
pub fn stretch(samples: &[i16], start_ms: i64, end_ms: i64) -> &[i16] {
    let at = |ms: i64| (ms.max(0) as usize * RATE / 1000).min(samples.len());
    let start = at(start_ms);
    &samples[start..at(end_ms).max(start)]
}

/// Samples as the models take them.
pub fn floats(samples: &[i16]) -> Vec<f32> {
    samples.iter().map(|v| *v as f32 / 32768.0).collect()
}

impl Sound for Held {
    fn close(&self) -> Result<(), Error> {
        *self.0.write().map_err(fail)? = None;
        Ok(())
    }

    fn duration(&self) -> Result<i64, Error> {
        Ok((self.samples()?.len() * 1000 / RATE) as i64)
    }

    fn read(&self, start_ms: i64, end_ms: i64) -> Result<Vec<u8>, Error> {
        Ok(stretch(&self.samples()?, start_ms, end_ms).iter().flat_map(|v| v.to_le_bytes()).collect())
    }

    fn clip(&self, start_ms: i64, end_ms: i64) -> Result<Vec<u8>, Error> {
        let mut stream = Stream::start(Vec::new(), RATE as u32)?;
        stream.push(stretch(&self.samples()?, start_ms, end_ms))?;
        stream.finish()
    }

    fn hear(&self, ctx: &Context, model: String) -> Result<Hearing, Error> {
        regions::hear(ctx, &model, &self.samples()?)
    }

    fn embed(&self, ctx: &Context, model: String, spans: Vec<Span>) -> Result<Vec<Voiceprint>, Error> {
        voice::embed(ctx, &model, &self.samples()?, &spans)
    }

    fn separate(&self, ctx: &Context, models: Models, speakers: i64) -> Result<Vec<Segment>, Error> {
        crate::separate(ctx, &self.samples()?, &models, speakers)
    }
}
