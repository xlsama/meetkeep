import { useVirtualizer } from "@tanstack/react-virtual";
import { motion } from "motion/react";
import { useEffect, useRef } from "react";
import { Avatar } from "@/components/avatar";
import { clock } from "@/lib/format";
import { useTranscript, type Group, type Line } from "@/lib/transcript";
import { cn } from "@/lib/utils";

type Variant = "main" | "panel";

const styles = {
  main: {
    gap: 20,
    paddingStart: 28,
    inset: "pr-10 pl-8",
    row: "gap-3",
    avatar: "size-7 text-meta",
    head: "h-7 gap-2",
    name: "text-label",
    stamp: "text-tiny",
    text: "text-lead leading-[1.75]",
    caret: "h-[17px]",
  },
  panel: {
    gap: 16,
    paddingStart: 16,
    inset: "px-4",
    row: "gap-2.5",
    avatar: "size-6 text-micro",
    head: "h-6 gap-[7px]",
    name: "text-meta",
    stamp: "text-micro",
    text: "text-body leading-[1.7]",
    caret: "h-4",
  },
} satisfies Record<Variant, unknown>;

export function Transcript({
  meetingId,
  live,
  variant,
  paddingEnd,
  activeSeq = null,
  revealSeq = null,
  revealKey,
  onSelect,
  className,
}: {
  meetingId: string;
  live: boolean;
  variant: Variant;
  paddingEnd: number;
  activeSeq?: number | null;
  revealSeq?: number | null;
  revealKey?: number;
  onSelect?: (line: Line) => void;
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

  const indexOf = (seq: number | null) =>
    seq === null ? -1 : groups.findIndex((g) => g.lines.some((l) => l.seq === seq));
  const activeIndex = indexOf(activeSeq);
  const revealIndex = indexOf(revealSeq);

  useEffect(() => {
    if (activeIndex >= 0) virtualizer.scrollToIndex(activeIndex, { align: "auto" });
  }, [activeIndex, virtualizer]);

  useEffect(() => {
    if (revealIndex >= 0) virtualizer.scrollToIndex(revealIndex, { align: "center" });
  }, [revealIndex, revealKey, virtualizer]);

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
              <GroupView group={group} variant={variant} activeSeq={activeSeq} onSelect={onSelect} />
            </div>
          );
        })}
      </div>
    </div>
  );
}

function GroupView({
  group,
  variant,
  activeSeq,
  onSelect,
}: {
  group: Group;
  variant: Variant;
  activeSeq: number | null;
  onSelect: ((line: Line) => void) | undefined;
}) {
  const style = styles[variant];
  return (
    <div className={cn("flex", style.row)}>
      <Avatar name={group.voice.name} tone={group.voice.avatar} className={style.avatar} />
      <div className="flex min-w-0 flex-1 flex-col">
        <div className={cn("flex items-center", style.head)}>
          <span className={cn("text-subtle", style.name)}>{group.voice.name}</span>
          <span className={cn("text-muted-foreground tabular-nums", style.stamp)}>
            {clock(group.startMs)}
          </span>
          {group.uncertain && <Badge>存疑</Badge>}
        </div>
        <div data-selectable className={cn("m-0", group.uncertain ? "text-subtle" : "text-prose", style.text)}>
          {group.lines.map((line) => (
            <LineView
              key={line.seq}
              line={line}
              caret={style.caret}
              active={line.seq === activeSeq}
              onSelect={onSelect}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function Badge({ children }: { children: string }) {
  return (
    <span className="inline-flex h-[18px] shrink-0 items-center self-center rounded-[5px] border border-warning-border bg-warning-bg px-1.5 text-micro font-semibold text-warning">
      {children}
    </span>
  );
}

function LineView({
  line,
  caret,
  active,
  onSelect,
}: {
  line: Line;
  caret: string;
  active: boolean;
  onSelect: ((line: Line) => void) | undefined;
}) {
  if (line.state === "done") {
    return (
      <span
        data-testid={`line-${line.seq}`}
        aria-current={active || undefined}
        onClick={
          onSelect &&
          (() => {
            if (window.getSelection()?.isCollapsed === false) return;
            onSelect(line);
          })
        }
        className={cn(
          "-mx-1.5 block rounded-[6px] px-1.5 transition-colors",
          onSelect && "cursor-pointer",
          active && "bg-hover",
        )}
      >
        {line.text}
      </span>
    );
  }
  if (line.state === "failed") {
    return (
      <span className="flex items-baseline gap-2 py-0.5">
        <Badge>转写失败</Badge>
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
