import { useQuery } from "@tanstack/react-query";
import type { LiveUtterance, Speaker, Track, UtteranceState } from "@/mygo";
import { meetingQuery } from "@/lib/queries";
import { useRecorder } from "@/lib/recorder";
import { voiceOf, type Voice } from "@/lib/speakers";
import { useWorkspace } from "@/lib/workspace";

export type Line = {
  seq: number;
  track: Track;
  startMs: number;
  endMs: number;
  text: string;
  speaker: number;
  uncertain: boolean;
  state: UtteranceState;
  error?: string;
};

export type Group = {
  key: number;
  voice: Voice;
  uncertain: boolean;
  startMs: number;
  lines: Line[];
};

const none: Record<number, LiveUtterance> = {};
const nobody: Speaker[] = [];

export function useTranscript(meetingId: string, live: boolean) {
  const detail = useQuery(meetingQuery(meetingId));
  const items = useRecorder((s) => (live && s.live.meetingId === meetingId ? s.live.items : none));
  const speakers = detail.data?.speakers ?? nobody;
  const original = useWorkspace((s) => s.original);
  const utterances = detail.data?.utterances ?? [];
  const polished = utterances.some((u) => u.filler || u.polished !== "");

  const merged = new Map<number, Line>();
  for (const u of utterances) {
    if (!original && u.filler) continue;
    const text = original || u.polished === "" ? u.text : u.polished;
    merged.set(u.seq, { seq: u.seq, track: u.track, startMs: u.startMs, endMs: u.endMs, text, speaker: u.speaker, uncertain: u.uncertain, state: "done" });
  }
  for (const u of Object.values(items)) {
    if (u.state === "dropped") merged.delete(u.seq);
    else merged.set(u.seq, { ...u, speaker: 0, uncertain: false });
  }
  const lines = [...merged.values()].toSorted((a, b) => a.startMs - b.startMs || a.seq - b.seq);

  const groups: Group[] = [];
  for (const line of lines) {
    const voice = voiceOf(line, speakers);
    const last = groups.at(-1);
    if (last && last.voice.key === voice.key && last.uncertain === line.uncertain) last.lines.push(line);
    else groups.push({ key: line.seq, voice, uncertain: line.uncertain, startMs: line.startMs, lines: [line] });
  }

  const spoken = lines.findLast((line) => line.state === "done" && line.text.trim() !== "");
  return {
    groups,
    lines,
    polished,
    last: spoken ? { line: spoken, voice: voiceOf(spoken, speakers) } : undefined,
    loading: detail.isPending,
  };
}

export function lineAt(lines: Line[], ms: number) {
  let found: Line | undefined;
  for (const line of lines) {
    if (line.state !== "done") continue;
    if (line.startMs > ms) break;
    found = line;
  }
  return found ?? lines.find((line) => line.state === "done");
}
