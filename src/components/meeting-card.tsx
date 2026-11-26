import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Avatar } from "@/components/avatar";
import { MeetingMenu, type OnError } from "@/components/meeting-menu";
import { stages } from "@/components/processing-band";
import { RecDot } from "@/components/rec-dot";
import type { Card, Speaker, Status } from "@/mygo";
import { clock, listDate, minutes } from "@/lib/format";
import { useProgress } from "@/lib/progress";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { speakerVoice } from "@/lib/speakers";
import { cn } from "@/lib/utils";

export const cardGrid = "grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-3";

export function MeetingCard({ card, now, onError }: { card: Card; now: Date; onError: OnError }) {
  const { meeting, speakers } = card;

  return (
    <div className="group/meeting relative min-w-0">
      <Link
        to="/meetings/$id"
        params={{ id: meeting.id }}
        data-testid={`meeting-card-${meeting.id}`}
        className="card flex h-[164px] flex-col gap-2 p-4 text-foreground shadow-[0_0_0_0.5px_rgba(0,0,0,0.12),0_1px_2px_rgba(0,0,0,0.04)] transition-[box-shadow,scale] duration-100 ease-out outline-none hover:shadow-[0_0_0_0.5px_rgba(0,0,0,0.3),0_1px_2px_rgba(0,0,0,0.06),0_3px_8px_-2px_rgba(0,0,0,0.08)] active:scale-[0.99] focus-visible:shadow-[0_0_0_3px_rgba(0,0,0,0.12)]"
      >
        <span className="truncate pr-[30px] text-body leading-[1.4] font-semibold">{meeting.title}</span>
        {card.headline ? (
          <p className="m-0 line-clamp-2 text-label leading-[1.6] text-secondary-foreground">{card.headline}</p>
        ) : (
          <p className="m-0 line-clamp-3 text-label leading-[1.6] text-muted-foreground">{card.opening}</p>
        )}
        {card.headline && card.topics.length > 0 && (
          <div className="flex h-[22px] flex-wrap gap-1.5 overflow-hidden">
            {card.topics.map((topic, index) => (
              <span
                key={index}
                className="inline-block h-[22px] max-w-full truncate rounded-xs bg-hover px-2 text-tiny leading-[22px] text-subtle"
              >
                {topic}
              </span>
            ))}
          </div>
        )}
        <div className="mt-auto flex items-center gap-2">
          <Speakers speakers={speakers} />
          <MeetingState card={card} now={now} className="ml-auto" />
        </div>
      </Link>
      <MeetingMenu meeting={meeting} onError={onError} className="absolute top-3 right-2.5" />
    </div>
  );
}

export function Speakers({ speakers }: { speakers: Speaker[] }) {
  const shown = [...speakers.filter((s) => !s.self), ...speakers.filter((s) => s.self)].slice(0, 5);
  return (
    <span className="inline-flex shrink-0">
      {shown.map((speaker, index) => (
        <Avatar
          key={speaker.id}
          name={speaker.name}
          tone={speakerVoice(speaker, speakers).avatar}
          className={cn("size-5 text-[10px] shadow-[0_0_0_2px_#ffffff]", index > 0 && "-ml-1")}
        />
      ))}
    </span>
  );
}

export function MeetingState({ card, now, className }: { card: Card; now: Date; className?: string }) {
  const { meeting } = card;
  const status = useRecorder((s) => s.status);
  const progress = useProgress((s) => s.items[meeting.id]) ?? card.progress;
  if (meeting.status === "recording") {
    return <RecordingLabel status={status.meeting?.id === meeting.id ? status : null} className={className} />;
  }
  if (meeting.status === "processing") {
    return (
      <span
        className={cn("inline-flex items-center gap-1.5 text-meta leading-normal text-muted-foreground tabular-nums", className)}
        data-testid={`meeting-state-${meeting.id}`}
      >
        {listDate(meeting.startedAt, now)}
        {progress?.step === "queued" ? (
          <>
            <span aria-hidden className="flex items-center gap-[2.5px]">
              {[0, 150, 300].map((ms) => (
                <span
                  key={ms}
                  className="size-[3px] animate-breathe rounded-full bg-muted-foreground"
                  style={{ animationDelay: `${ms}ms` }}
                />
              ))}
            </span>
            待整理
          </>
        ) : (
          <>
            <Ring percent={progress?.percent ?? 0} />
            <span className="font-medium text-foreground">
              {progress ? `${stages[progress.step].label} ${progress.percent}%` : "整理中"}
            </span>
          </>
        )}
      </span>
    );
  }
  return (
    <span
      className={cn(
        "text-meta leading-normal tabular-nums",
        meeting.status === "failed" ? "text-destructive" : "text-muted-foreground",
        className,
      )}
    >
      {listDate(meeting.startedAt, now)} · {meeting.status === "failed" ? "转写失败" : minutes(meeting.durationMs)}
    </span>
  );
}

const radius = 5.5;
const around = 2 * Math.PI * radius;

function Ring({ percent }: { percent: number }) {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" aria-hidden className="shrink-0 -rotate-90">
      <circle cx="7" cy="7" r={radius} fill="none" strokeWidth="2.4" className="stroke-border" />
      <circle
        cx="7"
        cy="7"
        r={radius}
        fill="none"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeDasharray={around}
        strokeDashoffset={around * (1 - percent / 100)}
        className="stroke-foreground transition-[stroke-dashoffset] duration-[800ms] ease-out"
      />
    </svg>
  );
}

const idle: Status = { phase: "idle", meeting: null, elapsedMs: 0, at: 0 };

export function RecordingLabel({ status, className }: { status: Status | null; className?: string }) {
  const elapsed = useElapsed(status ?? idle);
  return (
    <span
      className={cn("inline-flex items-center gap-1.5 text-meta leading-normal text-destructive tabular-nums", className)}
    >
      <RecDot paused={status?.phase === "paused"} className="size-1.5" />
      {status ? `录制中 · ${clock(elapsed)}` : "录制中"}
    </span>
  );
}

export function useNow() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
}
