use crate::fail;
use crate::mygo::{Error, Recording};
use ogg::writing::{PacketWriteEndInfo, PacketWriter};
use ropus::{Application, Bitrate, Channels, Encoder};
use std::fs::File;
use std::io::{BufWriter, Write};
use std::sync::Mutex;

const RATE: u32 = 48_000;
const FRAME: usize = 960;
const BITRATE: u32 = 32_000;
const SERIAL: u32 = 0x4D45_4554;
const PAGE_PACKETS: usize = 50;

pub struct OggOpus(Mutex<Option<State>>);

struct State {
    encoder: Encoder,
    writer: PacketWriter<'static, BufWriter<File>>,
    pending: Vec<i16>,
    held: Option<Vec<u8>>,
    position: u64,
    in_page: usize,
}

impl OggOpus {
    pub fn create(path: &str) -> Result<Self, Error> {
        let encoder = Encoder::builder(RATE, Channels::Mono, Application::Audio)
            .bitrate(Bitrate::Bits(BITRATE))
            .build()
            .map_err(fail)?;
        let mut writer = PacketWriter::new(BufWriter::new(File::create(path).map_err(fail)?));

        let mut head = Vec::with_capacity(19);
        head.extend_from_slice(b"OpusHead");
        head.extend_from_slice(&[1, 1]);
        head.extend_from_slice(&(encoder.lookahead() as u16).to_le_bytes());
        head.extend_from_slice(&RATE.to_le_bytes());
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

        Ok(OggOpus(Mutex::new(Some(State {
            encoder,
            writer,
            pending: Vec::with_capacity(4 * FRAME),
            held: None,
            position: 0,
            in_page: 0,
        }))))
    }
}

impl State {
    fn emit(&mut self, packet: Vec<u8>, last: bool) -> Result<(), Error> {
        self.position += FRAME as u64;
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
}

impl OggOpus {
    pub fn push(&self, samples: &[i16]) -> Result<(), Error> {
        let mut guard = self.0.lock().map_err(fail)?;
        let state = guard.as_mut().ok_or(Error::Closed)?;
        state.pending.extend_from_slice(samples);
        let mut used = 0;
        while state.pending.len() - used >= FRAME {
            let frame: Vec<i16> = state.pending[used..used + FRAME].to_vec();
            state.encode(&frame)?;
            used += FRAME;
        }
        state.pending.drain(..used);
        Ok(())
    }

    pub fn finish(&self) -> Result<(), Error> {
        let Some(mut state) = self.0.lock().map_err(fail)?.take() else { return Ok(()) };
        if !state.pending.is_empty() || state.held.is_none() {
            let mut frame = std::mem::take(&mut state.pending);
            frame.resize(FRAME, 0);
            state.encode(&frame)?;
        }
        if let Some(last) = state.held.take() {
            state.emit(last, true)?;
        }
        state.writer.inner_mut().flush().map_err(fail)
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
