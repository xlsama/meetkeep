import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { AudioLines, FileText, MessageSquareText, Search, type LucideIcon } from "lucide-react";
import { Dialog as DialogPrimitive } from "radix-ui";
import { useEffect, useState, type ReactNode } from "react";
import { create } from "zustand";
import { Empty } from "@/components/empty";
import { Highlight } from "@/components/highlight";
import { clock, meetingDate, minutes } from "@/lib/format";
import { cardsQuery, searchQuery } from "@/lib/queries";
import { shortcuts, withCommand } from "@/lib/ui";
import { cn } from "@/lib/utils";

export const useSearchDialog = create<{ open: boolean; setOpen: (open: boolean) => void }>()((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));

export function SearchTrigger() {
  const setOpen = useSearchDialog((s) => s.setOpen);
  return (
    <button
      type="button"
      data-testid="search"
      onClick={() => setOpen(true)}
      className="flex h-8 min-w-[150px] flex-[0_1_360px] items-center gap-2 rounded-[10px] bg-black/[0.055] px-3 text-muted-foreground transition-colors outline-none hover:bg-black/[0.085] focus-visible:shadow-[0_0_0_3px_rgba(0,0,0,0.1)]"
    >
      <Search className="size-3.5 shrink-0" />
      <span className="min-w-0 flex-1 truncate text-left text-ui">搜索</span>
      <kbd className="shrink-0 font-sans text-tiny text-placeholder">{shortcuts.search}</kbd>
    </button>
  );
}

export function SearchDialog() {
  const open = useSearchDialog((s) => s.open);
  const setOpen = useSearchDialog((s) => s.setOpen);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== "k" || !withCommand(event)) return;
      event.preventDefault();
      setOpen(!useSearchDialog.getState().open);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [setOpen]);

  return (
    <DialogPrimitive.Root open={open} onOpenChange={setOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/25" />
        <DialogPrimitive.Content
          aria-describedby={undefined}
          data-testid="search-dialog"
          className="fixed top-1/2 left-1/2 z-50 flex h-[min(520px,calc(100%-48px))] w-[600px] max-w-[calc(100%-48px)] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl bg-background shadow-[0_24px_64px_rgba(0,0,0,0.22),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none"
        >
          <DialogPrimitive.Title className="sr-only">搜索</DialogPrimitive.Title>
          <SearchBody close={() => setOpen(false)} />
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}

type Item = {
  key: string;
  icon: LucideIcon;
  title: ReactNode;
  meta: string;
  open: () => void;
};

const recentCount = 8;
const linesPerMeeting = 3;

function SearchBody({ close }: { close: () => void }) {
  const navigate = useNavigate();
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const cards = useQuery(cardsQuery);
  const result = useQuery({ ...searchQuery(query), enabled: query !== "", placeholderData: keepPreviousData });

  useEffect(() => {
    const timer = window.setTimeout(() => setQuery(text.trim()), 180);
    return () => window.clearTimeout(timer);
  }, [text]);

  const meeting = (id: string, t?: number) => () => {
    close();
    void navigate({ to: "/meetings/$id", params: { id }, search: t === undefined ? {} : { t } });
  };

  const items: Item[] = [];
  if (query === "") {
    for (const card of (cards.data ?? []).slice(0, recentCount)) {
      items.push({
        key: card.meeting.id,
        icon: AudioLines,
        title: card.meeting.title,
        meta: [meetingDate(card.meeting.startedAt), minutes(card.meeting.durationMs), card.headline].filter(Boolean).join(" · "),
        open: meeting(card.meeting.id),
      });
    }
  } else {
    let found = 0;
    for (const hit of result.data ?? []) {
      found += hit.lines.length + hit.notes.length + (hit.titleMatch ? 1 : 0);
      items.push({
        key: hit.meeting.id,
        icon: AudioLines,
        title: <Highlight text={hit.meeting.title} query={query} />,
        meta: [meetingDate(hit.meeting.startedAt), minutes(hit.meeting.durationMs)].join(" · "),
        open: meeting(hit.meeting.id),
      });
      for (const line of hit.lines.slice(0, linesPerMeeting)) {
        items.push({
          key: `${hit.meeting.id}-line-${line.seq}`,
          icon: MessageSquareText,
          title: <Highlight text={line.text} query={query} />,
          meta: [hit.meeting.title, line.speaker, clock(line.startMs)].filter(Boolean).join(" · "),
          open: meeting(hit.meeting.id, line.startMs),
        });
      }
      for (const [index, note] of hit.notes.slice(0, linesPerMeeting).entries()) {
        items.push({
          key: `${hit.meeting.id}-note-${index}`,
          icon: FileText,
          title: <Highlight text={note.text} query={query} />,
          meta: [hit.meeting.title, "纪要"].join(" · "),
          open: meeting(hit.meeting.id, note.atMs),
        });
      }
    }
    if (found > 0) {
      items.push({
        key: "all",
        icon: Search,
        title: `查看全部 ${found} 处`,
        meta: "",
        open: () => {
          close();
          void navigate({ to: "/search", search: { q: query } });
        },
      });
    }
  }
  const empty = query !== "" && result.isSuccess && !result.isPlaceholderData && items.length === 0;

  return (
    <>
      <label className="flex h-[52px] shrink-0 items-center gap-3 border-b border-divider px-[18px]">
        <Search className="size-[17px] shrink-0 text-muted-foreground" />
        <input
          autoFocus
          type="search"
          placeholder="搜索"
          aria-label="搜索会议"
          data-testid="search-input"
          spellCheck={false}
          autoComplete="off"
          value={text}
          onChange={(event) => {
            setText(event.target.value);
            setActive(0);
          }}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return;
            if (event.key === "ArrowDown" || event.key === "ArrowUp") {
              event.preventDefault();
              const step = event.key === "ArrowDown" ? 1 : -1;
              setActive((index) => Math.max(0, Math.min(items.length - 1, index + step)));
            }
            if (event.key === "Enter") items[active]?.open();
          }}
          className="min-w-0 flex-1 border-0 bg-transparent text-lead text-foreground outline-none placeholder:text-placeholder"
        />
      </label>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2" data-testid="search-results">
        {query === "" && items.length > 0 && (
          <div className="px-2.5 pt-1.5 pb-1 text-label text-muted-foreground">最近</div>
        )}
        {empty && <Empty kind="search" text="没有找到" size="sm" className="py-7" />}
        {items.map((item, index) => (
          <button
            key={item.key}
            type="button"
            ref={index === active ? (element) => element?.scrollIntoView({ block: "nearest" }) : undefined}
            data-active={index === active || undefined}
            onPointerMove={() => setActive(index)}
            onClick={item.open}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left outline-none",
              index === active && "bg-hover",
            )}
          >
            <item.icon className="size-4 shrink-0 text-subtle" />
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-body leading-[1.5] font-medium text-foreground">{item.title}</span>
              {item.meta && <span className="truncate text-label text-muted-foreground tabular-nums">{item.meta}</span>}
            </span>
          </button>
        ))}
      </div>
      <div className="flex h-9 shrink-0 items-center border-t border-divider bg-hover px-[18px] text-meta text-muted-foreground">
        ↑↓ 选择 · Enter 打开
      </div>
    </>
  );
}
