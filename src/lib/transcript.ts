import { useQuery } from "@tanstack/react-query";
import type { LiveUtterance, Mark, Speaker, Track, UtteranceState } from "@/mygo";
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
  marks: Mark[];
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
  continued: boolean;
};

const full = { letters: 180, ms: 60_000 };
const pause = { ms: 2000, letters: 60 };

export function spaced(before: string) {
  return /[\w.,!?;:)"']$/.test(before);
}

export function joined(lines: Line[]) {
  return lines.reduce((text, line) => text + (spaced(text) ? " " : "") + line.text, "");
}

const none: Record<number, LiveUtterance> = {};
const nobody: Speaker[] = [];
const unmarked: Mark[] = [];

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
    const shown = original || u.polished === "" ? { text: u.text, marks: u.marks } : { text: u.polished, marks: u.polishedMarks };
    merged.set(u.seq, { seq: u.seq, track: u.track, startMs: u.startMs, endMs: u.endMs, text: shown.text, marks: shown.marks ?? unmarked, speaker: u.speaker, uncertain: u.uncertain, state: "done" });
  }
  for (const u of Object.values(items)) {
    if (u.state === "dropped") merged.delete(u.seq);
    else merged.set(u.seq, { ...u, marks: unmarked, speaker: 0, uncertain: false });
  }
  const lines = [...merged.values()].toSorted((a, b) => a.startMs - b.startMs || a.seq - b.seq);

  const groups: Group[] = [];
  let letters = 0;
  for (const line of lines) {
    const voice = voiceOf(line, speakers);
    const last = groups.at(-1);
    const before = last?.lines.at(-1);
    const continued = last !== undefined && last.voice.key === voice.key && last.uncertain === line.uncertain;
    const ended =
      letters >= full.letters ||
      line.startMs - (last?.startMs ?? 0) >= full.ms ||
      (letters >= pause.letters && line.startMs - (before?.endMs ?? 0) >= pause.ms);
    if (last && continued && !ended) {
      last.lines.push(line);
      letters += line.text.length;
    } else {
      groups.push({ key: line.seq, voice, uncertain: line.uncertain, startMs: line.startMs, lines: [line], continued });
      letters = line.text.length;
    }
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
