import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import {
  Check,
  Download,
  Ellipsis,
  FolderOpen,
  LoaderCircle,
  Mic,
  Pause,
  Play,
  RotateCcw,
  Trash2,
  Upload,
  Volume2,
} from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useState } from "react";
import { LevelMeter } from "@/components/level-meter";
import { FaultNotice, Notice } from "@/components/notice";
import { PlayerBar } from "@/components/player-bar";
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
import { Library, type Format, type Meeting, type Progress, type Speaker, type Step } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock, meetingDate, minutes, systemLabel } from "@/lib/format";
import { resetPlayback, seek, togglePlayback, usePlayback } from "@/lib/player";
import { clearProgress, useProgress } from "@/lib/progress";
import { errorMessage, meetingQuery } from "@/lib/queries";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { initial, speakerVoice } from "@/lib/speakers";
import { lineAt, useTranscript } from "@/lib/transcript";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { SpeakersSheet } from "@/routes/speakers-sheet";

export function MeetingPage() {
  const { id = "" } = useParams({ strict: false });
  const { t } = useSearch({ strict: false });
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
  return live ? <LiveView key={id} meeting={meeting} /> : <MeetingView key={id} meeting={meeting} t={t} />;
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

async function run(onError: (error: string | null) => void, action: () => Promise<unknown>) {
  onError(null);
  try {
    await action();
  } catch (err) {
    onError(errorMessage(err));
  }
}

function reprocess(id: string, onError: (error: string | null) => void) {
  return run(onError, () => {
    clearProgress(id);
    return Library.reprocess(id);
  });
}

const typing = "input, textarea, select, button, a[href], [contenteditable], [role=dialog], [role=menu], [role=menuitem]";

function MeetingView({ meeting, t }: { meeting: Meeting; t: number | undefined }) {
  const detail = useQuery(meetingQuery(meeting.id));
  const speakers = detail.data?.speakers ?? [];
  const { groups, lines } = useTranscript(meeting.id, false);
  const tracked = useProgress((s) => s.items[meeting.id]);
  const progress = tracked ?? detail.data?.progress ?? null;
  const processing = meeting.status === "processing";
  const playable = meeting.audio !== "" && !processing;
  const [error, setError] = useState<string | null>(null);
  const [speakersOpen, setSpeakersOpen] = useState(false);
  const playError = usePlayback((s) => s.error);
  const activeSeq = usePlayback((s) => (s.engaged ? (lineAt(lines, s.current)?.seq ?? null) : null));
  const revealSeq = t === undefined ? null : (lineAt(lines, t)?.seq ?? null);
  const sources = [meeting.microphone && "麦克风", meeting.system && systemLabel(meeting.apps)].filter(Boolean);

  useEffect(() => {
    resetPlayback();
  }, [meeting.id]);

  useEffect(() => {
    if (t !== undefined) seek(t);
  }, [t]);

  useEffect(() => {
    if (!playable) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== " " || event.repeat || event.defaultPrevented) return;
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (event.target instanceof Element && event.target.closest(typing)) return;
      event.preventDefault();
      togglePlayback();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [playable]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-[60px] shrink-0 items-center gap-2.5 border-b border-divider pr-5 pl-8">
        <div className="flex min-w-0 flex-1 flex-col">
          <TitleInput id={meeting.id} title={meeting.title} className="text-title leading-[1.35] font-semibold" />
          <span className="truncate text-label text-muted-foreground tabular-nums" data-testid="meeting-meta">
            {[meetingDate(meeting.startedAt), minutes(meeting.durationMs), sources.join(" + ")].filter(Boolean).join(" · ")}
          </span>
        </div>
        {meeting.status === "done" && speakers.length > 0 && (
          <SpeakersButton speakers={speakers} onClick={() => setSpeakersOpen(true)} />
        )}
        <ExportMenu meetingId={meeting.id} onError={setError} />
        <MoreMenu meeting={meeting} onError={setError} />
      </header>
      {processing && <ProgressBand meeting={meeting} progress={progress} />}
      <div className="flex min-h-0 flex-1">
        <section aria-label="文字稿" className="flex min-w-0 flex-[1.15_1_0] flex-col">
          {meeting.status === "failed" && (
            <Notice
              testId="meeting-failed"
              message={meeting.error || "转写失败"}
              className="mx-8 mt-4"
              action={
                <button
                  type="button"
                  data-testid="reprocess"
                  onClick={() => void reprocess(meeting.id, setError)}
                  className="-my-0.5 inline-flex h-[22px] shrink-0 items-center rounded-xs px-2 text-label font-semibold text-warning transition-colors hover:bg-warning-border/50"
                >
                  重新转写
                </button>
              }
            />
          )}
          <Notice
            testId="meeting-error"
            message={error ?? playError}
            onDismiss={() => {
              setError(null);
              usePlayback.setState({ error: null });
            }}
            className="mx-8 mt-4"
          />
          <Transcript
            meetingId={meeting.id}
            live={false}
            variant="main"
            paddingEnd={28}
            activeSeq={activeSeq}
            revealSeq={revealSeq}
            onSelect={playable ? (line) => seek(line.startMs, true) : undefined}
          />
        </section>
        <SummaryAside />
      </div>
      {playable && <PlayerBar meetingId={meeting.id} durationMs={meeting.durationMs} groups={groups} />}
      {meeting.status === "done" && (
        <SpeakersSheet
          meetingId={meeting.id}
          speakers={speakers}
          open={speakersOpen && speakers.length > 0}
          onOpenChange={setSpeakersOpen}
        />
      )}
    </div>
  );
}

function SpeakersButton({ speakers, onClick }: { speakers: Speaker[]; onClick: () => void }) {
  const others = speakers.filter((s) => !s.self);
  const self = speakers.filter((s) => s.self);
  const shown = [...others.slice(0, 4 - self.length), ...self];
  return (
    <button
      type="button"
      aria-label={`说话人：${speakers.length} 人`}
      data-testid="meeting-speakers"
      onClick={onClick}
      className="inline-flex h-8 shrink-0 items-center gap-2 rounded-full border border-input pr-3 pl-1.5 text-ui font-medium whitespace-nowrap text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:ring-[3px] focus-visible:ring-black/10"
    >
      <span className="inline-flex">
        {shown.map((speaker, index) => (
          <span
            key={speaker.id}
            aria-hidden
            className={cn(
              "inline-flex size-[22px] items-center justify-center rounded-full text-micro font-semibold text-white shadow-[0_0_0_2px_#ffffff]",
              index > 0 && "-ml-1.5",
              speakerVoice(speaker, speakers).avatar,
            )}
          >
            {initial(speaker.name)}
          </span>
        ))}
      </span>
      {speakers.length} 人
    </button>
  );
}

const formats: { format: Format; label: string; testId: string }[] = [
  { format: "markdown", label: "Markdown", testId: "export-markdown" },
  { format: "text", label: "纯文本", testId: "export-text" },
  { format: "srt", label: "SRT 字幕", testId: "export-srt" },
];

function ExportMenu({ meetingId, onError }: { meetingId: string; onError: (error: string | null) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="md" data-testid="meeting-export" className="pr-[13px] pl-[11px]">
          <Upload className="size-[15px]" />
          导出
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {formats.map((item) => (
          <DropdownMenuItem
            key={item.format}
            data-testid={item.testId}
            onSelect={() => void run(onError, () => Library.export(meetingId, item.format))}
          >
            {item.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const stepLabel: Record<Step, string> = {
  import: "导入",
  transcribe: "重新转写",
  speakers: "区分说话人",
  summary: "生成纪要",
};

function ProgressBand({ meeting, progress }: { meeting: Meeting; progress: Progress | null }) {
  const imported = !meeting.microphone && !meeting.system;
  const steps: Step[] = imported
    ? ["import", "transcribe", "speakers", "summary"]
    : ["transcribe", "speakers", "summary"];
  const current = Math.max(0, progress ? steps.indexOf(progress.step) : 0);

  return (
    <div className="flex shrink-0 gap-1.5 border-b border-divider px-8 py-3.5" data-testid="processing">
      {steps.map((step, index) => {
        const done = index < current;
        const active = index === current;
        const width = done ? 100 : active ? (progress?.percent ?? 0) : 0;
        return (
          <div key={step} className="flex flex-1 flex-col gap-1.5" data-testid={`processing-${step}`}>
            <span
              className={cn(
                "flex items-center gap-1.5 text-label",
                active ? "font-semibold text-foreground" : done ? "text-foreground" : "text-muted-foreground",
              )}
            >
              {done && <Check className="size-[13px]" />}
              {stepLabel[step]}
              {active && progress && (
                <span className="font-normal text-muted-foreground tabular-nums" data-testid="processing-percent">
                  {progress.percent}%
                </span>
              )}
            </span>
            <span className="h-[3px] overflow-hidden rounded-[2px] bg-line">
              <span
                className="block h-full bg-foreground transition-[width] duration-300"
                style={{ width: `${width}%` }}
              />
            </span>
          </div>
        );
      })}
    </div>
  );
}

function MoreMenu({ meeting, onError }: { meeting: Meeting; onError: (error: string | null) => void }) {
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

  const processing = meeting.status === "processing";

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
          <DropdownMenuItem
            data-testid="more-save-recording"
            disabled={meeting.audio === "" || processing}
            onSelect={() => void run(onError, () => Library.saveRecording(meeting.id))}
          >
            <Download />
            保存录音
          </DropdownMenuItem>
          <DropdownMenuItem
            data-testid="more-reprocess"
            disabled={processing}
            onSelect={() => void reprocess(meeting.id, onError)}
          >
            <RotateCcw />
            重新转写
          </DropdownMenuItem>
          <DropdownMenuItem data-testid="meeting-reveal" onSelect={() => void run(onError, () => Library.reveal(meeting.id))}>
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
