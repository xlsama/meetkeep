use crate::mygo::{stream_of, CaptureError, Error, Stream};
use std::collections::VecDeque;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{sync_channel, Receiver};
use std::sync::Arc;
use wasapi::{
    initialize_mta, AudioClient, DeviceEnumerator, Direction, SampleType, StreamMode, WasapiError,
    WaveFormat,
};

const ACCESS_DENIED: i32 = 0x80070005u32 as i32;

const FRAME_BYTES: usize = 3200;
const WAIT_MS: u32 = 100;

#[derive(Clone, Copy)]
pub enum Target {
    Microphone,
    /// Everything the system plays but the process and its children.
    AllBut(u32),
    /// The process and its children alone.
    #[cfg_attr(not(test), allow(dead_code))]
    Only(u32),
}

struct Frames {
    receiver: Receiver<Result<Vec<u8>, Error>>,
    stop: Arc<AtomicBool>,
    failed: bool,
}

impl Iterator for Frames {
    type Item = Result<Vec<u8>, Error>;

    fn next(&mut self) -> Option<Self::Item> {
        if self.failed {
            return None;
        }
        match self.receiver.recv().ok()? {
            Ok(frame) => Some(Ok(frame)),
            Err(error) => {
                self.failed = true;
                Some(Err(error))
            }
        }
    }
}

impl Drop for Frames {
    fn drop(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
    }
}

pub fn frames(target: Target) -> Stream<Result<Vec<u8>, Error>> {
    let (sender, receiver) = sync_channel(32);
    let stop = Arc::new(AtomicBool::new(false));
    let flag = stop.clone();
    std::thread::spawn(move || {
        let result = capture(target, &flag, |frame| sender.send(Ok(frame)).is_ok());
        if let Err(e) = result {
            let denied = matches!(e.downcast_ref(), Some(WasapiError::Windows(w)) if w.code().0 == ACCESS_DENIED);
            let _ = sender.send(Err(if denied { CaptureError::Denied.into() } else { Error::other(e.to_string()) }));
        }
    });
    stream_of(Frames { receiver, stop, failed: false })
}

fn capture(
    target: Target,
    stop: &AtomicBool,
    mut emit: impl FnMut(Vec<u8>) -> bool,
) -> Result<(), Box<dyn std::error::Error>> {
    initialize_mta().ok()?;
    let format = WaveFormat::new(16, 16, SampleType::Int, 16000, 1, None);
    let mut client = match target {
        Target::Microphone => DeviceEnumerator::new()?
            .get_default_device(Direction::Capture)?
            .get_iaudioclient()?,
        Target::AllBut(process) => AudioClient::new_application_loopback_client(process, false)?,
        Target::Only(process) => AudioClient::new_application_loopback_client(process, true)?,
    };
    let mode = StreamMode::EventsShared { autoconvert: true, buffer_duration_hns: 0 };
    client.initialize_client(&format, Direction::Capture, mode, None)?;
    let event = client.set_get_eventhandle()?;
    let reader = client.get_audiocaptureclient()?;
    client.start_stream()?;

    let mut queue: VecDeque<u8> = VecDeque::new();
    'capture: while !stop.load(Ordering::Relaxed) {
        if event.wait_for_event(WAIT_MS).is_err() {
            queue.extend(std::iter::repeat(0).take(FRAME_BYTES));
        } else {
            while reader.get_next_packet_size()?.unwrap_or(0) > 0 {
                reader.read_from_device_to_deque(&mut queue)?;
            }
        }
        while queue.len() >= FRAME_BYTES {
            if !emit(queue.drain(..FRAME_BYTES).collect()) {
                break 'capture;
            }
        }
    }
    client.stop_stream()?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{Duration, Instant};

    fn loudest(target: Target, time: Duration) -> (usize, f64) {
        let began = Instant::now();
        let mut count = 0;
        let mut peak = 0f64;
        for frame in frames(target) {
            let frame = frame.expect("capture failed");
            let samples = frame.chunks(2).map(|b| i16::from_le_bytes([b[0], b[1]]) as f64 / 32768.0);
            let rms = (samples.clone().map(|v| v * v).sum::<f64>() / samples.count() as f64).sqrt();
            peak = peak.max(rms);
            count += 1;
            if began.elapsed() > time {
                break;
            }
        }
        (count, peak)
    }

    fn tone() -> Option<u32> {
        std::env::var("MEETKEEP_TONE_PID").ok()?.parse().ok()
    }

    #[test]
    fn system_audio_but_ours_hears_another_process() {
        let Some(_) = tone() else { return };
        let (count, peak) = loudest(Target::AllBut(std::process::id()), Duration::from_secs(3));
        println!("all but ours: {count} frames, peak rms {peak:.4}");
        assert!(count >= 20, "got {count} frames in 3 s");
        assert!(peak > 0.01, "peak rms {peak}");
    }

    #[test]
    fn one_process_alone_is_heard() {
        let Some(process) = tone() else { return };
        let (count, peak) = loudest(Target::Only(process), Duration::from_secs(3));
        println!("only {process}: {count} frames, peak rms {peak:.4}");
        assert!(peak > 0.01, "peak rms {peak}");
    }

    #[test]
    fn a_silent_process_stays_silent_while_another_plays() {
        let Some(_) = tone() else { return };
        let (count, peak) = loudest(Target::Only(std::process::id()), Duration::from_secs(3));
        println!("only ours: {count} frames, peak rms {peak:.4}");
        assert!(count >= 20, "got {count} frames in 3 s");
        assert!(peak < 0.001, "peak rms {peak}");
    }

    #[test]
    fn microphone_yields_frames() {
        let Some(_) = tone() else { return };
        let (count, peak) = loudest(Target::Microphone, Duration::from_secs(2));
        println!("microphone: {count} frames, peak rms {peak:.4}");
        assert!(count >= 10, "got {count} frames in 2 s");
    }
}
