import { useEffect, useState } from "react";
import { isMyGo } from "mygo-runtime";
import { create } from "zustand";
import { events, Recorder, type Fault, type Levels, type LiveUtterance, type Status } from "@/mygo";

type Live = { meetingId: string | null; items: Record<number, LiveUtterance> };

type RecorderState = {
  status: Status;
  ready: boolean;
  live: Live;
  levels: Levels;
  fault: Fault | null;
  dismissFault: () => void;
};

const idle: Status = { phase: "idle", meeting: null, elapsedMs: 0, at: Date.now() };
const silent: Levels = { mic: 0, system: 0 };

export const useRecorder = create<RecorderState>()((set) => ({
  status: idle,
  ready: false,
  live: { meetingId: null, items: {} },
  levels: silent,
  fault: null,
  dismissFault: () => set({ fault: null }),
}));

function liveFor(live: Live, meetingId: string): Live {
  return live.meetingId === meetingId ? live : { meetingId, items: {} };
}

function applyStatus(status: Status) {
  useRecorder.setState((state) => {
    const meetingId = status.meeting?.id ?? null;
    const changed = meetingId !== null && meetingId !== state.live.meetingId;
    return {
      status,
      ready: true,
      live: changed ? { meetingId, items: {} } : state.live,
      levels: status.phase === "recording" ? state.levels : silent,
      fault: changed ? null : state.fault,
    };
  });
}

function applyUtterance(u: LiveUtterance, keepNewer: boolean) {
  useRecorder.setState((state) => {
    const live = liveFor(state.live, u.meetingId);
    if (keepNewer && live.items[u.seq]) return state;
    return { live: { meetingId: live.meetingId, items: { ...live.items, [u.seq]: u } } };
  });
}

let started = false;

export function startRecorder() {
  if (started || !isMyGo()) return;
  started = true;
  events.status.on(applyStatus);
  events.utterance.on((u) => applyUtterance(u, false));
  events.levels.on((levels) => {
    if (useRecorder.getState().status.phase === "recording") useRecorder.setState({ levels });
  });
  events.fault.on((fault) => useRecorder.setState({ fault }));
  void Recorder.snapshot().then(({ status, pending }) => {
    const current = useRecorder.getState();
    if (!current.ready || current.status.at <= status.at) applyStatus(status);
    for (const u of pending) applyUtterance(u, true);
  });
}

export function isActive(status: Status) {
  return status.phase !== "idle" && status.meeting !== null;
}

export function elapsedOf(status: Status, now: number) {
  return status.phase === "recording" ? status.elapsedMs + Math.max(0, now - status.at) : status.elapsedMs;
}

export function useElapsed(status: Status) {
  const [now, setNow] = useState(() => Date.now());
  const running = status.phase === "recording";
  useEffect(() => {
    if (!running) return;
    let timer = 0;
    const tick = () => {
      const current = Date.now();
      setNow(current);
      const elapsed = status.elapsedMs + (current - status.at);
      timer = window.setTimeout(tick, 1000 - (((elapsed % 1000) + 1000) % 1000) + 5);
    };
    tick();
    return () => window.clearTimeout(timer);
  }, [running, status.elapsedMs, status.at]);
  return elapsedOf(status, running ? Math.max(now, status.at) : now);
}
