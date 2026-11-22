import { BoardPane } from "@/components/board";
import { NotesDoc } from "@/components/notes-doc";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Meeting } from "@/mygo";
import { clock } from "@/lib/format";
import { useNotes, type Jump } from "@/lib/notes";

export type AsideTab = "summary" | "board";

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
  const label = !notes ? null : notes.final ? (tab === "summary" ? "定稿" : null) : `整理到 ${clock(notes.coveredMs)}`;

  return (
    <aside aria-label="纪要与白板" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <Tabs value={tab} onValueChange={(value) => onTabChange(value === "board" ? "board" : "summary")} className="flex-1">
        <TabsList className="h-11 gap-[22px] border-divider px-7">
          <TabsTrigger value="summary" data-testid="tab-summary" className="text-row">
            纪要
          </TabsTrigger>
          <TabsTrigger value="board" data-testid="tab-board" className="text-row">
            白板
          </TabsTrigger>
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
          {tab === "board" && (
            <BoardPane meetingId={meeting.id} title={meeting.title} topics={notes?.topics ?? []} onJump={onJump} />
          )}
        </TabsContent>
      </Tabs>
    </aside>
  );
}
