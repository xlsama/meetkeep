import { create } from "zustand";
import { errorMessage } from "@/lib/queries";

type Playback = {
  audio: HTMLAudioElement | null;
  current: number;
  duration: number;
  playing: boolean;
  rate: number;
  engaged: boolean;
  error: string | null;
};

export const usePlayback = create<Playback>()(() => ({
  audio: null,
  current: 0,
  duration: 0,
  playing: false,
  rate: 1,
  engaged: false,
  error: null,
}));

const rates = [1, 1.25, 1.5, 2];
let pending: number | null = null;

function fail(err: unknown) {
  if (err instanceof DOMException && err.name === "AbortError") return;
  usePlayback.setState({ error: errorMessage(err) });
}

export function resetPlayback() {
  pending = null;
  usePlayback.setState({ current: 0, playing: false, engaged: false, error: null });
}

export function attachAudio(audio: HTMLAudioElement | null, duration: number) {
  pending = null;
  usePlayback.setState({ audio, duration, rate: 1, playing: false });
}

export function seek(ms: number, play = false) {
  const { audio, duration } = usePlayback.getState();
  const at = Math.max(0, duration > 0 ? Math.min(ms, duration) : ms);
  usePlayback.setState({ current: at, engaged: true });
  if (!audio) return;
  if (audio.readyState >= 1) audio.currentTime = at / 1000;
  else pending = at;
  if (play) audio.play().catch(fail);
}

export function togglePlayback() {
  const { audio } = usePlayback.getState();
  if (!audio) return;
  usePlayback.setState({ engaged: true, error: null });
  if (audio.paused) audio.play().catch(fail);
  else audio.pause();
}

export function cycleRate() {
  const { audio, rate } = usePlayback.getState();
  const next = rates[(rates.indexOf(rate) + 1) % rates.length] ?? 1;
  usePlayback.setState({ rate: next });
  if (!audio) return;
  audio.defaultPlaybackRate = next;
  audio.playbackRate = next;
}

export function audioLoaded(audio: HTMLAudioElement) {
  if (Number.isFinite(audio.duration) && audio.duration > 0) usePlayback.setState({ duration: audio.duration * 1000 });
  if (pending !== null) {
    audio.currentTime = pending / 1000;
    pending = null;
  }
  audio.playbackRate = usePlayback.getState().rate;
}

export function audioFailed(audio: HTMLAudioElement) {
  usePlayback.setState({ playing: false, error: audio.error?.message || "录音无法播放" });
}

export function rateLabel(rate: number) {
  return `${rate.toFixed(rate * 10 === Math.round(rate * 10) ? 1 : 2)}×`;
}
