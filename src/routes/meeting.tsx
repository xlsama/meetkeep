import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import {
  Check,
  Download,
  Ellipsis,
  FileText,
  FolderOpen,
  History,
  LoaderCircle,
  Mic,
  Pause,
  Play,
  RotateCcw,
  Star,
  Trash2,
  Upload,
  Volume2,
} from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useState } from "react";
import { Avatar } from "@/components/avatar";
import { LevelMeter } from "@/components/level-meter";
import { MeetingAside, type AsideTab } from "@/components/meeting-aside";
import { HeaderActions, HeaderTitle } from "@/components/header-slot";
import { FaultNotice, Notice } from "@/components/notice";
import { AudioBar, PlayerProvider, usePlayback, VideoStage } from "@/components/player";
import { RecDot } from "@/components/rec-dot";
import { SourceSummary } from "@/components/source-summary";
import { TitleInput } from "@/components/title-input";
import { Transcript } from "@/components/transcript";
import { Workspace } from "@/components/workspace";
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
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Library, type Format, type Meeting, type Progress, type Speaker, type Step } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock, meetingDate, minutes, systemLabel } from "@/lib/format";
import { clearProgress, useProgress } from "@/lib/progress";
import { seqOf, useNotes, type Jump } from "@/lib/notes";
import { errorMessage, meetingQuery } from "@/lib/queries";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { speakerVoice } from "@/lib/speakers";
import { lineAt, useTranscript } from "@/lib/transcript";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { SpeakersSheet } from "@/routes/speakers-sheet";

export function MeetingPage() {
  const { id = "" } = useParams({ strict: false });
  const { t, tab = "summary" } = useSearch({ strict: false });
  const navigate = useNavigate();
  const status = useRecorder((s) => s.status);
  const ready = useRecorder((s) => s.ready) || !isMyGo();
  const detail = useQuery(meetingQuery(id));
  const live = status.meeting?.id === id && status.phase !== "idle";
  const meeting = detail.data?.meeting ?? (live ? status.meeting : null);

  useWindowTitle(meeting?.title ?? "MeetKeep");

  useEffect(() => {
    if (detail.isError && !live) void navigate({ to: "/meetings", replace: true });
  }, [detail.isError, live, navigate]);

  const setTab = (next: AsideTab) =>
    void navigate({
      to: "/meetings/$id",
      params: { id },
      search: (prev) => ({ ...prev, tab: next === "summary" ? undefined : next }),
      replace: true,
    });

  if (!ready || !meeting) return null;
  return live ? (
    <LiveView key={id} meeting={meeting} tab={tab} onTabChange={setTab} />
  ) : (
    <MeetingView key={id} meeting={meeting} t={t} tab={tab} onTabChange={setTab} />
  );
}

type TabProps = { tab: AsideTab; onTabChange: (tab: AsideTab) => void };

function LiveView({ meeting, tab, onTabChange }: { meeting: Meeting } & TabProps) {
  const { lines } = useTranscript(meeting.id, true);
  const [focus, setFocus] = useState<{ seq: number; at: number } | null>(null);

  const jump = (target: Jump) => {
    const seq = seqOf(lines, target);
    if (seq !== null) setFocus({ seq, at: Date.now() });
  };

  return (
    <>
      <HeaderTitle>
        <TitleInput id={meeting.id} title={meeting.title} className="text-title font-semibold" />
      </HeaderTitle>
      <HeaderActions>
        <SourceSummary
          microphone={meeting.microphone}
          system={meeting.system}
          apps={meeting.apps}
          iconSize={13}
          className="min-w-0 gap-1.5 text-label text-muted-foreground"
        />
      </HeaderActions>
      <Workspace
        cards={{
          transcript: (
            <section aria-label="文字稿" className="relative flex min-h-0 min-w-0 flex-1 flex-col">
              <FaultNotice className="mx-8 mt-4" />
              <Transcript
                meetingId={meeting.id}
                live
                variant="main"
                paddingEnd={110}
                activeSeq={focus?.seq ?? null}
                revealSeq={focus?.seq ?? null}
                revealKey={focus?.at}
              />
              <RecordingBar meeting={meeting} />
            </section>
          ),
          notes: <MeetingAside meeting={meeting} live tab={tab} onTabChange={onTabChange} onJump={jump} />,
        }}
      />
    </>
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

type ViewProps = { meeting: Meeting; t: number | undefined } & TabProps;

function MeetingView(props: ViewProps) {
  const { meeting } = props;
  return (
    <PlayerProvider video={meeting.video !== "" && meeting.status !== "processing"}>
      <MeetingBody {...props} />
    </PlayerProvider>
  );
}

function MeetingBody({ meeting, t, tab, onTabChange }: ViewProps) {
  const navigate = useNavigate();
  const detail = useQuery(meetingQuery(meeting.id));
  const speakers = detail.data?.speakers ?? [];
  const { groups, lines } = useTranscript(meeting.id, false);
  const tracked = useProgress((s) => s.items[meeting.id]);
  const progress = tracked ?? detail.data?.progress ?? null;
  const processing = meeting.status === "processing";
  const video = meeting.video !== "" && !processing;
  const playable = (meeting.audio !== "" || meeting.video !== "") && !processing;
  const [error, setError] = useState<string | null>(null);
  const [speakersOpen, setSpeakersOpen] = useState(false);
  const playback = usePlayback();
  const activeSeq = playback.started ? (lineAt(lines, playback.currentMs)?.seq ?? null) : null;
  const revealSeq = t === undefined ? null : (lineAt(lines, t)?.seq ?? null);
  const { seek, ready } = playback;
  const sources = [meeting.microphone && "麦克风", meeting.system && systemLabel(meeting.apps)].filter(Boolean);

  useEffect(() => {
    if (t !== undefined && ready) seek(t);
  }, [t, ready, seek]);

  return (
    <>
      <HeaderTitle>
          <TitleInput id={meeting.id} title={meeting.title} className="text-title leading-[1.35] font-semibold" />
          <span className="max-w-full truncate text-label text-muted-foreground tabular-nums" data-testid="meeting-meta">
            {[meetingDate(meeting.startedAt), minutes(meeting.durationMs), sources.join(" + ")].filter(Boolean).join(" · ")}
          </span>
      </HeaderTitle>
      <HeaderActions>
        {meeting.status === "done" && speakers.length > 0 && (
          <SpeakersButton speakers={speakers} onClick={() => setSpeakersOpen(true)} />
        )}
        <ExportMenu meetingId={meeting.id} onError={setError} />
        <MoreMenu meeting={meeting} onError={setError} />
      </HeaderActions>
      {processing && <ProgressBand meeting={meeting} progress={progress} />}
      <Workspace
        cards={{
          transcript: (
        <section aria-label="文字稿" className="flex min-h-0 min-w-0 flex-1 flex-col [container-type:size]">
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
          <Notice testId="meeting-error" message={error} onDismiss={() => setError(null)} className="mx-8 mt-4" />
          {video && (
            <VideoStage
              meetingId={meeting.id}
              durationMs={meeting.durationMs}
              groups={groups}
              spaceDisabled={tab === "board"}
            />
          )}
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
          ),
          notes: (
        <MeetingAside
          meeting={meeting}
          live={false}
          tab={tab}
          onTabChange={onTabChange}
          onJump={(target) => {
            seek(target.atMs);
            void navigate({
              to: "/meetings/$id",
              params: { id: meeting.id },
              search: (prev) => ({ ...prev, t: target.atMs }),
              replace: true,
            });
          }}
          onRetry={meeting.status === "done" ? () => void run(setError, () => Library.regenerateNotes(meeting.id)) : undefined}
        />
          ),
        }}
      />
      {playable && !video && (
        <AudioBar meetingId={meeting.id} durationMs={meeting.durationMs} groups={groups} spaceDisabled={tab === "board"} />
      )}
      {meeting.status === "done" && (
        <SpeakersSheet
          meetingId={meeting.id}
          speakers={speakers}
          open={speakersOpen && speakers.length > 0}
          onOpenChange={setSpeakersOpen}
        />
      )}
    </>
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
          <Avatar
            key={speaker.id}
            name={speaker.name}
            tone={speakerVoice(speaker, speakers).avatar}
            className={cn("size-[22px] text-micro shadow-[0_0_0_2px_#ffffff]", index > 0 && "-ml-1.5")}
          />
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
    <div className="card flex shrink-0 gap-1.5 px-8 py-3.5" data-testid="processing">
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
      void navigate({ to: "/meetings", replace: true });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setDeleting(false);
    }
  };

  const processing = meeting.status === "processing";
  const notesState = useNotes(meeting.id);

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
            data-testid="more-star"
            onSelect={() => void run(onError, () => Library.star(meeting.id, !meeting.starred))}
          >
            <Star className={cn(meeting.starred && "fill-current")} />
            {meeting.starred ? "取消收藏" : "收藏"}
          </DropdownMenuItem>
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
          <DropdownMenuItem
            data-testid="more-regenerate-notes"
            disabled={meeting.status !== "done"}
            onSelect={() => void run(onError, () => Library.regenerateNotes(meeting.id))}
          >
            <FileText />
            重新生成纪要
          </DropdownMenuItem>
          {notesState?.replaced && (
            <DropdownMenuItem
              data-testid="more-restore-notes"
              onSelect={() => void run(onError, () => Library.restoreNotes(meeting.id))}
            >
              <History />
              恢复上一版纪要
            </DropdownMenuItem>
          )}
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
