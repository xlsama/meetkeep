import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "@tanstack/react-router";
import { Ellipsis, FolderOpen, LoaderCircle, Mic, Pause, Play, Trash2, Volume2 } from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useState } from "react";
import { LevelMeter } from "@/components/level-meter";
import { FaultNotice, Notice } from "@/components/notice";
import { RecDot } from "@/components/rec-dot";
import { SourceSummary } from "@/components/source-summary";
import { TitleInput } from "@/components/title-input";
import { Transcript } from "@/components/transcript";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Library, type Meeting } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock, meetingDate, minutes, systemLabel } from "@/lib/format";
import { errorMessage, meetingQuery } from "@/lib/queries";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { useWindowTitle } from "@/lib/ui";

export function MeetingPage() {
  const { id = "" } = useParams({ strict: false });
  const navigate = useNavigate();
  const status = useRecorder((s) => s.status);
  const ready = useRecorder((s) => s.ready) || !isMyGo();
  const detail = useQuery(meetingQuery(id));
  const live = status.meeting?.id === id && status.phase !== "idle";
  const meeting = detail.data?.meeting ?? (live ? status.meeting : null);

  useWindowTitle(meeting?.title ?? "MeetKeep");

  useEffect(() => {
    if (detail.isError && !live) void navigate({ to: "/", replace: true });
  }, [detail.isError, live, navigate]);

  if (!ready || !meeting) return null;
  return live ? <LiveView key={id} meeting={meeting} /> : <DetailView key={id} meeting={meeting} />;
}

function LiveView({ meeting }: { meeting: Meeting }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-[52px] shrink-0 items-center gap-3 border-b border-divider pr-5 pl-8">
        <TitleInput id={meeting.id} title={meeting.title} className="text-title font-semibold" />
        <SourceSummary
          microphone={meeting.microphone}
          system={meeting.system}
          apps={meeting.apps}
          iconSize={13}
          className="max-w-[45%] shrink-0 gap-1.5 text-label text-muted-foreground"
        />
      </header>
      <div className="flex min-h-0 flex-1">
        <section aria-label="文字稿" className="relative flex min-w-0 flex-[1.15_1_0] flex-col">
          <FaultNotice className="mx-8 mt-4" />
          <Transcript meetingId={meeting.id} live variant="main" paddingEnd={110} />
          <RecordingBar meeting={meeting} />
        </section>
        <SummaryAside />
      </div>
    </div>
  );
}

function RecordingBar({ meeting }: { meeting: Meeting }) {
  const controls = useControls();
  const elapsed = useElapsed(controls.status);
  const levels = useRecorder((s) => s.levels);

  return (
    <div className="pointer-events-none absolute inset-x-0 bottom-6 flex flex-col items-center gap-2 px-6">
      <Notice
        testId="rec-error"
        message={controls.error}
        onDismiss={controls.clearError}
        className="pointer-events-auto max-w-[420px]"
      />
      <div
        role="toolbar"
        aria-label="录制控制"
        className="pointer-events-auto flex h-[52px] items-center gap-3.5 rounded-full bg-primary pr-2 pl-5 text-white shadow-[0_12px_36px_rgba(0,0,0,0.22)]"
      >
        <span className="inline-flex items-center gap-2.5 text-lead font-medium tabular-nums" data-testid="rec-elapsed">
          <RecDot paused={controls.paused} color="bg-rec" className="size-2" />
          {clock(elapsed)}
        </span>
        <span aria-hidden className="h-5 w-px bg-ink-line" />
        {meeting.microphone && (
          <span className="inline-flex items-center gap-1.5 text-ink-muted" title="麦克风">
            <Mic className="size-[15px]" />
            <LevelMeter level={levels.mic} on="#ffffff" off="#4a4a4a" />
          </span>
        )}
        {meeting.system && (
          <span className="inline-flex items-center gap-1.5 text-ink-muted" title={systemLabel(meeting.apps)}>
            <Volume2 className="size-[15px]" />
            <LevelMeter level={levels.system} on="#ffffff" off="#4a4a4a" />
          </span>
        )}
        <span aria-hidden className="h-5 w-px bg-ink-line" />
        <span className="flex gap-1.5">
          <Button
            variant="ink"
            size="icon-lg"
            aria-label={controls.paused ? "继续" : "暂停"}
            data-testid="rec-pause"
            disabled={controls.stopping || controls.pausing}
            onClick={() => void controls.togglePause()}
          >
            {controls.paused ? <Play className="size-4" /> : <Pause className="size-4" />}
          </Button>
          <Button
            variant="light"
            size="xl"
            data-testid="rec-stop"
            disabled={controls.stopping}
            aria-busy={controls.stopping}
            className="pr-3.5 pl-3"
            onClick={() => void controls.stop()}
          >
            {controls.stopping ? (
              <LoaderCircle className="size-3.5 animate-spin text-subtle" />
            ) : (
              <span aria-hidden className="size-2.5 rounded-[3px] bg-rec-strong" />
            )}
            停止
          </Button>
        </span>
      </div>
    </div>
  );
}

function DetailView({ meeting }: { meeting: Meeting }) {
  const sources = [meeting.microphone && "麦克风", meeting.system && systemLabel(meeting.apps)].filter(Boolean);
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-[60px] shrink-0 items-center gap-2.5 border-b border-divider pr-5 pl-8">
        <div className="flex min-w-0 flex-1 flex-col">
          <TitleInput id={meeting.id} title={meeting.title} className="text-title leading-[1.35] font-semibold" />
          <span className="truncate text-label text-muted-foreground tabular-nums" data-testid="meeting-meta">
            {[meetingDate(meeting.startedAt), minutes(meeting.durationMs), sources.join(" + ")].filter(Boolean).join(" · ")}
          </span>
        </div>
        <MoreMenu meeting={meeting} />
      </header>
      <div className="flex min-h-0 flex-1">
        <section aria-label="文字稿" className="flex min-w-0 flex-[1.15_1_0] flex-col">
          <Transcript meetingId={meeting.id} live={false} variant="main" paddingEnd={28} />
        </section>
        <SummaryAside />
      </div>
    </div>
  );
}

function MoreMenu({ meeting }: { meeting: Meeting }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const remove = async () => {
    setDeleting(true);
    setError(null);
    try {
      await Library.delete(meeting.id);
      setConfirming(false);
      void navigate({ to: "/", replace: true });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <>
      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger asChild>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label="更多" data-testid="meeting-more">
                <Ellipsis className="size-4" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>更多</TooltipContent>
        </Tooltip>
        <DropdownMenuContent align="end">
          <DropdownMenuItem data-testid="meeting-reveal" onSelect={() => void Library.reveal(meeting.id)}>
            <FolderOpen />
            在访达中显示
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            data-testid="meeting-delete"
            onSelect={() => {
              setError(null);
              setConfirming(true);
            }}
          >
            <Trash2 />
            删除
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertDialog open={confirming} onOpenChange={(open) => !deleting && setConfirming(open)}>
        <AlertDialogContent data-testid="delete-dialog">
          <AlertDialogTitle>删除「{meeting.title}」？</AlertDialogTitle>
          <AlertDialogDescription>录音会移到废纸篓。</AlertDialogDescription>
          {error && <p className="m-0 mt-1 text-label text-destructive">{error}</p>}
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="outline" size="md" data-testid="meeting-delete-cancel" disabled={deleting}>
                取消
              </Button>
            </AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button
                variant="destructive"
                size="md"
                data-testid="meeting-delete-confirm"
                disabled={deleting}
                onClick={(event) => {
                  event.preventDefault();
                  void remove();
                }}
              >
                {deleting && <LoaderCircle className="size-[13px] animate-spin" />}
                删除
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function SummaryAside() {
  return (
    <aside aria-label="纪要与白板" className="flex min-w-0 flex-[1_1_0] flex-col border-l border-divider bg-panel">
      <Tabs defaultValue="summary" className="flex-1">
        <TabsList className="h-11 gap-[22px] border-divider px-7">
          <TabsTrigger value="summary" data-testid="tab-summary" className="text-row">
            纪要
          </TabsTrigger>
          <TabsTrigger value="board" data-testid="tab-board" className="text-row">
            白板
          </TabsTrigger>
        </TabsList>
        <TabsContent value="summary" />
        <TabsContent value="board" />
      </Tabs>
    </aside>
  );
}
