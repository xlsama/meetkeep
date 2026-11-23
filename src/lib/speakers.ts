import type { Speaker, Track } from "@/mygo";

const trackName: Record<Track, string> = { mic: "我", system: "对方", file: "说话人" };
export const tones = ["bg-them", "bg-them-2", "bg-them-3", "bg-them-4", "bg-them-5", "bg-them-6"] as const;

export type Voice = {
  key: string;
  speakerId: number | null;
  name: string;
  self: boolean;
  tone: string;
  avatar: string;
};

export function voiceOf(line: { speaker: number; track: Track }, speakers: Speaker[]): Voice {
  const found = line.speaker === 0 ? undefined : speakers.find((s) => s.id === line.speaker);
  if (found) return speakerVoice(found, speakers);
  const self = line.track === "mic";
  return {
    key: `track-${line.track}`,
    speakerId: null,
    name: trackName[line.track],
    self,
    tone: self ? "bg-foreground" : "bg-them",
    avatar: self ? "bg-secondary-foreground" : "bg-them",
  };
}

export function speakerVoice(speaker: Speaker, speakers: Speaker[]): Voice {
  if (speaker.self) {
    return {
      key: `speaker-${speaker.id}`,
      speakerId: speaker.id,
      name: speaker.name,
      self: true,
      tone: "bg-foreground",
      avatar: "bg-secondary-foreground",
    };
  }
  const index = speakers.filter((s) => !s.self).findIndex((s) => s.id === speaker.id);
  const tone = tones[Math.max(0, index) % tones.length] ?? "bg-them";
  return { key: `speaker-${speaker.id}`, speakerId: speaker.id, name: speaker.name, self: false, tone, avatar: tone };
}

export function speakerNamed(name: string, speakers: Speaker[]) {
  const owner = name.trim();
  if (owner === "") return undefined;
  const exact = speakers.find((s) => s.name === owner);
  if (exact) return exact;
  const found = Array.from(owner).length < 2 ? [] : speakers.filter((s) => s.name.includes(owner));
  return found.length === 1 ? found[0] : undefined;
}

export function ordered(speakers: Speaker[]) {
  return [...speakers.filter((s) => !s.self), ...speakers.filter((s) => s.self)];
}

export function byTalk(speakers: Speaker[]) {
  return speakers.toSorted((a, b) => b.durationMs - a.durationMs);
}

export function initial(name: string) {
  const numbered = /^说话人\s*(\d+)$/.exec(name.trim());
  return numbered?.[1] ?? Array.from(name.trim())[0] ?? "?";
}
