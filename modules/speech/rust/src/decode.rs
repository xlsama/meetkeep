use crate::fail;
use crate::mygo::{Audio, Error};
use ogg::PacketReader;
use ropus::{Channels, DecodeMode, Decoder};
use std::collections::VecDeque;
use std::fs::File;
use std::io::BufReader;
use std::sync::Mutex;

const RATE: u32 = 16000;

/// Reads the samples of an Ogg Opus file, downmixed to mono at a rate Opus
/// decodes to. A file cut short, as by a crash, ends where it was cut.
pub struct Sound(Mutex<OpusFile>);

impl Sound {
    pub fn open(path: &str) -> Result<Self, Error> {
        Ok(Sound(Mutex::new(OpusFile::open(path, RATE)?)))
    }
}

impl Audio for Sound {
    fn close(&self) -> Result<(), Error> {
        Ok(())
    }

    fn read(&self) -> Result<Vec<u8>, Error> {
        let samples = self.0.lock().map_err(fail)?.take(RATE as usize * 10)?;
        Ok(samples.into_iter().flat_map(i16::to_le_bytes).collect())
    }
}

pub struct OpusFile {
    reader: PacketReader<BufReader<File>>,
    decoder: Decoder,
    channels: usize,
    skip: usize,
    frame: Vec<i16>,
    ready: VecDeque<i16>,
    ended: bool,
}

impl OpusFile {
    pub fn open(path: &str, rate: u32) -> Result<Self, Error> {
        let mut reader = PacketReader::new(BufReader::new(File::open(path).map_err(fail)?));
        let head = reader.read_packet_expected().map_err(fail)?.data;
        if head.len() < 19 || &head[..8] != b"OpusHead" {
            return Err(Error::other("the file is not Ogg Opus"));
        }
        let channels = match head[9] {
            1 => Channels::Mono,
            2 => Channels::Stereo,
            n => return Err(Error::other(format!("Opus with {n} channels is not supported"))),
        };
        let pre_skip = u16::from_le_bytes([head[10], head[11]]) as usize;
        reader.read_packet_expected().map_err(fail)?;
        Ok(OpusFile {
            reader,
            decoder: Decoder::new(rate, channels).map_err(fail)?,
            channels: channels.count(),
            skip: pre_skip * rate as usize / 48_000,
            frame: vec![0; rate as usize * 120 / 1000 * channels.count()],
            ready: VecDeque::new(),
            ended: false,
        })
    }

    /// Takes up to n samples: fewer only at the end of the file.
    pub fn take(&mut self, n: usize) -> Result<Vec<i16>, Error> {
        while self.ready.len() < n && !self.ended {
            let Ok(Some(packet)) = self.reader.read_packet() else {
                self.ended = true;
                break;
            };
            let Ok(count) = self.decoder.decode(&packet.data, &mut self.frame, DecodeMode::Normal) else {
                continue;
            };
            let mono = self.frame[..count * self.channels]
                .chunks(self.channels)
                .map(|f| (f.iter().map(|v| *v as i32).sum::<i32>() / self.channels as i32) as i16);
            let dropped = self.skip.min(count);
            self.skip -= dropped;
            self.ready.extend(mono.skip(dropped));
        }
        let n = n.min(self.ready.len());
        Ok(self.ready.drain(..n).collect())
    }

    pub fn all(mut self) -> Result<Vec<i16>, Error> {
        let mut samples = Vec::new();
        loop {
            let part = self.take(1 << 16)?;
            if part.is_empty() {
                return Ok(samples);
            }
            samples.extend(part);
        }
    }
}
