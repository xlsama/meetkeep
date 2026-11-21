import { AnimatePresence, motion } from "motion/react";
import { Notice } from "@/components/notice";
import type { NotesState } from "@/mygo";
import { clock } from "@/lib/format";
import { todoMeta, type Jump } from "@/lib/notes";
import { cn } from "@/lib/utils";

type OnJump = (jump: Jump) => void;

const enter = {
  initial: { opacity: 0, y: 4 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.22, ease: "easeOut" },
} as const;

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

export function PanelNotes({ state, onJump }: { state: NotesState | null; onJump: OnJump }) {
  const notes = state?.notes ?? null;
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div
        data-testid="notes"
        data-selectable
        className="flex min-h-0 flex-1 flex-col gap-[18px] overflow-y-auto overscroll-contain p-[18px]"
      >
        <Notice testId="notes-error" message={state?.error || null} />
        {notes && (
          <>
            <AnimatePresence initial={false}>
              {notes.topics.map((topic) => (
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
                        更新中
                      </span>
                    )}
                  </div>
                  <ul className="m-0 flex list-disc flex-col gap-0.5 p-0 pl-3.5">
                    <AnimatePresence initial={false}>
                      {topic.points.map((point, i) => (
                        <motion.li
                          key={i}
                          {...enter}
                          data-testid={`notes-point-${topic.id}-${i}`}
                          onClick={picked(onJump, point)}
                          className={cn(
                            "cursor-pointer text-row leading-[1.65] marker:text-faint",
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
            {notes.todos.length > 0 && (
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
          </>
        )}
      </div>
      {notes && (
        <div className="shrink-0 border-t border-divider px-[18px] py-[9px] text-meta text-muted-foreground tabular-nums">
          整理到 {clock(notes.coveredMs)}
        </div>
      )}
    </div>
  );
}
