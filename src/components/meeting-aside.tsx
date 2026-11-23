import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { BoardPane } from "@/components/board";
import { NotesDoc } from "@/components/notes-doc";
import { SpeakersPane } from "@/components/speakers-pane";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Meeting } from "@/mygo";
import { copyText } from "@/lib/clipboard";
import { clock } from "@/lib/format";
import { notesToDoc, summaryMarkdown, type DocNode } from "@/lib/notes-doc";
import { useQuery } from "@tanstack/react-query";
import { useNotes, type Jump } from "@/lib/notes";
import { meetingQuery } from "@/lib/queries";
import { byTalk } from "@/lib/speakers";

export type AsideTab = "summary" | "board" | "speakers";

const asideTabs: AsideTab[] = ["summary", "board", "speakers"];

export function MeetingAside({
  meeting,
  live,
  tab,
  onTabChange,
  onJump,
  pending = false,
  onRetry,
}: {
  meeting: Meeting;
  live: boolean;
  tab: AsideTab;
  onTabChange: (tab: AsideTab) => void;
  onJump: (jump: Jump) => void;
  pending?: boolean;
  onRetry?: () => void;
}) {
  const state = useNotes(meeting.id);
  const [copied, setCopied] = useState(false);
  const notes = state?.notes ?? null;
  const speakers = useQuery(meetingQuery(meeting.id)).data?.speakers ?? [];
  const withSpeakers = !live && speakers.length > 0;
  const shown = tab === "speakers" && !withSpeakers ? "summary" : tab;
  const copy = async () => {
    if (!state || !notes) return;
    let doc: DocNode = notesToDoc(notes);
    if (state.doc !== "") {
      try {
        doc = JSON.parse(state.doc) as DocNode;
      } catch {
        doc = notesToDoc(notes);
      }
    }
    try {
      const names = byTalk(speakers).map((speaker) => speaker.name);
      const head = [meeting.title, ...(names.length > 0 ? [`发言人：${names.join("、")}`] : [])];
      await copyText([...head, "", summaryMarkdown(doc)].join("\n"));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };
  const label = !notes ? null : notes.final ? (shown === "summary" ? "定稿" : null) : `整理到 ${clock(notes.coveredMs)}`;

  return (
    <aside aria-label="纪要、白板与发言人" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <Tabs
        value={shown}
        onValueChange={(value) => onTabChange(asideTabs.find((item) => item === value) ?? "summary")}
        className="flex-1"
      >
        <TabsList className="h-11 gap-[22px] border-divider px-7">
          <TabsTrigger value="summary" data-testid="tab-summary" className="text-row">
            纪要
          </TabsTrigger>
          <TabsTrigger value="board" data-testid="tab-board" className="text-row">
            白板
          </TabsTrigger>
          {withSpeakers && (
            <TabsTrigger value="speakers" data-testid="tab-speakers" className="gap-1.5 text-row">
              发言人
              <span className="text-label font-normal text-muted-foreground tabular-nums">{speakers.length}</span>
            </TabsTrigger>
          )}
          <span className="ml-auto flex items-center gap-1.5 self-center">
            {label && (
              <span className="text-meta text-muted-foreground tabular-nums" data-testid="notes-status">
                {label}
              </span>
            )}
            {shown === "summary" && notes?.final && (
              <button
                type="button"
                aria-label="复制摘要和待办"
                title="复制摘要和待办"
                data-testid="notes-copy"
                onClick={() => void copy()}
                className="inline-flex size-7 items-center justify-center rounded-[8px] text-subtle transition-colors outline-none hover:bg-hover hover:text-foreground focus-visible:bg-hover"
              >
                {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
              </button>
            )}
          </span>
        </TabsList>
        <TabsContent value="summary" className="flex flex-col">
          <NotesDoc
            meetingId={meeting.id}
            state={state}
            live={live}
            pending={pending}
            speakers={withSpeakers ? speakers : []}
            onSpeakers={() => onTabChange("speakers")}
            onJump={onJump}
            onRetry={onRetry}
          />
        </TabsContent>
        <TabsContent value="board" className="flex flex-col">
          {shown === "board" && (
            <BoardPane meetingId={meeting.id} title={meeting.title} topics={notes?.topics ?? []} onJump={onJump} />
          )}
        </TabsContent>
        {withSpeakers && (
          <TabsContent value="speakers" className="flex flex-col">
            {shown === "speakers" && (
              <SpeakersPane meetingId={meeting.id} durationMs={meeting.durationMs} onJump={onJump} />
            )}
          </TabsContent>
        )}
      </Tabs>
    </aside>
  );
}
