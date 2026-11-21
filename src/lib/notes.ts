import { useQuery } from "@tanstack/react-query";
import type { Notes } from "@/mygo";
import { notesQuery } from "@/lib/queries";
import { lineAt, type Line } from "@/lib/transcript";

export type Jump = { seq: number; atMs: number };

export function useNotes(meetingId: string) {
  const query = useQuery({ ...notesQuery(meetingId), enabled: meetingId !== "" });
  return query.data ?? null;
}

export function todoMeta(todo: { owner: string; due: string }) {
  return [todo.owner, todo.due].filter((part) => part.trim() !== "").join(" · ");
}

export function doubts(notes: Notes) {
  return notes.open.length + notes.topics.reduce((sum, t) => sum + t.points.filter((p) => p.uncertain).length, 0);
}

export function seqOf(lines: Line[], jump: Jump) {
  if (lines.some((line) => line.seq === jump.seq)) return jump.seq;
  return lineAt(lines, jump.atMs)?.seq ?? null;
}
