import { ArrowDown } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { Notice } from "@/components/notice";
import type { NotesState } from "@/mygo";
import { clock } from "@/lib/format";
import { todoMeta, type Jump } from "@/lib/notes";
import type { Line } from "@/lib/transcript";
import { cn } from "@/lib/utils";

type OnJump = (jump: Jump) => void;

const enter = {
  initial: { opacity: 0, y: 4 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.22, ease: "easeOut" },
} as const;

const fresh = {
  initial: { opacity: 0, y: 4, backgroundColor: "rgba(253, 238, 181, 0.9)" },
  animate: { opacity: 1, y: 0, backgroundColor: "rgba(253, 238, 181, 0)" },
  transition: { duration: 0.22, ease: "easeOut", backgroundColor: { duration: 7, ease: "easeOut" } },
} as const;

const near = 48;
const heard = 3;

function picked(onJump: OnJump, jump: Jump) {
  return () => {
    if (window.getSelection()?.isCollapsed === false) return;
    onJump(jump);
  };
}

function Stamp({ atMs, seq, onJump }: { atMs: number; seq: number; onJump: OnJump }) {
  return (
    <button
      type="button"
      aria-label={`跳到 ${clock(atMs)}`}
      data-testid="notes-timestamp"
      onClick={(event) => {
        event.stopPropagation();
        onJump({ seq, atMs });
      }}
      className="notes-time"
    >
      {clock(atMs)}
    </button>
  );
}

export function PanelNotes({ state, lines, onJump }: { state: NotesState | null; lines: Line[]; onJump: OnJump }) {
  const notes = state?.notes ?? null;
  const topics = notes?.topics ?? [];
  const scroller = useRef<HTMLDivElement>(null);
  const [atEnd, setAtEnd] = useState(true);
  const written = topics.reduce((sum, topic) => sum + topic.points.length, topics.length);

  useEffect(() => {
    const box = scroller.current;
    if (box && atEnd) box.scrollTo({ top: box.scrollHeight, behavior: "smooth" });
  }, [written, atEnd]);

  const latest = lines.filter((line) => line.state === "done" && line.text.trim() !== "").slice(-heard);

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <div
        ref={scroller}
        data-testid="notes"
        data-selectable
        onScroll={(event) => {
          const box = event.currentTarget;
          setAtEnd(box.scrollHeight - box.scrollTop - box.clientHeight < near);
        }}
        className="flex min-h-0 flex-1 flex-col gap-[18px] overflow-y-auto overscroll-contain p-[18px]"
      >
        <Notice testId="notes-error" message={state?.error || null} />
        {topics.length === 0 && (
          <div className="flex flex-1 flex-col justify-end gap-1.5" data-testid="notes-listening">
            {latest.map((line) => (
              <p key={line.seq} className="m-0 text-row leading-[1.65] text-placeholder">
                {line.text}
              </p>
            ))}
            <span className="mt-1 inline-flex items-center gap-[7px] text-label text-muted-foreground">
              <span aria-hidden className="size-[5px] animate-pulse-dot rounded-full bg-success" />
              正在听，说到第一件事就开始记
            </span>
          </div>
        )}
        {notes && notes.todos.length > 0 && (
          <section className="flex shrink-0 flex-col gap-1">
            <h4 className="m-0 mb-0.5 text-meta font-medium tracking-[0.02em] text-muted-foreground">待办</h4>
            <AnimatePresence initial={false}>
              {notes.todos.map((todo, index) => {
                const meta = todoMeta(todo);
                return (
                  <motion.div
                    key={index}
                    {...enter}
                    data-testid={`notes-todo-${index}`}
                    onClick={picked(onJump, todo)}
                    className="cursor-pointer text-row leading-[1.65] text-prose"
                  >
                    {todo.text}
                    {meta !== "" && (
                      <>
                        {" "}
                        <span className="notes-chip">{meta}</span>
                      </>
                    )}
                  </motion.div>
                );
              })}
            </AnimatePresence>
          </section>
        )}
        <AnimatePresence initial={false}>
          {topics.map((topic) => (
            <motion.section
              key={topic.id}
              {...enter}
              data-testid={`notes-topic-${topic.id}`}
              className="flex shrink-0 flex-col gap-1"
            >
              <div className="flex items-baseline gap-2">
                <h3 className="m-0 text-[14.5px] leading-[1.45] font-semibold text-balance">{topic.title}</h3>
                <Stamp atMs={topic.atMs} seq={topic.points[0]?.seq ?? -1} onJump={onJump} />
                {topic.updating && (
                  <span className="ml-auto inline-flex shrink-0 items-center gap-[5px] self-center text-tiny text-muted-foreground">
                    <span aria-hidden className="size-[5px] animate-pulse-dot rounded-full bg-success" />
                    正在谈
                  </span>
                )}
              </div>
              <ul className="m-0 flex list-disc flex-col gap-0.5 p-0 pl-3.5">
                <AnimatePresence initial={false}>
                  {topic.points.map((point, i) => (
                    <motion.li
                      key={i}
                      {...fresh}
                      data-testid={`notes-point-${topic.id}-${i}`}
                      onClick={picked(onJump, point)}
                      className={cn(
                        "-mx-1 cursor-pointer rounded-[5px] px-1 text-row leading-[1.65] marker:text-faint",
                        point.uncertain ? "text-subtle" : "text-prose",
                      )}
                    >
                      {point.text}
                      {point.uncertain && (
                        <>
                          {" "}
                          <span className="notes-doubt">存疑</span>
                        </>
                      )}
                    </motion.li>
                  ))}
                </AnimatePresence>
              </ul>
            </motion.section>
          ))}
        </AnimatePresence>
      </div>
      <AnimatePresence>
        {!atEnd && (
          <motion.button
            type="button"
            data-testid="notes-latest"
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.15 }}
            onClick={() => scroller.current?.scrollTo({ top: scroller.current.scrollHeight, behavior: "smooth" })}
            className="absolute bottom-[46px] left-1/2 inline-flex h-7 -translate-x-1/2 items-center gap-1 rounded-full bg-primary pr-3 pl-2.5 text-label font-medium text-primary-foreground shadow-[0_6px_18px_rgba(0,0,0,0.2)] outline-none"
          >
            <ArrowDown className="size-3.5" />
            回到最新
          </motion.button>
        )}
      </AnimatePresence>
      {notes && (
        <div className="shrink-0 border-t border-divider px-[18px] py-[9px] text-meta text-muted-foreground tabular-nums">
          整理到 {clock(notes.coveredMs)}
        </div>
      )}
    </div>
  );
}
