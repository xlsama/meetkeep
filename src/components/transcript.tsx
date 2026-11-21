import { useVirtualizer } from "@tanstack/react-virtual";
import { motion } from "motion/react";
import { useEffect, useRef } from "react";
import { clock, speaker } from "@/lib/format";
import { useTranscript, type Group, type Line } from "@/lib/transcript";
import { cn } from "@/lib/utils";

type Variant = "main" | "panel";

const styles = {
  main: {
    gap: 20,
    paddingStart: 28,
    inset: "pr-10 pl-8",
    group: "gap-[3px]",
    head: "gap-2",
    name: "text-label",
    stamp: "text-tiny",
    text: "pl-3.5 text-lead leading-[1.75]",
    caret: "h-[17px]",
  },
  panel: {
    gap: 16,
    paddingStart: 16,
    inset: "px-[18px]",
    group: "gap-0.5",
    head: "gap-[7px]",
    name: "text-meta",
    stamp: "text-micro",
    text: "pl-[13px] text-body leading-[1.7]",
    caret: "h-4",
  },
} satisfies Record<Variant, unknown>;

export function Transcript({
  meetingId,
  live,
  variant,
  paddingEnd,
  className,
}: {
  meetingId: string;
  live: boolean;
  variant: Variant;
  paddingEnd: number;
  className?: string;
}) {
  "use no memo";
  const { groups } = useTranscript(meetingId, live);
  const scrollRef = useRef<HTMLDivElement>(null);
  const style = styles[variant];

  const virtualizer = useVirtualizer({
    count: groups.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 84,
    getItemKey: (index) => groups[index]?.key ?? index,
    gap: style.gap,
    paddingStart: style.paddingStart,
    paddingEnd,
    overscan: 6,
    ...(live && { anchorTo: "end" as const, followOnAppend: true, scrollEndThreshold: 80 }),
  });

  const initialized = useRef(false);
  useEffect(() => {
    if (!live || initialized.current || groups.length === 0) return;
    initialized.current = true;
    virtualizer.scrollToEnd();
  }, [live, groups.length, virtualizer]);

  return (
    <div
      ref={scrollRef}
      data-testid="transcript"
      className={cn("min-h-0 flex-1 overflow-y-auto overscroll-contain", className)}
    >
      <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => {
          const group = groups[item.index];
          if (!group) return null;
          return (
            <div
              key={item.key}
              ref={virtualizer.measureElement}
              data-index={item.index}
              className={cn("absolute top-0 left-0 w-full", style.inset)}
              style={{ transform: `translateY(${item.start}px)` }}
            >
              <GroupView group={group} variant={variant} />
            </div>
          );
        })}
      </div>
    </div>
  );
}

function GroupView({ group, variant }: { group: Group; variant: Variant }) {
  const style = styles[variant];
  return (
    <div className={cn("flex flex-col", style.group)}>
      <div className={cn("flex items-center", style.head)}>
        <span
          aria-hidden
          className={cn("size-1.5 shrink-0 rounded-full", group.track === "mic" ? "bg-foreground" : "bg-them")}
        />
        <span className={cn("font-semibold text-foreground", style.name)}>{speaker[group.track]}</span>
        <span className={cn("font-mono text-muted-foreground tabular-nums", style.stamp)}>{clock(group.startMs)}</span>
      </div>
      <div data-selectable className={cn("m-0 text-prose", style.text)}>
        {group.lines.map((line) => (
          <LineView key={line.seq} line={line} caret={style.caret} />
        ))}
      </div>
    </div>
  );
}

function LineView({ line, caret }: { line: Line; caret: string }) {
  if (line.state === "done") return <span className="block">{line.text}</span>;
  if (line.state === "failed") {
    return (
      <span className="flex items-baseline gap-2 py-0.5">
        <span className="inline-flex h-[18px] shrink-0 items-center self-center rounded-[5px] border border-warning-border bg-warning-bg px-1.5 text-micro font-semibold text-warning">
          转写失败
        </span>
        <span className="min-w-0 text-label leading-normal break-words text-destructive">{line.error}</span>
      </span>
    );
  }
  if (line.state === "transcribing") {
    return (
      <span className="block">
        <span
          aria-hidden
          className={cn("inline-block w-0.5 animate-caret bg-foreground align-[-3px]", caret)}
        />
      </span>
    );
  }
  return (
    <span aria-hidden className={cn("flex items-center gap-1", caret)}>
      {[0, 1, 2].map((index) => (
        <motion.span
          key={index}
          className="size-[5px] rounded-full bg-muted-foreground"
          animate={{ opacity: [0.25, 1, 0.25] }}
          transition={{ duration: 1.2, repeat: Infinity, ease: "easeInOut", delay: index * 0.18 }}
        />
      ))}
    </span>
  );
}
