import {
  defineTextTrack,
  Hotkey,
  isMediaSeekCapable,
  selectPlayback,
  selectPlaybackRate,
  selectTime,
  TimeSlider,
  useCreateTextTrack,
  useLatestRef,
  useMedia,
  usePlayer,
  type TextCueInit,
} from "@videojs/react";
import { Audio, AudioPlayer } from "@videojs/react/audio";
import { Video, VideoPlayer } from "@videojs/react/video";
import { ChevronDown, ChevronUp } from "lucide-react";
import { useCallback, useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { AudioSkin } from "@/components/videojs/audio/skin";
import { Button as MediaButton } from "@/components/videojs/ui/button";
import { ButtonTooltip } from "@/components/videojs/ui/button-tooltip";
import { VideoSkin } from "@/components/videojs/video/skin";
import { bindMain, stopSample } from "@/lib/sample";
import { segmentsOf } from "@/lib/segments";
import type { Group } from "@/lib/transcript";
import { cn } from "@/lib/utils";

const rates = [5, 3, 2, 1.5, 1.25, 1];
const collapsedKey = "meetkeep:video-collapsed";
const speakers = defineTextTrack({ kind: "metadata", label: "speakers" });

const skinTokens = {
  "--media-font-family": "var(--font-sans)",
  "--media-controls-before-start": "visible",
} as CSSProperties;

const videoTokens = { ...skinTokens, "--media-border-radius": "var(--radius-lg)" } as CSSProperties;

const audioTokens = {
  ...skinTokens,
  "--media-accent-color": "var(--color-primary)",
  "--media-accent-text-color": "var(--color-primary-foreground)",
  "--media-border-color": "var(--color-border)",
  "--media-controls": "var(--color-background)",
  "--media-controls-radius": "8px",
  "--media-surface-border": "transparent",
  "--media-shadow-surface": "none",
  "--media-shadow-surface-inset": "none",
} as CSSProperties;

export function PlayerProvider({ video, children }: { video: boolean; children: ReactNode }) {
  return video ? (
    <VideoPlayer defaultPlaybackRates={rates}>
      <SampleBridge />
      {children}
    </VideoPlayer>
  ) : (
    <AudioPlayer defaultPlaybackRates={rates}>
      <SampleBridge />
      {children}
    </AudioPlayer>
  );
}

function SampleBridge() {
  const playback = usePlayer(selectPlayback);
  const pause = playback?.pause;
  const paused = playback?.paused ?? true;

  useEffect(() => (pause ? bindMain(pause) : undefined), [pause]);
  useEffect(() => {
    if (!paused) stopSample();
  }, [paused]);

  return null;
}

export function usePlayback() {
  const time = usePlayer(selectTime);
  const playback = usePlayer(selectPlayback);
  const rate = usePlayer(selectPlaybackRate)?.playbackRate ?? 1;
  const media = useMedia();
  const latest = useLatestRef({ time, playback });
  const now = useCallback(() => Math.round((isMediaSeekCapable(media) ? media.currentTime : 0) * 1000), [media]);
  const seek = useCallback(
    (ms: number, play = false) => {
      void latest.current.time?.seek(ms / 1000);
      if (play) latest.current.playback?.play().catch(() => undefined);
    },
    [latest],
  );
  return {
    currentMs: (time?.currentTime ?? 0) * 1000,
    ready: (time?.duration ?? 0) > 0,
    started: playback?.started ?? false,
    playing: playback ? !playback.paused : false,
    rate,
    now,
    seek,
  };
}

function SpeakerMarkers({ groups, durationMs }: { groups: Group[]; durationMs: number }) {
  const duration = usePlayer(selectTime)?.duration ?? 0;
  const segments = segmentsOf(groups, duration > 0 ? duration * 1000 : durationMs);
  const cues: TextCueInit[] = segments.map((segment) => ({
    startTime: segment.start / 1000,
    endTime: segment.end / 1000,
  }));
  useCreateTextTrack(speakers, { cues });

  return (
    <TimeSlider.Markers
      filter={speakers}
      className="pointer-events-none absolute inset-x-0 top-1/2 h-1 -translate-y-1/2"
      renderMarker={(props, state) => (
        <div
          {...props}
          className={cn(
            "absolute inset-y-0 left-(--media-slider-marker-start) w-(--media-slider-marker-width) rounded-[2px] border-x border-transparent bg-clip-padding",
            segments[state.index]?.tone,
            segments[state.index]?.self ? "opacity-45" : "opacity-60",
          )}
        />
      )}
    />
  );
}

type StageProps = { meetingId: string; durationMs: number; groups: Group[]; spaceDisabled: boolean };

export function VideoStage({ meetingId, durationMs, groups, spaceDisabled }: StageProps) {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(collapsedKey) === "1");
  const toggle = () => {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(collapsedKey, next ? "1" : "0");
  };
  const label = collapsed ? "展开画面" : "收起画面";

  return (
    <div data-testid="player-video" className="shrink-0 px-8 pt-4">
      <VideoSkin
        style={videoTokens}
        collapsed={collapsed}
        className={collapsed ? "h-14!" : "mx-auto aspect-video h-auto! w-[min(100%,calc(45cqh*16/9))]!"}
        markers={<SpeakerMarkers groups={groups} durationMs={durationMs} />}
        actions={
          <ButtonTooltip side="top" label={label}>
            <MediaButton type="button" aria-label={label} data-testid="player-collapse" onClick={toggle}>
              {collapsed ? <ChevronDown className="size-media-icon" /> : <ChevronUp className="size-media-icon" />}
            </MediaButton>
          </ButtonTooltip>
        }
      >
        <Video
          src={`recording://localhost/${meetingId}/video`}
          preload="metadata"
          playsInline
          fallback={null}
          className={cn(collapsed && "invisible")}
        />
        <Hotkey keys="Space" action="togglePaused" target="document" disabled={spaceDisabled} />
      </VideoSkin>
    </div>
  );
}

export function AudioBar({ meetingId, durationMs, groups, spaceDisabled }: StageProps) {
  return (
    <footer className="shrink-0 border-t border-divider px-1.5 py-1">
      <AudioSkin style={audioTokens} markers={<SpeakerMarkers groups={groups} durationMs={durationMs} />}>
        <Audio src={`recording://localhost/${meetingId}/audio`} preload="metadata" fallback={null} />
        <Hotkey keys="Space" action="togglePaused" target="document" disabled={spaceDisabled} />
      </AudioSkin>
    </footer>
  );
}
