import { useQuery } from "@tanstack/react-query";
import { Popover } from "radix-ui";
import { useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { usePlayback } from "@/components/player";
import { SpeakerPopover } from "@/components/speaker-popover";
import type { Speaker } from "@/mygo";
import { clock } from "@/lib/format";
import type { Jump } from "@/lib/notes";
import { meetingQuery } from "@/lib/queries";
import { byTalk, speakerVoice } from "@/lib/speakers";
import { useTranscript, type Group } from "@/lib/transcript";
import { cn } from "@/lib/utils";

type Block = { seq: number; start: number; end: number };

function blocksOf(groups: Group[], speakerId: number, duration: number) {
  const blocks: Block[] = [];
  const gap = duration * 0.004;
  for (const group of groups) {
    const first = group.lines[0];
    const last = group.lines.at(-1);
    if (group.voice.speakerId !== speakerId || !first || !last) continue;
    const previous = blocks.at(-1);
    if (previous && group.startMs - previous.end < gap) previous.end = Math.max(previous.end, last.endMs);
    else blocks.push({ seq: first.seq, start: group.startMs, end: last.endMs });
  }
  return blocks;
}

export function SpeakersPane({
  meetingId,
  durationMs,
  onJump,
}: {
  meetingId: string;
  durationMs: number;
  onJump: (jump: Jump) => void;
}) {
  const speakers = useQuery(meetingQuery(meetingId)).data?.speakers ?? [];
  const { groups, lines } = useTranscript(meetingId, false);
  const duration = Math.max(durationMs, lines.at(-1)?.endMs ?? 0, 1);
  const total = speakers.reduce((sum, speaker) => sum + speaker.durationMs, 0);
  const ordered = byTalk(speakers);

  return (
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-7 py-5" data-testid="speakers-pane">
      <div className="flex flex-col gap-5">
        {ordered.map((speaker) => (
          <SpeakerRow
            key={speaker.id}
            meetingId={meetingId}
            speaker={speaker}
            speakers={speakers}
            share={total > 0 ? Math.round((speaker.durationMs / total) * 100) : 0}
            blocks={blocksOf(groups, speaker.id, duration)}
            duration={duration}
            onJump={onJump}
          />
        ))}
      </div>
    </div>
  );
}

function SpeakerRow({
  meetingId,
  speaker,
  speakers,
  share,
  blocks,
  duration,
  onJump,
}: {
  meetingId: string;
  speaker: Speaker;
  speakers: Speaker[];
  share: number;
  blocks: Block[];
  duration: number;
  onJump: (jump: Jump) => void;
}) {
  const voice = speakerVoice(speaker, speakers);
  const [editing, setEditing] = useState(false);
  const avatar = useRef<HTMLButtonElement>(null);

  return (
    <Popover.Root open={editing} onOpenChange={setEditing}>
      <div className="flex gap-3" data-testid={`speaker-lane-${speaker.id}`}>
        {speaker.self ? (
          <Avatar name={speaker.name} tone={voice.avatar} className="size-8 text-label" />
        ) : (
          <button
            ref={avatar}
            type="button"
            tabIndex={-1}
            aria-label={`改名：${speaker.name}`}
            onClick={() => setEditing((open) => !open)}
            className="shrink-0 cursor-pointer self-start rounded-full outline-none transition-shadow hover:shadow-[0_0_0_3px_rgba(0,0,0,0.08)]"
          >
            <Avatar name={speaker.name} tone={voice.avatar} className="size-8 text-label" />
          </button>
        )}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex h-6 items-center gap-2">
            {speaker.self ? (
              <span className="truncate text-ui text-foreground">{speaker.name}</span>
            ) : (
              <Popover.Trigger asChild>
                <button
                  type="button"
                  className="-mx-1.5 inline-flex h-6 min-w-0 cursor-pointer items-center rounded-md px-1.5 text-ui text-foreground outline-none transition-colors hover:bg-hover focus-visible:bg-hover data-[state=open]:bg-hover"
                >
                  <span className="truncate">{speaker.name}</span>
                </button>
              </Popover.Trigger>
            )}
            <span className="shrink-0 text-label text-muted-foreground tabular-nums">{share}%</span>
          </div>
          <Lane blocks={blocks} duration={duration} tone={voice.avatar} onJump={onJump} />
        </div>
      </div>
      {editing && !speaker.self && (
        <SpeakerPopover meetingId={meetingId} speakerId={speaker.id} avatar={avatar} rename onDone={() => setEditing(false)} />
      )}
    </Popover.Root>
  );
}

const reach = 6;

function Lane({
  blocks,
  duration,
  tone,
  onJump,
}: {
  blocks: Block[];
  duration: number;
  tone: string;
  onJump: (jump: Jump) => void;
}) {
  const [hovered, setHovered] = useState<number | null>(null);
  const { currentMs, started } = usePlayback();

  const at = (event: { clientX: number; currentTarget: HTMLElement }) => {
    const box = event.currentTarget.getBoundingClientRect();
    if (box.width <= 0) return null;
    const ms = ((event.clientX - box.left) / box.width) * duration;
    const slack = (reach / box.width) * duration;
    let found: number | null = null;
    let nearest = slack;
    for (const [index, block] of blocks.entries()) {
      const distance = ms < block.start ? block.start - ms : ms > block.end ? ms - block.end : 0;
      if (distance <= nearest) {
        nearest = distance;
        found = index;
      }
    }
    return found;
  };
  const shown = hovered === null ? undefined : blocks[hovered];
  const playing = started ? blocks.findIndex((block) => currentMs >= block.start && currentMs < block.end) : -1;

  return (
    <div
      className={cn("relative h-6", hovered !== null && "cursor-pointer")}
      onPointerMove={(event) => setHovered(at(event))}
      onPointerLeave={() => setHovered(null)}
      onClick={(event) => {
        const block = blocks[at(event) ?? -1];
        if (block) onJump({ seq: block.seq, atMs: block.start });
      }}
    >
      <div className="absolute inset-x-0 top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-secondary" />
      {blocks.map((block, index) => (
        <span
          key={block.seq}
          data-testid="speaker-block"
          className={cn(
            "absolute top-1/2 min-w-[3px] -translate-y-1/2 rounded-full transition-[height,opacity]",
            tone,
            index === hovered ? "h-2.5 opacity-100" : index === playing ? "h-2 opacity-100" : "h-1.5 opacity-75",
          )}
          style={{ left: `${(block.start / duration) * 100}%`, width: `${((block.end - block.start) / duration) * 100}%` }}
        />
      ))}
      {started && (
        <span
          aria-hidden
          className="pointer-events-none absolute top-1 bottom-1 w-0.5 -translate-x-1/2 rounded-full bg-foreground"
          style={{ left: `${Math.min(100, (currentMs / duration) * 100)}%` }}
        />
      )}
      {shown && (
        <span
          role="tooltip"
          data-testid="speaker-block-time"
          className="pointer-events-none absolute bottom-full z-10 mb-0.5 -translate-x-1/2 rounded-md bg-primary px-1.5 py-0.5 text-tiny whitespace-nowrap text-primary-foreground tabular-nums shadow-[0_4px_12px_rgba(0,0,0,0.18)]"
          style={{ left: `${((shown.start + shown.end) / 2 / duration) * 100}%` }}
        >
          {clock(shown.start)}
        </span>
      )}
    </div>
  );
}
