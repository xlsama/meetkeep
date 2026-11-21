import { AnimatePresence, motion } from "motion/react";
import type { ReactNode } from "react";
import { Notice } from "@/components/notice";
import type { Notes, NotesState, Point, Todo, Topic } from "@/mygo";
import { clock } from "@/lib/format";
import { ordinal, todoMeta, type Jump } from "@/lib/notes";
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

export function Stamp({ atMs, seq, onJump }: { atMs: number; seq: number; onJump: OnJump }) {
  return (
    <button
      type="button"
      aria-label={`跳到 ${clock(atMs)}`}
      onClick={(event) => {
        event.stopPropagation();
        onJump({ seq, atMs });
      }}
      className="ml-1.5 inline rounded-[4px] px-0.5 align-[1px] font-mono text-tiny text-muted-foreground tabular-nums transition-colors hover:bg-hover hover:text-foreground"
    >
      {clock(atMs)}
    </button>
  );
}

function Doubt({ className }: { className: string }) {
  return (
    <span
      className={cn(
        "inline-flex h-[18px] items-center rounded-[5px] border border-warning-border bg-warning-bg px-1.5 align-[1px] text-micro font-semibold text-warning",
        className,
      )}
    >
      存疑
    </span>
  );
}

function Updating({ panel }: { panel?: boolean }) {
  return (
    <span
      className={cn(
        "inline-flex items-center text-muted-foreground",
        panel ? "ml-auto gap-[5px] text-tiny" : "gap-1.5 text-meta",
      )}
    >
      <span aria-hidden className={cn("animate-pulse-dot rounded-full bg-success", panel ? "size-[5px]" : "size-1.5")} />
      更新中
    </span>
  );
}

function Heading({ children }: { children: ReactNode }) {
  return <h4 className="m-0 mb-1.5 text-label font-semibold text-muted-foreground">{children}</h4>;
}

export function NotesPane({
  state,
  live,
  onJump,
  onRetry,
}: {
  state: NotesState | null;
  live: boolean;
  onJump: OnJump;
  onRetry?: () => void;
}) {
  const notes = state?.notes ?? null;
  return (
    <div
      data-testid="notes"
      data-selectable
      className={cn("min-h-0 flex-1 overflow-y-auto overscroll-contain px-7 pt-6", live ? "pb-[110px]" : "pb-6")}
    >
      <Notice
        testId="notes-error"
        message={state?.error || null}
        className="mb-5"
        action={
          onRetry && (
            <button
              type="button"
              data-testid="notes-retry"
              onClick={onRetry}
              className="-my-0.5 inline-flex h-[22px] shrink-0 items-center rounded-xs px-2 text-label font-semibold text-warning transition-colors hover:bg-warning-border/50"
            >
              重试
            </button>
          )
        }
      />
      {notes && (notes.final ? <FinalNotes notes={notes} onJump={onJump} /> : <DraftNotes notes={notes} onJump={onJump} />)}
    </div>
  );
}

function DraftNotes({ notes, onJump }: { notes: Notes; onJump: OnJump }) {
  return (
    <div className="flex flex-col gap-6">
      <Topics topics={notes.topics} onJump={onJump} draft />
      {notes.todos.length > 0 && (
        <section className="flex flex-col gap-0.5 pt-1">
          <Heading>待办</Heading>
          <Todos todos={notes.todos} onJump={onJump} />
        </section>
      )}
    </div>
  );
}

function FinalNotes({ notes, onJump }: { notes: Notes; onJump: OnJump }) {
  return (
    <div className="flex flex-col gap-[22px]">
      {notes.headline !== "" && (
        <h2 className="m-0 text-[19px] leading-[1.45] font-semibold tracking-[-0.01em]">{notes.headline}</h2>
      )}
      {notes.todos.length > 0 && (
        <section className="flex flex-col gap-0.5">
          <Heading>待办</Heading>
          <Todos todos={notes.todos} onJump={onJump} />
        </section>
      )}
      {notes.open.length > 0 && (
        <section className="flex flex-col gap-1">
          <h4 className="m-0 mb-1 text-label font-semibold text-muted-foreground">还没定</h4>
          {notes.open.map((point, index) => (
            <p
              key={index}
              data-testid={`notes-open-${index}`}
              onClick={picked(onJump, point)}
              className="m-0 cursor-pointer text-body leading-[1.7]"
            >
              {point.text}
              <Stamp atMs={point.atMs} seq={point.seq} onJump={onJump} />
            </p>
          ))}
        </section>
      )}
      <Topics topics={notes.topics} onJump={onJump} draft={false} />
    </div>
  );
}

function Topics({ topics, onJump, draft }: { topics: Topic[]; onJump: OnJump; draft: boolean }) {
  return (
    <AnimatePresence initial={false}>
      {topics.map((topic, index) => (
        <motion.section
          key={topic.id}
          {...enter}
          data-testid={`notes-topic-${topic.id}`}
          className="flex flex-col gap-2"
        >
          <div className="flex items-baseline gap-2.5">
            <span className="w-5 shrink-0 font-mono text-meta text-muted-foreground">{ordinal(index)}</span>
            <h3 className="m-0 text-title leading-[1.4] font-semibold">{topic.title}</h3>
            {draft && topic.updating && <Updating />}
          </div>
          <ul className="m-0 flex list-none flex-col gap-1 p-0 pl-[30px]">
            <AnimatePresence initial={false}>
              {topic.points.map((point, i) => (
                <PointItem key={i} topicId={topic.id} index={i} point={point} onJump={onJump} />
              ))}
            </AnimatePresence>
          </ul>
        </motion.section>
      ))}
    </AnimatePresence>
  );
}

function PointItem({ topicId, index, point, onJump }: { topicId: string; index: number; point: Point; onJump: OnJump }) {
  return (
    <motion.li
      {...enter}
      data-testid={`notes-point-${topicId}-${index}`}
      onClick={picked(onJump, point)}
      className={cn("cursor-pointer text-body leading-[1.7]", point.uncertain ? "text-subtle" : "text-prose")}
    >
      {point.uncertain ? (
        <>
          <span className="underline decoration-faint decoration-dotted underline-offset-4">{point.text}</span>
          <Doubt className="ml-2" />
        </>
      ) : (
        point.text
      )}
      <Stamp atMs={point.atMs} seq={point.seq} onJump={onJump} />
    </motion.li>
  );
}

function Todos({ todos, onJump }: { todos: Todo[]; onJump: OnJump }) {
  return (
    <AnimatePresence initial={false}>
      {todos.map((todo, index) => {
        const meta = todoMeta(todo);
        return (
          <motion.div
            key={index}
            {...enter}
            data-testid={`notes-todo-${index}`}
            onClick={picked(onJump, todo)}
            className="flex min-h-8 cursor-pointer items-center gap-2.5"
          >
            <span className="text-body">{todo.text}</span>
            {meta !== "" && <span className="text-label text-muted-foreground">{meta}</span>}
            <Stamp atMs={todo.atMs} seq={todo.seq} onJump={onJump} />
          </motion.div>
        );
      })}
    </AnimatePresence>
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
                  <div className="flex items-center gap-2">
                    <h3 className="m-0 text-[14.5px] leading-[1.45] font-semibold">{topic.title}</h3>
                    <Stamp atMs={topic.atMs} seq={topic.points[0]?.seq ?? -1} onJump={onJump} />
                    {topic.updating && <Updating panel />}
                  </div>
                  <ul className="m-0 flex list-disc flex-col gap-0.5 p-0 pl-3.5 text-placeholder">
                    <AnimatePresence initial={false}>
                      {topic.points.map((point, i) => (
                        <motion.li
                          key={i}
                          {...enter}
                          data-testid={`notes-point-${topic.id}-${i}`}
                          onClick={picked(onJump, point)}
                          className={cn(
                            "cursor-pointer text-row leading-[1.65]",
                            point.uncertain ? "text-subtle" : "text-prose",
                          )}
                        >
                          {point.text}
                          {point.uncertain && <Doubt className="ml-1.5" />}
                        </motion.li>
                      ))}
                    </AnimatePresence>
                  </ul>
                </motion.section>
              ))}
            </AnimatePresence>
            {notes.todos.length > 0 && (
              <section className="flex shrink-0 flex-col">
                <h4 className="m-0 mb-1 text-meta font-semibold text-muted-foreground">待办</h4>
                <AnimatePresence initial={false}>
                  {notes.todos.map((todo, index) => {
                    const meta = todoMeta(todo);
                    return (
                      <motion.div
                        key={index}
                        {...enter}
                        data-testid={`notes-todo-${index}`}
                        onClick={picked(onJump, todo)}
                        className="flex min-h-[30px] cursor-pointer items-center gap-[9px]"
                      >
                        <span className="text-row">{todo.text}</span>
                        {meta !== "" && <span className="text-meta text-muted-foreground">{meta}</span>}
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
