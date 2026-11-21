import type { Group } from "@/lib/transcript";

export type Segment = { key: number; tone: string; self: boolean; start: number; end: number };

export function segmentsOf(groups: Group[], duration: number) {
  const segments: Segment[] = [];
  if (duration <= 0) return segments;
  const min = duration * 0.006;
  groups.forEach((group, index) => {
    const start = index === 0 ? 0 : Math.min(group.startMs, duration);
    const end = Math.min(groups[index + 1]?.startMs ?? duration, duration);
    if (end <= start) return;
    const last = segments.at(-1);
    if (last && (last.tone === group.voice.tone || end - start < min)) last.end = end;
    else segments.push({ key: group.key, tone: group.voice.tone, self: group.voice.self, start, end });
  });
  return segments;
}
