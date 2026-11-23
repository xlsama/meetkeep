import { ChevronDown, ChevronUp, Search } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Line } from "@/lib/transcript";
import { cn } from "@/lib/utils";

export function useTranscriptSearch(lines: Line[]) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [at, setAt] = useState(0);
  const query = open ? text.trim() : "";
  const needle = query.toLowerCase();
  const matches =
    needle === ""
      ? []
      : lines.filter((line) => line.state === "done" && line.text.toLowerCase().includes(needle)).map((line) => line.seq);
  const index = matches.length === 0 ? -1 : Math.min(at, matches.length - 1);

  return {
    open,
    text,
    query,
    count: matches.length,
    index,
    current: matches[index] ?? null,
    show: () => setOpen(true),
    close: () => {
      setOpen(false);
      setText("");
      setAt(0);
    },
    type: (next: string) => {
      setText(next);
      setAt(0);
    },
    step: (delta: number) => {
      if (matches.length > 0) setAt((index + delta + matches.length) % matches.length);
    },
  };
}

type TranscriptSearch = ReturnType<typeof useTranscriptSearch>;

const iconButton =
  "inline-flex size-7 shrink-0 items-center justify-center rounded-[8px] text-subtle transition-colors outline-none hover:bg-hover hover:text-foreground focus-visible:bg-hover disabled:pointer-events-none disabled:opacity-40";

export function TranscriptHeader({ search, menu }: { search: TranscriptSearch; menu?: ReactNode }) {
  const input = useRef<HTMLInputElement>(null);
  const { open, show } = search;

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== "f" || !event.metaKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      show();
      input.current?.select();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [show]);

  return (
    <div className="flex h-11 shrink-0 items-center gap-1 pr-3 pl-7" data-testid="transcript-header">
      {open ? (
        <>
          <label className="-ml-2.5 flex h-7 min-w-0 flex-1 items-center gap-2 rounded-[8px] bg-secondary pr-1 pl-2.5">
            <Search className="size-3.5 shrink-0 text-muted-foreground" />
            <input
              ref={input}
              autoFocus
              type="search"
              placeholder="搜索"
              aria-label="搜索文字稿"
              data-testid="transcript-search"
              spellCheck={false}
              autoComplete="off"
              value={search.text}
              onChange={(event) => search.type(event.target.value)}
              onKeyDown={(event) => {
                if (event.nativeEvent.isComposing) return;
                if (event.key === "Enter") search.step(event.shiftKey ? -1 : 1);
                if (event.key === "Escape") {
                  event.stopPropagation();
                  search.close();
                }
              }}
              className="min-w-0 flex-1 border-0 bg-transparent text-ui text-foreground outline-none placeholder:text-placeholder"
            />
            {search.query !== "" && (
              <span
                className={cn("shrink-0 text-label tabular-nums", search.count === 0 ? "text-destructive" : "text-muted-foreground")}
                data-testid="transcript-search-count"
              >
                {search.count === 0 ? "没有找到" : `${search.index + 1} / ${search.count}`}
              </span>
            )}
            <button
              type="button"
              aria-label="上一处"
              disabled={search.count === 0}
              onClick={() => search.step(-1)}
              className={cn(iconButton, "size-6")}
            >
              <ChevronUp className="size-3.5" />
            </button>
            <button
              type="button"
              aria-label="下一处"
              data-testid="transcript-search-next"
              disabled={search.count === 0}
              onClick={() => search.step(1)}
              className={cn(iconButton, "size-6")}
            >
              <ChevronDown className="size-3.5" />
            </button>
          </label>
          <button
            type="button"
            data-testid="transcript-search-cancel"
            onClick={search.close}
            className="h-7 shrink-0 rounded-[8px] px-2 text-ui text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:bg-hover"
          >
            取消
          </button>
        </>
      ) : (
        <>
          <h2 className="m-0 min-w-0 flex-1 truncate text-row font-semibold text-foreground">文字稿</h2>
          <button type="button" aria-label="搜索文字稿" data-testid="transcript-search-open" onClick={show} className={iconButton}>
            <Search className="size-4" />
          </button>
          {menu}
        </>
      )}
    </div>
  );
}

export { iconButton as transcriptIconButton };
