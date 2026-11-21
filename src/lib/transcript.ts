import { useQuery } from "@tanstack/react-query";
import type { LiveUtterance, Track, UtteranceState } from "@/mygo";
import { meetingQuery } from "@/lib/queries";
import { useRecorder } from "@/lib/recorder";

export type Line = {
  seq: number;
  track: Track;
  startMs: number;
  text: string;
  state: UtteranceState;
  error?: string;
};

export type Group = {
  key: number;
  track: Track;
  startMs: number;
  lines: Line[];
};

const none: Record<number, LiveUtterance> = {};

export function useTranscript(meetingId: string, live: boolean) {
  const detail = useQuery(meetingQuery(meetingId));
  const items = useRecorder((s) => (live && s.live.meetingId === meetingId ? s.live.items : none));

  const merged = new Map<number, Line>();
  for (const u of detail.data?.utterances ?? []) merged.set(u.seq, { ...u, state: "done" });
  for (const u of Object.values(items)) {
    if (u.state === "dropped") merged.delete(u.seq);
    else merged.set(u.seq, u);
  }
  const lines = [...merged.values()].toSorted((a, b) => a.startMs - b.startMs || a.seq - b.seq);

  const groups: Group[] = [];
  for (const line of lines) {
    const last = groups.at(-1);
    if (last && last.track === line.track) last.lines.push(line);
    else groups.push({ key: line.seq, track: line.track, startMs: line.startMs, lines: [line] });
  }

  const spoken = lines.findLast((line) => line.state === "done" && line.text.trim() !== "");
  return { groups, last: spoken, loading: detail.isPending };
}
