import { useEffect, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Mic, Volume2, type LucideIcon } from "lucide-react";
import type { Utterance } from "./mygo";
import { useStore, type Source } from "./store";

export function App() {
  const fault = useStore((s) => s.fault);
  return (
    <div className="flex h-full flex-col">
      <header className="flex h-14 shrink-0 items-center gap-3 border-b border-line px-4">
        <RecordButton />
        <Elapsed />
        <div className="ml-auto flex items-center gap-2">
          <SourceToggle source="mic" icon={Mic} label="麦克风" />
          <SourceToggle source="system" icon={Volume2} label="系统音频" />
        </div>
      </header>
      <Transcript />
      {fault && (
        <p data-testid="fault" className="shrink-0 border-t border-line px-4 py-2 text-sm text-record select-text">
          {fault}
        </p>
      )}
    </div>
  );
}

function RecordButton() {
  const running = useStore((s) => s.running);
  const enabled = useStore((s) => s.sources.mic || s.sources.system);
  const start = useStore((s) => s.start);
  const stop = useStore((s) => s.stop);
  return (
    <button
      type="button"
      data-testid="record-toggle"
      aria-label={running ? "停止" : "开始录制"}
      aria-pressed={running}
      disabled={!enabled}
      onClick={() => void (running ? stop() : start())}
      className="grid size-8 place-items-center rounded-full bg-raised transition-transform active:scale-95 disabled:opacity-40"
    >
      <span
        className={
          running ? "size-3 rounded-[3px] bg-record transition-all" : "size-3.5 rounded-full bg-record transition-all"
        }
      />
    </button>
  );
}

function Elapsed() {
  const running = useStore((s) => s.running);
  const startedAt = useStore((s) => s.startedAt);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(timer);
  }, [running]);
  if (!running) return null;
  return (
    <span data-testid="elapsed" className="text-sm text-muted tabular-nums">
      {clock(Math.max(now - startedAt, 0))}
    </span>
  );
}

function SourceToggle({ source, icon: Icon, label }: { source: Source; icon: LucideIcon; label: string }) {
  const on = useStore((s) => s.sources[source]);
  const running = useStore((s) => s.running);
  const level = useStore((s) => s.levels[source]);
  const toggle = useStore((s) => s.toggleSource);
  return (
    <button
      type="button"
      data-testid={`source-${source}`}
      aria-label={label}
      aria-pressed={on}
      disabled={running}
      onClick={() => toggle(source)}
      className={`flex h-8 items-center gap-2 rounded-full px-3 transition-colors ${on ? "bg-raised text-ink" : "text-muted opacity-50"}`}
    >
      <Icon className="size-4" strokeWidth={1.75} />
      <span className="flex h-4 w-[3px] items-end overflow-hidden rounded-full bg-line">
        <span
          className={`w-full rounded-full transition-[height] duration-100 ${source === "mic" ? "bg-me" : "bg-other"}`}
          style={{ height: `${Math.round((on ? level : 0) * 100)}%` }}
        />
      </span>
    </button>
  );
}

function Transcript() {
  const utterances = useStore((s) => s.utterances);
  const parent = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: utterances.length,
    getScrollElement: () => parent.current,
    estimateSize: () => 44,
    getItemKey: (index) => utterances[index]!.id,
    anchorTo: "end",
    followOnAppend: true,
    scrollEndThreshold: 80,
    overscan: 8,
  });
  return (
    <div ref={parent} data-testid="transcript" className="min-h-0 flex-1 overflow-y-auto py-3">
      <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => (
          <div
            key={item.key}
            ref={virtualizer.measureElement}
            data-index={item.index}
            className="absolute top-0 left-0 w-full"
            style={{ transform: `translateY(${item.start}px)` }}
          >
            <Row utterance={utterances[item.index]!} />
          </div>
        ))}
      </div>
    </div>
  );
}

function Row({ utterance }: { utterance: Utterance }) {
  const mine = utterance.source === "mic";
  return (
    <div data-testid="utterance" data-state={utterance.state} className="grid grid-cols-[3rem_2.5rem_1fr] gap-x-2 px-4 py-2">
      <span className="pt-0.5 text-xs text-muted tabular-nums">{clock(utterance.startMs)}</span>
      <span className={`pt-px text-sm font-medium ${mine ? "text-me" : "text-other"}`}>{mine ? "我" : "对方"}</span>
      {utterance.state === "speaking" || utterance.state === "transcribing" ? (
        <Dots dim={utterance.state === "transcribing"} />
      ) : (
        <p
          className={`text-[15px] leading-6 select-text ${utterance.state === "failed" ? "text-record" : "text-ink"}`}
        >
          {utterance.text}
        </p>
      )}
    </div>
  );
}

function Dots({ dim }: { dim: boolean }) {
  return (
    <span className={`flex h-6 items-center gap-1 ${dim ? "opacity-50" : ""}`}>
      {[0, 1, 2].map((i) => (
        <span key={i} className="size-1.5 animate-dot rounded-full bg-muted" style={{ animationDelay: `${i * 0.15}s` }} />
      ))}
    </span>
  );
}

function clock(ms: number) {
  const seconds = Math.floor(ms / 1000);
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
}
