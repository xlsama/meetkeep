import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Link, useSearch } from "@tanstack/react-router";
import { useState } from "react";
import { Empty } from "@/components/empty";
import { PageTitle } from "@/components/header-slot";
import { Highlight } from "@/components/highlight";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { SearchHit } from "@/mygo";
import { clock, meetingDate } from "@/lib/format";
import { searchQuery } from "@/lib/queries";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";

type Filter = "all" | "transcript" | "notes" | "title";

const filters: { value: Filter; label: string }[] = [
  { value: "all", label: "全部" },
  { value: "transcript", label: "文字稿" },
  { value: "notes", label: "纪要" },
  { value: "title", label: "标题" },
];

export function SearchPage() {
  useWindowTitle("MeetKeep");
  const { q } = useSearch({ strict: false });
  const query = (q ?? "").trim();
  const result = useQuery({ ...searchQuery(query), enabled: query !== "", placeholderData: keepPreviousData });
  const [filter, setFilter] = useState<Filter>("all");
  const hits = result.data ?? [];
  const found = hits.reduce((sum, hit) => sum + hit.lines.length + hit.notes.length + (hit.titleMatch ? 1 : 0), 0);
  const empty = result.isSuccess && !result.isPlaceholderData && hits.length === 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageTitle
        title={`“${query}”`}
        meta={hits.length > 0 ? `${hits.length} 场会议 · ${found} 处` : undefined}
      />
      {empty && <Empty kind="search" text="没有找到" />}
      {hits.length > 0 && (
        <Tabs
          value={filter}
          onValueChange={(value) => setFilter(filters.find((f) => f.value === value)?.value ?? "all")}
          className="flex-1"
        >
          <TabsList className="h-11 gap-[22px] border-divider px-10">
            {filters.map((f) => (
              <TabsTrigger key={f.value} value={f.value} data-testid={`search-filter-${f.value}`} className="text-row">
                {f.label}
              </TabsTrigger>
            ))}
          </TabsList>
          {filters.map((f) => (
            <TabsContent key={f.value} value={f.value} className="overflow-y-auto overscroll-contain">
              <Results hits={hits} filter={f.value} query={query} />
            </TabsContent>
          ))}
        </Tabs>
      )}
    </div>
  );
}

function Results({ hits, filter, query }: { hits: SearchHit[]; filter: Filter; query: string }) {
  const shown = hits.filter((hit) =>
    filter === "transcript"
      ? hit.lines.length > 0
      : filter === "notes"
        ? hit.notes.length > 0
        : filter === "title"
          ? hit.titleMatch
          : true,
  );
  return (
    <div className="flex max-w-[820px] flex-col gap-[26px] px-7 py-[22px]">
      {shown.map((hit) => (
        <section key={hit.meeting.id} className="flex flex-col gap-0.5" data-testid={`search-hit-${hit.meeting.id}`}>
          <Link
            to="/meetings/$id"
            params={{ id: hit.meeting.id }}
            data-testid={`search-meeting-${hit.meeting.id}`}
            className="flex min-w-0 items-baseline gap-2.5 px-3 pb-1 text-foreground transition-colors hover:text-secondary-foreground"
          >
            <h2 className="m-0 min-w-0 truncate text-lead font-semibold">
              <Highlight text={hit.meeting.title} query={query} />
            </h2>
            <span
              className={cn(
                "shrink-0 text-label tabular-nums",
                hit.meeting.status === "failed" ? "text-destructive" : "text-muted-foreground",
              )}
            >
              {meetingDate(hit.meeting.startedAt)}
              {hit.meeting.status === "failed" && " · 转写失败"}
            </span>
          </Link>
          {(filter === "all" || filter === "transcript") &&
            hit.lines.map((line) => (
              <Link
                key={line.seq}
                to="/meetings/$id"
                params={{ id: hit.meeting.id }}
                search={{ t: line.startMs }}
                data-testid={`search-line-${hit.meeting.id}-${line.seq}`}
                className="grid grid-cols-[44px_72px_minmax(0,1fr)] items-baseline gap-x-2.5 rounded-md px-3 py-2 text-foreground transition-colors hover:bg-hover"
              >
                <span className="text-tiny text-muted-foreground tabular-nums">{clock(line.startMs)}</span>
                <span className="inline-flex min-w-0 items-center gap-1.5 text-label font-semibold">
                  <span
                    aria-hidden
                    className={cn("size-1.5 shrink-0 rounded-full", line.self ? "bg-foreground" : "bg-them")}
                  />
                  <span className="truncate">{line.speaker}</span>
                </span>
                <span className="text-body leading-[1.7]">
                  <Highlight text={line.text} query={query} />
                </span>
              </Link>
            ))}
          {(filter === "all" || filter === "notes") &&
            hit.notes.map((note, index) => (
              <Link
                key={`note-${index}`}
                to="/meetings/$id"
                params={{ id: hit.meeting.id }}
                search={{ t: note.atMs }}
                data-testid={`search-note-${hit.meeting.id}-${index}`}
                className="grid grid-cols-[44px_72px_minmax(0,1fr)] items-baseline gap-x-2.5 rounded-md px-3 py-2 text-foreground transition-colors hover:bg-hover"
              >
                <span className="text-tiny text-muted-foreground">纪要</span>
                <span className="inline-flex items-center">
                  <span aria-hidden className="size-1.5 shrink-0 rounded-full bg-faint" />
                </span>
                <span className="text-body leading-[1.7]">
                  <Highlight text={note.text} query={query} />
                </span>
              </Link>
            ))}
        </section>
      ))}
    </div>
  );
}
