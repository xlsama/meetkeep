import { create } from "zustand";

export const useSample = create<{ playing: string | null }>()(() => ({ playing: null }));

let audio: HTMLAudioElement | null = null;
let pauseMain: (() => void) | null = null;

export function bindMain(pause: () => void) {
  pauseMain = pause;
  return () => {
    if (pauseMain === pause) pauseMain = null;
  };
}

let frame = 0;

export function stopSample() {
  cancelAnimationFrame(frame);
  audio?.pause();
  if (useSample.getState().playing !== null) useSample.setState({ playing: null });
}

function ready(el: HTMLAudioElement) {
  if (el.readyState >= 1) return Promise.resolve();
  return new Promise<void>((resolve, reject) => {
    const done = () => {
      el.removeEventListener("loadedmetadata", done);
      el.removeEventListener("error", fail);
      resolve();
    };
    const fail = () => {
      el.removeEventListener("loadedmetadata", done);
      el.removeEventListener("error", fail);
      reject(new Error(el.error?.message || "录音无法播放"));
    };
    el.addEventListener("loadedmetadata", done);
    el.addEventListener("error", fail);
  });
}

export async function toggleSample(key: string, meetingId: string, startMs: number, endMs: number) {
  if (useSample.getState().playing === key) {
    stopSample();
    return;
  }
  stopSample();
  pauseMain?.();
  audio ??= new Audio();
  const el = audio;
  const src = `recording://localhost/${meetingId}/audio`;
  if (el.src !== src) {
    el.src = src;
    el.preload = "metadata";
  }
  useSample.setState({ playing: key });
  try {
    await ready(el);
    if (useSample.getState().playing !== key) return;
    el.currentTime = startMs / 1000;
    await el.play();
  } catch (err) {
    if (useSample.getState().playing === key) stopSample();
    if (err instanceof DOMException && err.name === "AbortError") return;
    throw err;
  }
  const tick = () => {
    if (useSample.getState().playing !== key) return;
    if (el.paused || el.currentTime * 1000 >= endMs) {
      stopSample();
      return;
    }
    frame = requestAnimationFrame(tick);
  };
  frame = requestAnimationFrame(tick);
}

