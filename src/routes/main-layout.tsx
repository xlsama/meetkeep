import { useQuery } from "@tanstack/react-query";
import { Link, Outlet, useRouter } from "@tanstack/react-router";
import { Plus, Search, Settings, Users } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useState } from "react";
import { RecDot } from "@/components/rec-dot";
import { ScrollArea } from "@/components/ui/scroll-area";
import { events, type Meeting, type Status } from "@/mygo";
import { clock, dayGroup, listDate, minutes, type DayGroup } from "@/lib/format";
import { meetingsQuery, settingsQuery } from "@/lib/queries";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { useSheet } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { NewSheet } from "@/routes/new-sheet";

export function MainLayout() {
  const router = useRouter();
  useQuery(settingsQuery);

  useEffect(() => {
    if (!isMyGo()) return;
    return events.navigate.on((route) => {
      router.history.push(route || "/");
    });
  }, [router]);

  return (
    <div className="flex h-full">
      <Sidebar />
      <main className="relative flex min-w-0 flex-1 flex-col">
        <Outlet />
      </main>
      <NewSheet />
    </div>
  );
}

const order: DayGroup[] = ["今天", "昨天", "本周", "更早"];

function Sidebar() {
  const meetings = useQuery(meetingsQuery);
  const status = useRecorder((s) => s.status);
  const sheetOpen = useSheet((s) => s.open);
  const setSheetOpen = useSheet((s) => s.setOpen);

  const now = useNow();
  const sections = order
    .map((label) => ({ label, items: (meetings.data ?? []).filter((m) => dayGroup(m.startedAt, now) === label) }))
    .filter((section) => section.items.length > 0);

  return (
    <nav aria-label="会议" className="flex w-[232px] shrink-0 flex-col border-r border-line bg-sidebar">
      <div className="flex items-center gap-1.5 px-2.5 py-3">
        <label className="flex h-8 min-w-0 flex-1 items-center gap-2 rounded-[9px] bg-line px-2.5 text-muted-foreground">
          <Search className="size-3.5 shrink-0" />
          <input
            type="search"
            placeholder="搜索"
            aria-label="搜索会议"
            disabled
            data-testid="search"
            className="min-w-0 flex-1 border-0 bg-transparent text-ui text-foreground outline-none"
          />
        </label>
        <button
          type="button"
          aria-label="新录制"
          data-testid="new-recording"
          onClick={() => setSheetOpen(true)}
          className={cn(
            "inline-flex size-8 shrink-0 items-center justify-center rounded-[9px] text-secondary-foreground transition-colors hover:bg-selected",
            sheetOpen && "bg-selected",
          )}
        >
          <Plus className="size-[17px]" />
        </button>
      </div>
      <ScrollArea className="min-h-0 flex-1" viewportClassName="px-2 py-0.5">
        <div className="flex flex-col gap-[18px] pb-3">
          {sections.map((section) => (
            <div key={section.label} className="flex flex-col gap-px">
              <div className="px-2.5 pb-1 text-tiny font-medium text-muted-foreground">{section.label}</div>
              <AnimatePresence initial={false}>
                {section.items.map((meeting) => (
                  <motion.div
                    key={meeting.id}
                    layout="position"
                    initial={{ opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.15 }}
                  >
                    <MeetingItem meeting={meeting} status={status} now={now} />
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>
          ))}
        </div>
      </ScrollArea>
      <div className="flex flex-col gap-px border-t border-line p-2">
        <button
          type="button"
          disabled
          data-testid="nav-people"
          className="flex h-8 items-center gap-2.5 rounded-sm px-2.5 text-ui text-secondary-foreground disabled:opacity-50"
        >
          <Users className="size-[15px] shrink-0" />
          <span>人物</span>
        </button>
        <Link
          to="/settings"
          data-testid="nav-settings"
          className="flex h-8 items-center gap-2.5 rounded-sm px-2.5 text-ui text-secondary-foreground transition-colors hover:bg-accent"
          activeProps={{ className: "bg-selected font-medium hover:bg-selected" }}
        >
          <Settings className="size-[15px] shrink-0" />
          <span>设置</span>
        </Link>
      </div>
    </nav>
  );
}

function MeetingItem({ meeting, status, now }: { meeting: Meeting; status: Status; now: Date }) {
  const recording = meeting.status === "recording";
  return (
    <Link
      to="/meetings/$id"
      params={{ id: meeting.id }}
      data-testid={`meeting-item-${meeting.id}`}
      className="flex flex-col gap-px rounded-sm px-2.5 py-[7px] text-foreground transition-colors hover:bg-accent"
      activeProps={{ className: "bg-selected hover:bg-selected" }}
    >
      <span className="truncate text-ui leading-[1.45] font-medium">{meeting.title}</span>
      {recording ? (
        <RecordingLabel status={status.meeting?.id === meeting.id ? status : null} />
      ) : (
        <span className="text-meta leading-normal text-muted-foreground tabular-nums">
          {listDate(meeting.startedAt, now)} · {minutes(meeting.durationMs)}
        </span>
      )}
    </Link>
  );
}

function RecordingLabel({ status }: { status: Status | null }) {
  const elapsed = useElapsed(status ?? idleStatus);
  return (
    <span className="inline-flex items-center gap-1.5 text-meta leading-normal text-destructive tabular-nums">
      <RecDot paused={status?.phase === "paused"} className="size-1.5" />
      {status ? `录制中 · ${clock(elapsed)}` : "录制中"}
    </span>
  );
}

function useNow() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
}

const idleStatus: Status = { phase: "idle", meeting: null, elapsedMs: 0, at: 0 };
