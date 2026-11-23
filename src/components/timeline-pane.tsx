import { useEffect, useRef } from "react";
import { usePlayback } from "@/components/player";
import type { Chapter } from "@/mygo";
import { clock } from "@/lib/format";
import type { Jump } from "@/lib/notes";
import { cn } from "@/lib/utils";

export function TimelinePane({ chapters, onJump }: { chapters: Chapter[]; onJump: (jump: Jump) => void }) {
  const { currentMs, started } = usePlayback();
  const current = started ? chapters.findLastIndex((chapter) => chapter.atMs <= currentMs) : -1;
  const list = useRef<HTMLOListElement>(null);

  useEffect(() => {
    list.current?.querySelector("[aria-current]")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [current]);

  return (
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-3.5" data-testid="timeline-pane">
      <ol ref={list} className="m-0 flex list-none flex-col gap-0.5 p-0">
        {chapters.map((chapter, index) => (
          <li key={chapter.seq}>
            <button
              type="button"
              aria-current={index === current || undefined}
              data-testid={`timeline-chapter-${index}`}
              onClick={() => onJump({ seq: chapter.seq, atMs: chapter.atMs, through: true })}
              className={cn(
                "flex w-full cursor-pointer gap-3 rounded-[10px] px-3 py-2.5 text-left transition-colors outline-none hover:bg-hover focus-visible:bg-hover",
                index === current && "bg-hover",
              )}
            >
              <span className="min-w-10 shrink-0 pt-[3px] text-label text-muted-foreground tabular-nums">{clock(chapter.atMs)}</span>
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="text-body leading-[1.55] font-semibold text-foreground">{chapter.title}</span>
                {chapter.text !== "" && <span className="text-body leading-[1.7] text-prose">{chapter.text}</span>}
              </span>
            </button>
          </li>
        ))}
      </ol>
    </div>
  );
}
