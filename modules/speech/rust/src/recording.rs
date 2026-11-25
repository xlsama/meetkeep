use crate::fail;
use crate::mygo::{Error, Recording};
use ogg::writing::{PacketWriteEndInfo, PacketWriter};
use ropus::{Application, Bitrate, Channels, Encoder};
use std::fs::File;
use std::io::{BufWriter, Write};
use std::sync::Mutex;

const RATE: u32 = 48_000;
const BITRATE: u32 = 32_000;
const SERIAL: u32 = 0x4D45_4554;
const PAGE_PACKETS: usize = 50;

/// Writes mono sound into an Ogg Opus stream, 20 ms to a packet.
pub struct Stream<W: Write + 'static> {
    encoder: Encoder,
    writer: PacketWriter<'static, W>,
    frame: usize,
    pending: Vec<i16>,
    held: Option<Vec<u8>>,
    position: u64,
    in_page: usize,
}

impl<W: Write + 'static> Stream<W> {
    /// Starts a stream of sound sampled at rate, which Opus takes: 8, 12,
    /// 16, 24 or 48 kHz.
    pub fn start(to: W, rate: u32) -> Result<Self, Error> {
        let encoder = Encoder::builder(rate, Channels::Mono, Application::Audio)
            .bitrate(Bitrate::Bits(BITRATE))
            .build()
            .map_err(fail)?;
        let mut writer = PacketWriter::new(to);

        let mut head = Vec::with_capacity(19);
        head.extend_from_slice(b"OpusHead");
        head.extend_from_slice(&[1, 1]);
        head.extend_from_slice(&((encoder.lookahead() * RATE / rate) as u16).to_le_bytes());
        head.extend_from_slice(&rate.to_le_bytes());
        head.extend_from_slice(&[0, 0, 0]);
        writer.write_packet(head, SERIAL, PacketWriteEndInfo::EndPage, 0).map_err(fail)?;

        let vendor = b"MeetKeep";
        let mut tags = Vec::new();
        tags.extend_from_slice(b"OpusTags");
        tags.extend_from_slice(&(vendor.len() as u32).to_le_bytes());
        tags.extend_from_slice(vendor);
        tags.extend_from_slice(&0u32.to_le_bytes());
        writer.write_packet(tags, SERIAL, PacketWriteEndInfo::EndPage, 0).map_err(fail)?;
        writer.inner_mut().flush().map_err(fail)?;

        let frame = rate as usize / 50;
        Ok(Stream { encoder, writer, frame, pending: Vec::with_capacity(4 * frame), held: None, position: 0, in_page: 0 })
    }

    fn emit(&mut self, packet: Vec<u8>, last: bool) -> Result<(), Error> {
        self.position += RATE as u64 / 50;
        self.in_page += 1;
        let end = if last {
            PacketWriteEndInfo::EndStream
        } else if self.in_page >= PAGE_PACKETS {
            PacketWriteEndInfo::EndPage
        } else {
            PacketWriteEndInfo::NormalPacket
        };
        let flush = !matches!(end, PacketWriteEndInfo::NormalPacket);
        self.writer.write_packet(packet, SERIAL, end, self.position).map_err(fail)?;
        if flush {
            self.in_page = 0;
            self.writer.inner_mut().flush().map_err(fail)?;
        }
        Ok(())
    }

    fn encode(&mut self, frame: &[i16]) -> Result<(), Error> {
        let mut packet = vec![0u8; 4000];
        let n = self.encoder.encode(frame, &mut packet).map_err(fail)?;
        packet.truncate(n);
        if let Some(previous) = self.held.replace(packet) {
            self.emit(previous, false)?;
        }
        Ok(())
    }

    pub fn push(&mut self, samples: &[i16]) -> Result<(), Error> {
        self.pending.extend_from_slice(samples);
        let mut used = 0;
        while self.pending.len() - used >= self.frame {
            let frame: Vec<i16> = self.pending[used..used + self.frame].to_vec();
            self.encode(&frame)?;
            used += self.frame;
        }
        self.pending.drain(..used);
        Ok(())
    }

    /// Ends the stream and returns what it was written to.
    pub fn finish(mut self) -> Result<W, Error> {
        if !self.pending.is_empty() || self.held.is_none() {
            let mut frame = std::mem::take(&mut self.pending);
            frame.resize(self.frame, 0);
            self.encode(&frame)?;
        }
        if let Some(last) = self.held.take() {
            self.emit(last, true)?;
        }
        self.writer.inner_mut().flush().map_err(fail)?;
        Ok(self.writer.into_inner())
    }
}

/// An Ogg Opus file being written from 48 kHz sound.
pub struct OggOpus(Mutex<Option<Stream<BufWriter<File>>>>);

impl OggOpus {
    pub fn create(path: &str) -> Result<Self, Error> {
        let file = BufWriter::new(File::create(path).map_err(fail)?);
        Ok(OggOpus(Mutex::new(Some(Stream::start(file, RATE)?))))
    }

    pub fn push(&self, samples: &[i16]) -> Result<(), Error> {
        self.0.lock().map_err(fail)?.as_mut().ok_or(Error::Closed)?.push(samples)
    }

    pub fn finish(&self) -> Result<(), Error> {
        let Some(stream) = self.0.lock().map_err(fail)?.take() else { return Ok(()) };
        stream.finish().map(|_| ())
    }
}

impl Recording for OggOpus {
    fn write(&self, pcm: Vec<u8>) -> Result<(), Error> {
        let samples: Vec<i16> = pcm.chunks_exact(2).map(|b| i16::from_le_bytes([b[0], b[1]])).collect();
        self.push(&samples)
    }

    fn close(&self) -> Result<(), Error> {
        self.finish()
    }
}
