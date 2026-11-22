import { BoardPane } from "@/components/board";
import { NotesDoc } from "@/components/notes-doc";
import { SpeakersPane } from "@/components/speakers-pane";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Meeting } from "@/mygo";
import { clock } from "@/lib/format";
import { useQuery } from "@tanstack/react-query";
import { useNotes, type Jump } from "@/lib/notes";
import { meetingQuery } from "@/lib/queries";

export type AsideTab = "summary" | "board" | "speakers";

const asideTabs: AsideTab[] = ["summary", "board", "speakers"];

export function MeetingAside({
  meeting,
  live,
  tab,
  onTabChange,
  onJump,
  onRetry,
}: {
  meeting: Meeting;
  live: boolean;
  tab: AsideTab;
  onTabChange: (tab: AsideTab) => void;
  onJump: (jump: Jump) => void;
  onRetry?: () => void;
}) {
  const state = useNotes(meeting.id);
  const notes = state?.notes ?? null;
  const speakers = useQuery(meetingQuery(meeting.id)).data?.speakers.length ?? 0;
  const withSpeakers = !live && speakers > 0;
  const shown = tab === "speakers" && !withSpeakers ? "summary" : tab;
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
              <span className="text-label font-normal text-muted-foreground tabular-nums">{speakers}</span>
            </TabsTrigger>
          )}
          {label && (
            <span className="ml-auto self-center text-meta text-muted-foreground tabular-nums" data-testid="notes-status">
              {label}
            </span>
          )}
        </TabsList>
        <TabsContent value="summary" className="flex flex-col">
          <NotesDoc meetingId={meeting.id} state={state} live={live} onJump={onJump} onRetry={onRetry} />
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
