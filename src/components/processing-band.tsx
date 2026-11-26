import { Check } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { Spark } from "@/components/empty";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Meeting, Progress, Step } from "@/mygo";
import { cn } from "@/lib/utils";

export type Stage = Exclude<Step, "queued">;

type Phase = Stage | "queued" | "done" | "failed";

export const stages: Record<Stage, { label: string; doing: string; weight: number; until: number }> = {
  import: { label: "导入", doing: "正在导入", weight: 8, until: 5 },
  transcribe: { label: "转写", doing: "正在转写", weight: 52, until: 60 },
  speakers: { label: "识别说话人", doing: "正在识别说话人", weight: 24, until: 84 },
  summary: { label: "生成纪要", doing: "正在生成纪要", weight: 16, until: 100 },
};

const celebration = 1600;
const gap = 8;
const ease = [0.4, 0, 0.2, 1] as const;

export function stagesOf(meeting: Meeting): Stage[] {
  return !meeting.microphone && !meeting.system
    ? ["import", "transcribe", "speakers", "summary"]
    : ["transcribe", "speakers", "summary"];
}

export function ProcessingBand({
  meeting,
  progress,
  onRetry,
}: {
  meeting: Meeting;
  progress: Progress | null;
  onRetry: () => void;
}) {
  const [status, setStatus] = useState(meeting.status);
  const [celebrating, setCelebrating] = useState(false);
  if (status !== meeting.status) {
    setStatus(meeting.status);
    setCelebrating(status === "processing" && meeting.status === "done");
  }

  useEffect(() => {
    if (!celebrating) return;
    const timer = window.setTimeout(() => setCelebrating(false), celebration);
    return () => window.clearTimeout(timer);
  }, [celebrating]);

  const steps = stagesOf(meeting);
  const phase: Phase | null = celebrating
    ? "done"
    : meeting.status === "failed"
      ? "failed"
      : meeting.status === "processing"
        ? (progress?.step ?? steps[0] ?? "transcribe")
        : null;

  return (
    <AnimatePresence initial={false}>
      {phase && (
        <motion.div
          key="band"
          initial={{ height: 0, opacity: 0, marginBottom: -gap }}
          animate={{ height: "auto", opacity: 1, marginBottom: 0 }}
          exit={{ height: 0, opacity: 0, marginBottom: -gap }}
          transition={{ duration: 0.45, ease }}
          className="shrink-0 overflow-hidden rounded-[12px]"
        >
          <Band
            phase={phase}
            steps={steps}
            progress={progress}
            reason={meeting.error}
            onRetry={onRetry}
          />
        </motion.div>
      )}
    </AnimatePresence>
  );
}

function Band({
  phase,
  steps,
  progress,
  reason,
  onRetry,
}: {
  phase: Phase;
  steps: Stage[];
  progress: Progress | null;
  reason: string;
  onRetry: () => void;
}) {
  const failed = phase === "failed";
  const done = phase === "done";
  const stage = failed ? (progress && progress.step !== "queued" ? progress.step : null) : done || phase === "queued" ? null : phase;
  const at = stage ? steps.indexOf(stage) : -1;
  const running = at >= 0 && !failed;
  const percent = done ? 100 : (progress?.percent ?? 0);

  return (
    <div
      className="card flex h-[52px] items-center gap-2.5 pr-5 pl-[11px]"
      data-testid={failed ? "meeting-failed" : "processing"}
      data-phase={phase}
    >
      <span
        className={cn(
          "relative flex size-[30px] shrink-0 items-center justify-center rounded-[9px] text-spark transition-colors duration-200",
          done ? "bg-foreground" : failed ? "bg-failed-bg" : "bg-hover",
        )}
      >
        <Glyph key={phase} phase={phase} />
      </span>
      <span className="flex min-w-[150px] items-baseline gap-2 whitespace-nowrap">
        <span
          key={phase}
          data-testid={phase === "queued" ? "processing-queued" : "processing-label"}
          className={cn(
            "text-row leading-[1.4] font-semibold",
            running ? "text-shimmer" : "animate-rise",
            failed && "text-destructive",
          )}
        >
          {failed ? "转写失败" : done ? "整理好了" : phase === "queued" ? "待整理" : stages[phase].doing}
        </span>
        {running && (
          <span className="text-meta text-muted-foreground tabular-nums">
            {at + 1} / {steps.length}
          </span>
        )}
        {failed && reason !== "" && (
          <span className="max-w-[360px] truncate text-label text-muted-foreground" title={reason} data-selectable>
            {reason}
          </span>
        )}
      </span>
      <span
        className={cn(
          "mr-1.5 ml-2.5 flex min-w-[120px] flex-1 items-center transition-[gap] duration-[400ms]",
          done ? "gap-0" : "gap-[3px]",
        )}
      >
        {steps.map((step, index) => {
          const from = index === 0 ? 0 : stages[steps[index - 1] ?? step].until;
          const part = Math.max(0, Math.min(1, (percent - from) / (stages[step].until - from)));
          const width = done || (at > index && at >= 0) ? 100 : index === at ? Math.round(part * 100) : 0;
          const active = index === at && running;
          return (
            <Tooltip key={step}>
              <TooltipTrigger asChild>
                <span
                  className="relative flex h-5 min-w-0 items-center"
                  style={{ flex: `${stages[step].weight} 1 0` }}
                  data-testid={`processing-${step}`}
                >
                  <span className="relative h-1 w-full overflow-hidden rounded-[2px] bg-rail">
                    <span
                      className={cn(
                        "block h-full rounded-[2px] transition-[width] duration-[800ms] ease-[cubic-bezier(0.22,1,0.36,1)]",
                        failed && index === at ? "bg-destructive" : "bg-foreground",
                      )}
                      style={{ width: `${failed && index === at ? Math.max(width, 12) : width}%` }}
                    />
                    {active && (
                      <span
                        aria-hidden
                        className="absolute inset-y-0 w-16 animate-travel"
                        style={{ background: "linear-gradient(90deg, rgba(13,13,13,0), rgba(13,13,13,0.2), rgba(13,13,13,0))" }}
                      />
                    )}
                  </span>
                  {active && (
                    <Spark
                      className="absolute -top-0.5 -ml-0.5 size-[11px] transition-[left] duration-[800ms] ease-[cubic-bezier(0.22,1,0.36,1)]"
                      style={{ left: `${width}%` }}
                    />
                  )}
                </span>
              </TooltipTrigger>
              <TooltipContent>{stages[step].label}</TooltipContent>
            </Tooltip>
          );
        })}
      </span>
      {running && <Rolling value={percent} />}
      {failed && (
        <Button size="sm" data-testid="reprocess" onClick={onRetry}>
          重试
        </Button>
      )}
    </div>
  );
}

const digit = 18;
const rolled = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

function Rolling({ value }: { value: number }) {
  const columns = [
    { at: Math.floor(value / 100), shown: value >= 100 },
    { at: Math.floor(value / 10) % 10, shown: value >= 10 },
    { at: value % 10, shown: true },
  ];
  return (
    <span
      className="flex h-[18px] min-w-9 items-start justify-end text-ui leading-[18px] font-semibold tabular-nums"
      data-testid="processing-percent"
      data-value={value}
    >
      <span className="sr-only">{value}%</span>
      <span aria-hidden className="flex h-[18px] items-start overflow-hidden">
        {columns.map((column, index) => (
          <span
            key={index}
            className="flex flex-col text-center transition-[transform,width,opacity] duration-[600ms] ease-[cubic-bezier(0.22,1,0.36,1)]"
            style={{
              transform: `translateY(-${column.at * digit}px)`,
              width: column.shown ? "1ch" : "0ch",
              opacity: column.shown ? 1 : 0,
            }}
          >
            {rolled.map((n) => (
              <span key={n} className="h-[18px] shrink-0">
                {n}
              </span>
            ))}
          </span>
        ))}
      </span>
      <span aria-hidden className="ml-px font-medium text-muted-foreground">
        %
      </span>
    </span>
  );
}

const waves = [7, 12, 16, 10, 6];
const lines = [14, 14, 9];
const sparks = [
  [16, -11],
  [18, 8],
  [2, 18],
  [-16, 10],
  [-17, -9],
  [1, -19],
] as const;

function delay(ms: number): CSSProperties {
  return { animationDelay: `${ms}ms` };
}

function Glyph({ phase }: { phase: Phase }) {
  if (phase === "done") {
    return (
      <>
        <Check className="size-4 animate-pop text-white" strokeWidth={3} />
        {sparks.map(([dx, dy], index) => (
          <span
            key={index}
            aria-hidden
            className={cn(
              "absolute top-[13px] left-[13px] animate-burst rounded-full",
              index % 2 === 0 ? "size-[4.5px] bg-spark" : "size-[3.5px] bg-foreground",
            )}
            style={{ "--dx": `${dx}px`, "--dy": `${dy}px` } as CSSProperties}
          />
        ))}
      </>
    );
  }
  return <span className="flex animate-glyph items-center justify-center">{glyphs[phase]}</span>;
}

const glyphs: Record<Exclude<Phase, "done">, ReactNode> = {
  queued: (
    <span className="flex items-center gap-[3px]">
      {[0, 150, 300].map((ms) => (
        <span key={ms} className="size-[3.5px] animate-breathe rounded-full bg-foreground" style={delay(ms)} />
      ))}
    </span>
  ),
  import: (
    <svg width="16" height="16" viewBox="0 0 18 18" aria-hidden fill="none" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M2.5 11v2.5a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2V11" className="stroke-foreground" />
      <path d="M9 2.5v8M5.8 7.4 9 10.6l3.2-3.2" stroke="currentColor" className="animate-drop" />
    </svg>
  ),
  transcribe: (
    <span className="flex h-4 items-center gap-0.5">
      {waves.map((height, index) => (
        <span
          key={index}
          className={cn("w-0.5 animate-wave rounded-[1px]", index === 2 ? "bg-spark" : "bg-foreground")}
          style={{ height, ...delay(index * 120) }}
        />
      ))}
    </span>
  ),
  speakers: (
    <span className="flex items-center">
      <span className="size-[11px] animate-talk rounded-full bg-foreground" />
      <span className="-ml-[3px] size-[11px] animate-talk rounded-full bg-spark shadow-[0_0_0_1.5px_var(--color-hover)]" style={delay(600)} />
    </span>
  ),
  summary: (
    <>
      <span className="flex w-3.5 flex-col gap-[3px]">
        {lines.map((width, index) => (
          <span
            key={index}
            className="h-0.5 origin-left animate-write rounded-[1px] bg-foreground"
            style={{ width, ...delay(index * 250) }}
          />
        ))}
      </span>
      <Spark className="absolute -top-2 -right-2 size-2" />
    </>
  ),
  failed: (
    <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden className="text-destructive">
      <path d="M12 6.5v7" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" />
      <circle cx="12" cy="17.6" r="1.6" fill="currentColor" />
    </svg>
  ),
};
