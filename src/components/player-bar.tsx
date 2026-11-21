import { Pause, Play } from "lucide-react";
import { useEffect, useRef, type KeyboardEvent, type PointerEvent } from "react";
import { clock } from "@/lib/format";
import {
  attachAudio,
  audioFailed,
  audioLoaded,
  cycleRate,
  rateLabel,
  seek,
  togglePlayback,
  usePlayback,
} from "@/lib/player";
import type { Group } from "@/lib/transcript";
import { cn } from "@/lib/utils";

type Segment = { key: number; tone: string; self: boolean; size: number };

function segmentsOf(groups: Group[], duration: number) {
  const segments: Segment[] = [];
  if (duration <= 0) return segments;
  const min = duration * 0.006;
  groups.forEach((group, index) => {
    const start = index === 0 ? 0 : Math.min(group.startMs, duration);
    const end = Math.min(groups[index + 1]?.startMs ?? duration, duration);
    const size = Math.max(0, end - start);
    const last = segments.at(-1);
    if (last && (last.tone === group.voice.tone || size < min)) last.size += size;
    else segments.push({ key: group.key, tone: group.voice.tone, self: group.voice.self, size });
  });
  return segments.filter((segment) => segment.size > 0);
}

export function PlayerBar({ meetingId, durationMs, groups }: { meetingId: string; durationMs: number; groups: Group[] }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const playing = usePlayback((s) => s.playing);
  const current = usePlayback((s) => s.current);
  const duration = usePlayback((s) => s.duration);
  const rate = usePlayback((s) => s.rate);

  useEffect(() => {
    const audio = audioRef.current;
    attachAudio(audio, durationMs);
    return () => {
      audio?.pause();
      attachAudio(null, 0);
    };
  }, [meetingId, durationMs]);

  useEffect(() => {
    if (!playing) return;
    let frame = 0;
    const tick = () => {
      const audio = audioRef.current;
      if (audio) usePlayback.setState({ current: audio.currentTime * 1000 });
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [playing]);

  const segments = segmentsOf(groups, duration);
  const ratio = duration > 0 ? Math.min(1, Math.max(0, current / duration)) : 0;

  return (
    <footer className="flex h-[60px] shrink-0 items-center gap-3.5 border-t border-divider pr-6 pl-8">
      <audio
        ref={audioRef}
        src={`recording://localhost/${meetingId}/audio`}
        preload="metadata"
        onPlay={() => usePlayback.setState({ playing: true, engaged: true })}
        onPause={(event) => usePlayback.setState({ playing: false, current: event.currentTarget.currentTime * 1000 })}
        onSeeked={(event) => {
          if (event.currentTarget.paused) usePlayback.setState({ current: event.currentTarget.currentTime * 1000 });
        }}
        onLoadedMetadata={(event) => audioLoaded(event.currentTarget)}
        onDurationChange={(event) => audioLoaded(event.currentTarget)}
        onError={(event) => audioFailed(event.currentTarget)}
      />
      <button
        type="button"
        aria-label={playing ? "暂停" : "播放"}
        data-testid="player-toggle"
        onClick={togglePlayback}
        className="inline-flex size-[34px] shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors outline-none hover:bg-[#262626] focus-visible:ring-[3px] focus-visible:ring-black/15"
      >
        {playing ? <Pause className="size-3.5" /> : <Play className="size-3.5" />}
      </button>
      <span className="font-mono text-meta text-foreground tabular-nums" data-testid="player-current">
        {clock(current)}
      </span>
      <SeekBar segments={segments} ratio={ratio} current={current} duration={duration} />
      <span className="font-mono text-meta text-muted-foreground tabular-nums" data-testid="player-duration">
        {clock(duration)}
      </span>
      <button
        type="button"
        aria-label="播放速度"
        data-testid="player-speed"
        onClick={cycleRate}
        className="h-7 shrink-0 rounded-sm px-2.5 text-label font-medium text-subtle tabular-nums transition-colors outline-none hover:bg-hover focus-visible:ring-[3px] focus-visible:ring-black/10"
      >
        {rateLabel(rate)}
      </button>
    </footer>
  );
}

function SeekBar({
  segments,
  ratio,
  current,
  duration,
}: {
  segments: Segment[];
  ratio: number;
  current: number;
  duration: number;
}) {
  const dragging = useRef(false);

  const seekTo = (event: PointerEvent<HTMLDivElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (rect.width <= 0 || duration <= 0) return;
    seek(Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width)) * duration);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const step = { ArrowLeft: -5000, ArrowDown: -5000, ArrowRight: 5000, ArrowUp: 5000 }[event.key];
    if (step !== undefined) seek(current + step);
    else if (event.key === "Home") seek(0);
    else if (event.key === "End") seek(duration);
    else return;
    event.preventDefault();
  };

  return (
    <div
      role="slider"
      tabIndex={0}
      aria-label="播放进度"
      aria-valuemin={0}
      aria-valuemax={Math.round(duration / 1000)}
      aria-valuenow={Math.round(current / 1000)}
      aria-valuetext={clock(current)}
      data-testid="player-seek"
      className="relative flex h-6 min-w-0 flex-1 cursor-pointer touch-none items-center gap-0.5 rounded-xs outline-none focus-visible:ring-[3px] focus-visible:ring-black/10"
      onPointerDown={(event) => {
        event.currentTarget.setPointerCapture(event.pointerId);
        dragging.current = true;
        seekTo(event);
      }}
      onPointerMove={(event) => {
        if (dragging.current) seekTo(event);
      }}
      onPointerUp={(event) => {
        dragging.current = false;
        event.currentTarget.releasePointerCapture(event.pointerId);
      }}
      onPointerCancel={() => {
        dragging.current = false;
      }}
      onKeyDown={onKeyDown}
    >
      {segments.length > 0 ? (
        segments.map((segment) => (
          <span
            key={segment.key}
            className={cn("h-1 min-w-px rounded-[2px]", segment.tone, segment.self ? "opacity-45" : "opacity-60")}
            style={{ flex: segment.size }}
          />
        ))
      ) : (
        <span className="h-1 flex-1 rounded-[2px] bg-line" />
      )}
      <span
        aria-hidden
        className="absolute top-1 -ml-px h-4 w-0.5 rounded-[1px] bg-foreground"
        style={{ left: `${ratio * 100}%` }}
      />
    </div>
  );
}
