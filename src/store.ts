import { create } from "zustand";
import { isCallError } from "mygo-runtime";
import { Session, events, type Levels, type Utterance } from "./mygo";

export type Source = "mic" | "system";

interface Store {
  running: boolean;
  startedAt: number;
  sources: Record<Source, boolean>;
  levels: Levels;
  utterances: Utterance[];
  fault: string;
  toggleSource: (source: Source) => void;
  start: () => Promise<void>;
  stop: () => Promise<void>;
}

export const useStore = create<Store>((set, get) => ({
  running: false,
  startedAt: 0,
  sources: { mic: true, system: true },
  levels: { mic: 0, system: 0 },
  utterances: [],
  fault: "",
  toggleSource: (source) => set(({ sources }) => ({ sources: { ...sources, [source]: !sources[source] } })),
  start: async () => {
    await stopping;
    set({ fault: "", utterances: [] });
    try {
      await Session.start(get().sources);
      set({ running: true, startedAt: Date.now() });
    } catch (err) {
      if (!isCallError(err)) throw err;
      set({ fault: err.message });
    }
  },
  stop: async () => {
    set({ running: false });
    stopping = Session.stop();
    await stopping;
    set({ levels: { mic: 0, system: 0 } });
  },
}));

let stopping: Promise<void> | undefined;

function upsert(list: Utterance[], next: Utterance): Utterance[] {
  const index = list.findIndex((u) => u.id === next.id);
  if (next.state === "dropped") return index < 0 ? list : list.toSpliced(index, 1);
  if (index >= 0) return list.with(index, next);
  const at = list.findLastIndex((u) => u.startMs <= next.startMs) + 1;
  return list.toSpliced(at, 0, next);
}

const subscriptions = [
  events.utterance.on((u) => useStore.setState(({ utterances }) => ({ utterances: upsert(utterances, u) }))),
  events.levels.on((levels) => useStore.setState({ levels })),
  events.fault.on((fault) => useStore.setState({ fault: fault.message })),
];

import.meta.hot?.dispose(() => subscriptions.forEach((off) => off()));
