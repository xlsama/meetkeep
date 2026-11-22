import type { Mark } from "@/mygo";

export type Word = { at: number; end: number; ms: number };

const segmenter = new Intl.Segmenter(undefined, { granularity: "word" });

export function wordsOf(line: { text: string; marks: Mark[]; startMs: number; endMs: number }): Word[] {
  const { text } = line;
  const [head, ...rest] = line.marks;
  const marks = head?.at === 0 ? rest : line.marks;
  const words: Word[] = [];
  let from = { at: 0, ms: head?.at === 0 ? head.ms : line.startMs };
  let next = 0;
  for (const part of segmenter.segment(text)) {
    if (!part.isWordLike) continue;
    for (let mark = marks[next]; mark && mark.at <= part.index; mark = marks[++next]) from = mark;
    const to = marks[next] ?? { at: text.length, ms: Math.max(line.endMs, from.ms) };
    const ms = to.at > from.at ? from.ms + ((to.ms - from.ms) * (part.index - from.at)) / (to.at - from.at) : from.ms;
    words.push({ at: part.index, end: part.index + part.segment.length, ms: Math.round(ms) });
  }
  return words;
}

export function wordAt(words: Word[], ms: number) {
  let found = -1;
  for (const [i, word] of words.entries()) {
    if (word.ms > ms) break;
    found = i;
  }
  return found;
}

export function wordNear(words: Word[], offset: number) {
  let found = words[0];
  for (const word of words) {
    if (word.at > offset) break;
    found = word;
  }
  return found;
}
