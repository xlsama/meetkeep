mod mygo;
#[cfg(windows)]
mod win;

use mygo::*;
use std::sync::Arc;

pub fn make_recorder() -> Result<Arc<dyn Recorder>, Error> {
    if cfg!(windows) {
        Ok(Arc::new(NativeRecorder))
    } else {
        Err(Error::Unsupported)
    }
}

struct NativeRecorder;

#[cfg(windows)]
impl Recorder for NativeRecorder {
    fn apps(&self) -> Result<Vec<App>, Error> {
        win::apps().map_err(|e| Error::other(e.to_string()))
    }

    fn microphone(&self) -> Result<String, Error> {
        win::microphone().map_err(|e| Error::other(e.to_string()))
    }

    fn headphones(&self) -> Result<bool, Error> {
        Ok(false)
    }

    fn frames(&self, _ctx: &Context, source: Source, options: Options) -> Stream<Result<Vec<u8>, Error>> {
        match source {
            Source::Microphone => win::frames(vec![win::Target::Microphone]),
            Source::System if options.apps.is_empty() => win::frames(vec![win::Target::AllBut(std::process::id())]),
            Source::System => win::frames(win::targets(&options.apps)),
        }
    }
}

#[cfg(not(windows))]
impl Recorder for NativeRecorder {
    fn apps(&self) -> Result<Vec<App>, Error> {
        Err(Error::Unsupported)
    }

    fn microphone(&self) -> Result<String, Error> {
        Err(Error::Unsupported)
    }

    fn headphones(&self) -> Result<bool, Error> {
        Err(Error::Unsupported)
    }

    fn frames(&self, _ctx: &Context, _source: Source, _options: Options) -> Stream<Result<Vec<u8>, Error>> {
        stream_of(std::iter::once(Err(Error::Unsupported)))
    }
}
