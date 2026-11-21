use crate::mygo::{stream_of, App, CaptureError, Error, Stream};
use std::collections::{BTreeMap, VecDeque};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{sync_channel, Receiver, SyncSender};
use std::sync::Arc;
use sysinfo::{Pid, ProcessesToUpdate, System};
use wasapi::{
    initialize_mta, AudioClient, DeviceEnumerator, Direction, SampleType, SessionState, StreamMode,
    WasapiError, WaveFormat,
};

const ACCESS_DENIED: i32 = 0x80070005u32 as i32;
const FRAME_BYTES: usize = 9600;
const WAIT_MS: u32 = 100;

type Failure = Box<dyn std::error::Error>;

#[derive(Clone, Copy)]
pub enum Target {
    Microphone,
    /// Everything the system plays but the process and its children.
    AllBut(u32),
    /// The process and its children alone.
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

pub fn frames(targets: Vec<Target>) -> Stream<Result<Vec<u8>, Error>> {
    let (sender, receiver) = sync_channel(32);
    let stop = Arc::new(AtomicBool::new(false));
    if targets.is_empty() {
        let _ = sender.send(Err(Error::other("the chosen apps are not running")));
    } else if targets.len() == 1 {
        spawn(targets[0], stop.clone(), sender);
    } else {
        let parts: Vec<Receiver<Result<Vec<u8>, Error>>> = targets
            .iter()
            .map(|target| {
                let (part, frames) = sync_channel(32);
                spawn(*target, stop.clone(), part);
                frames
            })
            .collect();
        std::thread::spawn(move || mix(parts, sender));
    }
    stream_of(Frames { receiver, stop, failed: false })
}

fn spawn(target: Target, stop: Arc<AtomicBool>, sender: SyncSender<Result<Vec<u8>, Error>>) {
    std::thread::spawn(move || {
        let result = capture(target, &stop, |frame| sender.send(Ok(frame)).is_ok());
        if let Err(e) = result {
            let denied = matches!(e.downcast_ref(), Some(WasapiError::Windows(w)) if w.code().0 == ACCESS_DENIED);
            let _ = sender.send(Err(if denied { CaptureError::Denied.into() } else { Error::other(e.to_string()) }));
        }
    });
}

fn mix(parts: Vec<Receiver<Result<Vec<u8>, Error>>>, sender: SyncSender<Result<Vec<u8>, Error>>) {
    while let Ok(first) = parts[0].recv() {
        let mut sum = match first {
            Ok(frame) => samples(&frame),
            Err(e) => {
                let _ = sender.send(Err(e));
                return;
            }
        };
        for part in &parts[1..] {
            if let Ok(Ok(frame)) = part.try_recv() {
                for (total, sample) in sum.iter_mut().zip(samples(&frame)) {
                    *total += sample;
                }
            }
        }
        let frame = sum
            .iter()
            .flat_map(|v| ((*v).clamp(i16::MIN as i32, i16::MAX as i32) as i16).to_le_bytes())
            .collect();
        if sender.send(Ok(frame)).is_err() {
            return;
        }
    }
}

fn samples(frame: &[u8]) -> Vec<i32> {
    frame.chunks_exact(2).map(|b| i16::from_le_bytes([b[0], b[1]]) as i32).collect()
}

fn capture(target: Target, stop: &AtomicBool, mut emit: impl FnMut(Vec<u8>) -> bool) -> Result<(), Failure> {
    initialize_mta().ok()?;
    let format = WaveFormat::new(16, 16, SampleType::Int, 48000, 1, None);
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

pub fn microphone() -> Result<String, Failure> {
    initialize_mta().ok()?;
    Ok(DeviceEnumerator::new()?.get_default_device(Direction::Capture)?.get_friendlyname()?)
}

fn processes() -> Result<System, Failure> {
    let mut system = System::new()?;
    system.refresh_processes(ProcessesToUpdate::All, true);
    Ok(system)
}

fn name(system: &System, process: u32) -> Option<String> {
    Some(system.process(Pid::from_u32(process))?.name().to_string_lossy().to_lowercase())
}

/// The topmost ancestor of process that runs the same executable: the app's own process.
fn root(system: &System, process: u32) -> u32 {
    let Some(own) = name(system, process) else { return process };
    let mut current = process;
    while let Some(parent) = system.process(Pid::from_u32(current)).and_then(|p| p.parent()) {
        if name(system, parent.as_u32()).as_deref() != Some(&own) {
            break;
        }
        current = parent.as_u32();
    }
    current
}

pub fn apps() -> Result<Vec<App>, Failure> {
    initialize_mta().ok()?;
    let system = processes()?;
    let own = name(&system, std::process::id());
    let mut found: BTreeMap<String, bool> = BTreeMap::new();
    for device in &DeviceEnumerator::new()?.get_device_collection(Direction::Render)? {
        let sessions = device?.get_iaudiosessionmanager()?.get_audiosessionenumerator()?;
        for i in 0..sessions.get_count()? {
            let session = sessions.get_session(i)?;
            let Some(id) = name(&system, session.get_process_id()?) else { continue };
            if Some(&id) == own.as_ref() {
                continue;
            }
            let playing = session.get_state()? == SessionState::Active;
            *found.entry(id).or_default() |= playing;
        }
    }
    let mut apps: Vec<App> = found
        .into_iter()
        .map(|(id, playing)| App {
            name: id.trim_end_matches(".exe").to_string(),
            id,
            icon: Vec::new(),
            playing,
        })
        .collect();
    apps.sort_by_key(|app| !app.playing);
    Ok(apps)
}

pub fn targets(apps: &[String]) -> Vec<Target> {
    let Ok(system) = processes() else { return Vec::new() };
    let mut roots: Vec<u32> = system
        .processes()
        .iter()
        .filter(|(_, process)| apps.contains(&process.name().to_string_lossy().to_lowercase()))
        .map(|(pid, _)| root(&system, pid.as_u32()))
        .collect();
    roots.sort_unstable();
    roots.dedup();
    roots.into_iter().map(Target::Only).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{Duration, Instant};

    fn loudest(target: Target, time: Duration) -> (usize, f64) {
        let began = Instant::now();
        let mut count = 0;
        let mut peak = 0f64;
        for frame in frames(vec![target]) {
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
    fn the_tone_player_is_listed_and_found() {
        let Some(process) = tone() else { return };
        let listed = apps().expect("apps");
        println!("apps: {:?}", listed.iter().map(|a| (&a.id, a.playing)).collect::<Vec<_>>());
        let system = processes().expect("processes");
        let player = name(&system, process).expect("the player's name");
        assert!(listed.iter().any(|a| a.id == player && a.playing), "{player} is not listed as playing");
        let found = targets(&[player]);
        assert!(!found.is_empty());
        let (_, peak) = loudest(found[0], Duration::from_secs(3));
        assert!(peak > 0.01, "peak rms {peak}");
    }

    #[test]
    fn two_targets_mix() {
        let Some(process) = tone() else { return };
        let began = Instant::now();
        let mut peak = 0f64;
        for frame in frames(vec![Target::Only(std::process::id()), Target::Only(process)]) {
            let frame = frame.expect("capture failed");
            let values = frame.chunks(2).map(|b| i16::from_le_bytes([b[0], b[1]]) as f64 / 32768.0);
            peak = peak.max((values.clone().map(|v| v * v).sum::<f64>() / values.count() as f64).sqrt());
            if began.elapsed() > Duration::from_secs(3) {
                break;
            }
        }
        assert!(peak > 0.01, "peak rms {peak}");
    }

    #[test]
    fn microphone_yields_frames() {
        let Some(_) = tone() else { return };
        let (count, peak) = loudest(Target::Microphone, Duration::from_secs(2));
        println!("microphone: {count} frames, peak rms {peak:.4}");
        assert!(count >= 10, "got {count} frames in 2 s");
    }
}
