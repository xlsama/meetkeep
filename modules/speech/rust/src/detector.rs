use crate::mygo::{Detector, Error};
use crate::{builder, fail};
use ort::session::Session;
use ort::value::Tensor;
use std::sync::Mutex;

static MODEL: &[u8] = include_bytes!("../models/silero_vad.onnx");

const WINDOW: usize = 512;
const CONTEXT: usize = 64;
const RATE: i64 = 16000;

pub struct Silero(Mutex<State>);

struct State {
    session: Session,
    memory: Vec<f32>,
    input: Vec<f32>,
    pending: Vec<f32>,
}

impl Silero {
    pub fn new() -> Result<Self, Error> {
        let session = builder(1)?.commit_from_memory(MODEL).map_err(fail)?;
        Ok(Silero(Mutex::new(State {
            session,
            memory: vec![0.0; 2 * 128],
            input: vec![0.0; CONTEXT + WINDOW],
            pending: Vec::with_capacity(4 * WINDOW),
        })))
    }
}

impl Detector for Silero {
    fn close(&self) -> Result<(), Error> {
        Ok(())
    }

    fn feed(&self, pcm: Vec<u8>) -> Result<Vec<f32>, Error> {
        let mut guard = self.0.lock().map_err(fail)?;
        let state = &mut *guard;
        state
            .pending
            .extend(pcm.chunks_exact(2).map(|b| i16::from_le_bytes([b[0], b[1]]) as f32 / 32768.0));
        let mut probabilities = Vec::with_capacity(state.pending.len() / WINDOW);
        let mut used = 0;
        while state.pending.len() - used >= WINDOW {
            state.input.copy_within(WINDOW.., 0);
            state.input[CONTEXT..].copy_from_slice(&state.pending[used..used + WINDOW]);
            used += WINDOW;
            let x = Tensor::from_array(([1, CONTEXT + WINDOW], state.input.clone())).map_err(fail)?;
            let memory = Tensor::from_array(([2, 1, 128], state.memory.clone())).map_err(fail)?;
            let rate = Tensor::from_array(((), vec![RATE])).map_err(fail)?;
            let outputs = state
                .session
                .run(ort::inputs!["input" => x, "state" => memory, "sr" => rate])
                .map_err(fail)?;
            probabilities.push(outputs["output"].try_extract_tensor::<f32>().map_err(fail)?.1[0]);
            state.memory.copy_from_slice(outputs["stateN"].try_extract_tensor::<f32>().map_err(fail)?.1);
        }
        state.pending.drain(..used);
        Ok(probabilities)
    }
}
