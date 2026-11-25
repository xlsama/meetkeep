import { useVirtualizer } from "@tanstack/react-virtual";
import { motion } from "motion/react";
import {} from "lucide-react";
import { Popover } from "radix-ui";
import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { Avatar } from "@/components/avatar";
import { Highlight } from "@/components/highlight";
import { SpeakerPopover } from "@/components/speaker-popover";
import { Button } from "@/components/ui/button";
import { clock } from "@/lib/format";
import { spaced, useTranscript, type Group, type Line } from "@/lib/transcript";
import { errorMessage } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { wordAt, wordNear, wordsOf, type Word } from "@/lib/words";
import { Library, type Guess } from "@/mygo";

type Variant = "main" | "panel";

export type Playhead = { now: () => number; playing: boolean; ms: number };

// The words fade out over this many pixels as they scroll under the heading.
const fade = 20;

const styles = {
  main: {
    gap: 20,
    paddingStart: 8,
    inset: "pr-10 pl-8",
    row: "gap-3",
    avatar: "size-[23px] text-micro",
    head: "h-[23px] gap-2",
    name: "text-label",
    stamp: "text-tiny",
    text: "text-body leading-[25px]",
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
  query = "",
  findSeq = null,
  playhead,
  onSeek,
  className,
}: {
  meetingId: string;
  live: boolean;
  variant: Variant;
  paddingEnd: number;
  activeSeq?: number | null;
  revealSeq?: number | null;
  revealKey?: number;
  query?: string;
  findSeq?: number | null;
  playhead?: Playhead;
  onSeek?: (ms: number) => void;
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
    gap: style.gap / 2,
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
  const findIndex = indexOf(findSeq);

  useEffect(() => {
    if (activeIndex >= 0) virtualizer.scrollToIndex(activeIndex, { align: "auto" });
  }, [activeIndex, virtualizer]);

  useEffect(() => {
    if (revealIndex >= 0) virtualizer.scrollToIndex(revealIndex, { align: "center" });
  }, [revealIndex, revealKey, virtualizer]);

  useEffect(() => {
    if (findIndex >= 0) virtualizer.scrollToIndex(findIndex, { align: "center" });
  }, [findIndex, findSeq, virtualizer]);

  return (
    <div
      ref={scrollRef}
      data-testid="transcript-lines"
      className={cn(
        "min-h-0 flex-1 overflow-y-auto overscroll-contain",
        variant === "main" && "[mask-image:linear-gradient(to_bottom,transparent,black_var(--fade,0px))]",
        className,
      )}
      onScroll={(event) => {
        const box = event.currentTarget;
        box.style.setProperty("--fade", `${Math.min(box.scrollTop, fade)}px`);
      }}
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
              style={{
                transform: `translateY(${item.start}px)`,
                paddingTop: item.index > 0 && !group.continued ? style.gap / 2 : 0,
              }}
            >
              <GroupView
                group={group}
                meetingId={meetingId}
                variant={variant}
                activeSeq={activeSeq}
                query={query}
                findSeq={findSeq}
                playhead={playhead}
                onSeek={onSeek}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}

function GroupView({
  group,
  meetingId,
  variant,
  activeSeq,
  query,
  findSeq,
  playhead,
  onSeek,
}: {
  group: Group;
  meetingId: string;
  variant: Variant;
  activeSeq: number | null;
  query: string;
  findSeq: number | null;
  playhead: Playhead | undefined;
  onSeek: ((ms: number) => void) | undefined;
}) {
  const style = styles[variant];
  const { voice } = group;
  const [editing, setEditing] = useState(false);
  const avatar = useRef<HTMLButtonElement>(null);
  const speakerId = voice.self ? null : voice.speakerId;

  return (
    <Popover.Root open={editing} onOpenChange={setEditing}>
      <div className={cn("flex", style.row)}>
        {group.continued ? (
          <span aria-hidden className={cn("shrink-0", style.avatar)} />
        ) : speakerId === null ? (
          <Avatar name={voice.name} tone={voice.avatar} className={style.avatar} />
        ) : (
          <button
            ref={avatar}
            type="button"
            tabIndex={-1}
            aria-label={`改名：${voice.name}`}
            data-testid={`speaker-avatar-${group.key}`}
            onClick={() => setEditing((open) => !open)}
            className="shrink-0 cursor-pointer self-start rounded-full outline-none transition-shadow hover:shadow-[0_0_0_3px_rgba(0,0,0,0.08)]"
          >
            <Avatar name={voice.name} tone={voice.avatar} className={style.avatar} />
          </button>
        )}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className={cn("flex items-center", style.head)}>
            {group.continued ? null : speakerId === null ? (
              <span className={cn("text-subtle", style.name)}>{voice.name}</span>
            ) : (
              <Popover.Trigger asChild>
                <button
                  type="button"
                  data-testid={`speaker-name-${group.key}`}
                  className={cn(
                    "-mx-1.5 inline-flex h-6 cursor-pointer items-center rounded-md px-1.5 text-subtle outline-none transition-colors hover:bg-hover hover:text-foreground focus-visible:bg-hover data-[state=open]:bg-hover data-[state=open]:text-foreground",
                    style.name,
                  )}
                >
                  {voice.name}
                </button>
              </Popover.Trigger>
            )}
            <span className={cn("text-muted-foreground tabular-nums", style.stamp)}>{clock(group.startMs)}</span>
            {group.uncertain && <Badge>存疑</Badge>}
          </div>
          <div data-selectable className={cn("m-0", group.uncertain ? "text-subtle" : "text-prose", style.text)}>
            {group.lines.map((line, index) => (
              <LineView
                key={line.seq}
                meetingId={meetingId}
                line={line}
                spaced={spaced(group.lines[index - 1]?.text ?? "")}
                caret={style.caret}
                active={line.seq === activeSeq}
                query={query}
                found={line.seq === findSeq}
                playhead={line.seq === activeSeq ? playhead : undefined}
                onSeek={onSeek}
              />
            ))}
          </div>
        </div>
      </div>
      {editing && speakerId !== null && (
        <SpeakerPopover meetingId={meetingId} speakerId={speakerId} avatar={avatar} onDone={() => setEditing(false)} />
      )}
    </Popover.Root>
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
  meetingId,
  line,
  spaced,
  caret,
  active,
  query,
  found,
  playhead,
  onSeek,
}: {
  meetingId: string;
  line: Line;
  spaced: boolean;
  caret: string;
  active: boolean;
  query: string;
  found: boolean;
  playhead: Playhead | undefined;
  onSeek: ((ms: number) => void) | undefined;
}) {
  if (line.state === "done") {
    return (
      <>
        {spaced && " "}
        <span
          data-testid={`line-${line.seq}`}
          aria-current={active || undefined}
          onClick={
            onSeek &&
            ((event) => {
              if (window.getSelection()?.isCollapsed === false) return;
              onSeek(wordNear(wordsOf(line), offsetAt(event))?.ms ?? line.startMs);
            })
          }
          className={cn(
            "rounded-[4px] box-decoration-clone transition-colors",
            onSeek && "cursor-pointer",
            active && "bg-hover",
            found && "bg-mark/45",
          )}
        >
          <Spoken meetingId={meetingId} line={line} query={query} playhead={playhead} />
        </span>
      </>
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

function offsetAt(event: MouseEvent<HTMLElement>) {
  const { clientX: x, clientY: y, currentTarget: root } = event;
  const caret = document.caretRangeFromPoint(x, y);
  if (!caret || !root.contains(caret.startContainer)) return 0;
  const { startContainer: node, startOffset: at } = caret;
  const range = document.createRange();
  range.setStart(root, 0);
  range.setEnd(node, at);
  const offset = range.toString().length;
  if (!(node instanceof Text)) return offset;
  const over = (from: number) => {
    if (from < 0 || from >= node.length) return false;
    range.setStart(node, from);
    range.setEnd(node, from + 1);
    return [...range.getClientRects()].some((box) => x >= box.left && x <= box.right && y >= box.top && y <= box.bottom);
  };
  return !over(at) && over(at - 1) ? offset - 1 : offset;
}

const silent: Word[] = [];

function Spoken({
  meetingId,
  line,
  query,
  playhead,
}: {
  meetingId: string;
  line: Line;
  query: string;
  playhead: Playhead | undefined;
}) {
  const now = playhead?.now;
  const playing = playhead?.playing ?? false;
  const ms = playhead?.ms ?? 0;
  const words = playhead ? wordsOf(line) : silent;
  const [heard, setHeard] = useState(() => wordAt(words, ms));

  useEffect(() => {
    if (!playing || !now) return;
    let frame = requestAnimationFrame(function tick() {
      setHeard(wordAt(words, now()));
      frame = requestAnimationFrame(tick);
    });
    return () => cancelAnimationFrame(frame);
  }, [words, now, playing]);

  return <Words meetingId={meetingId} line={line} query={query} word={words[playing ? heard : wordAt(words, ms)]} />;
}

function Words({ meetingId, line, query, word }: { meetingId: string; line: Line; query: string; word?: Word }) {
  const { text } = line;
  const places = line.guesses
    .flatMap((guess) => {
      const found: { at: number; end: number; guess: Guess }[] = [];
      for (let at = text.indexOf(guess.heard); at !== -1; at = text.indexOf(guess.heard, at + guess.heard.length)) {
        found.push({ at, end: at + guess.heard.length, guess });
      }
      return found;
    })
    .toSorted((a, b) => a.at - b.at);
  const parts: ReactNode[] = [];
  let from = 0;
  for (const place of places) {
    if (place.at < from) continue;
    if (place.at > from) parts.push(<Stretch key={from} text={text} from={from} to={place.at} query={query} word={word} />);
    parts.push(
      <Guessed key={place.at} meetingId={meetingId} testId={`guess-${line.seq}-${place.at}`} guess={place.guess}>
        <Stretch text={text} from={place.at} to={place.end} query={query} word={word} />
      </Guessed>,
    );
    from = place.end;
  }
  if (from < text.length) parts.push(<Stretch key={from} text={text} from={from} to={text.length} query={query} word={word} />);
  return parts;
}

function Stretch({ text, from, to, query, word }: { text: string; from: number; to: number; query: string; word?: Word }) {
  if (!word || word.end <= from || word.at >= to) return <Highlight text={text.slice(from, to)} query={query} />;
  const at = Math.max(word.at, from);
  const end = Math.min(word.end, to);
  return (
    <>
      <Highlight text={text.slice(from, at)} query={query} />
      <span data-testid="word-active" className="-mx-0.5 rounded-[4px] bg-primary box-decoration-clone px-0.5 text-primary-foreground">
        {text.slice(at, end)}
      </span>
      <Highlight text={text.slice(end, to)} query={query} />
    </>
  );
}

function Guessed({
  meetingId,
  testId,
  guess,
  children,
}: {
  meetingId: string;
  testId: string;
  guess: Guess;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const answer = async (misheard: boolean) => {
    setBusy(true);
    setError(null);
    try {
      await Library.answerGuess(meetingId, guess.heard, guess.meant, misheard);
      setOpen(false);
    } catch (err) {
      setError(errorMessage(err));
    }
    setBusy(false);
  };

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <span
          role="button"
          tabIndex={-1}
          data-testid={testId}
          className="cursor-pointer underline decoration-warning decoration-dotted decoration-[1.5px] underline-offset-[5px] data-[state=open]:decoration-solid"
        >
          {children}
        </span>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          side="top"
          sideOffset={6}
          collisionPadding={12}
          onOpenAutoFocus={(event) => event.preventDefault()}
          onClick={(event) => event.stopPropagation()}
          className="z-50 flex max-w-[320px] flex-col gap-1 rounded-xl bg-popover p-1.5 text-popover-foreground shadow-[0_12px_36px_rgba(0,0,0,0.14),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"
        >
          <div className="flex items-center gap-1">
            <Button variant="primary" size="md" disabled={busy} data-testid="guess-confirm" onClick={() => void answer(true)}>
              改成「{guess.meant}」
            </Button>
            <Button variant="ghost" size="md" disabled={busy} data-testid="guess-drop" onClick={() => void answer(false)}>
              没听错
            </Button>
          </div>
          {error && <span className="px-1.5 pb-0.5 text-label text-destructive">{error}</span>}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
