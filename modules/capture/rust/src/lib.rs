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

impl Recorder for NativeRecorder {
    #[cfg(windows)]
    fn frames(&self, _ctx: &Context, source: Source) -> Stream<Result<Vec<u8>, Error>> {
        win::frames(match source {
            Source::Microphone => win::Target::Microphone,
            Source::System => win::Target::AllBut(std::process::id()),
        })
    }

    #[cfg(not(windows))]
    fn frames(&self, _ctx: &Context, _source: Source) -> Stream<Result<Vec<u8>, Error>> {
        stream_of(std::iter::once(Err(Error::Unsupported)))
    }
}
