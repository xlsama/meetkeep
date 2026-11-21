import { useQuery } from "@tanstack/react-query";
import { AppWindow, ChevronRight, LoaderCircle, Mic, PanelRightOpen, Pause, Play, Upload, Volume2 } from "lucide-react";
import { useEffect, useState, type ComponentProps, type ReactNode } from "react";
import { LastLine } from "@/components/last-line";
import { LevelMeter } from "@/components/level-meter";
import { RecDot } from "@/components/rec-dot";
import { SourceSummary } from "@/components/source-summary";
import { Button } from "@/components/ui/button";
import { Library, Recorder, Windows, type Meeting } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock, dayLabel } from "@/lib/format";
import { errorMessage, meetingQuery, meetingsQuery, settingsQuery } from "@/lib/queries";
import { isActive, useElapsed, useRecorder } from "@/lib/recorder";
import { cn } from "@/lib/utils";

export function MenuPage() {
  const status = useRecorder((s) => s.status);
  const active = isActive(status);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const importFile = async () => {
    setImporting(true);
    setError(null);
    try {
      const meeting = await Library.import();
      if (meeting) {
        await Windows.showMain(`/meetings/${meeting.id}`);
        await Windows.closeMenu();
      }
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setImporting(false);
    }
  };

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") void Windows.closeMenu();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <div
      data-testid="menu"
      className="flex w-[320px] flex-col rounded-lg border-[0.5px] border-black/12 bg-background p-1.5 text-ui leading-normal text-foreground"
    >
      {active && status.meeting ? <RecordingTop meeting={status.meeting} /> : <IdleTop />}
      <Separator />
      {active && status.meeting ? (
        <>
          <Item testId="menu-show-panel" icon={<PanelRightOpen />} shortcut="⌥⌘M" onClick={() => void Windows.showPanel()}>
            显示侧边面板
          </Item>
          <Item
            testId="menu-open-main"
            icon={<AppWindow />}
            onClick={() => void Windows.showMain(`/meetings/${status.meeting?.id ?? ""}`)}
          >
            打开主窗口
          </Item>
        </>
      ) : (
        <>
          <Item
            testId="menu-import"
            icon={importing ? <LoaderCircle className="animate-spin" /> : <Upload />}
            disabled={importing}
            onClick={() => void importFile()}
          >
            导入录音或视频…
          </Item>
          {error && (
            <p role="alert" data-testid="menu-import-error" data-selectable className="m-0 px-2.5 py-1 text-label break-words text-destructive">
              {error}
            </p>
          )}
          <Item testId="menu-open-main" icon={<AppWindow />} onClick={() => void Windows.showMain("")}>
            打开 MeetKeep
          </Item>
        </>
      )}
      <Recent exclude={active ? status.meeting?.id : undefined} />
      <Separator />
      <Item testId="menu-settings" shortcut="⌘," onClick={() => void Windows.showMain("/settings")}>
        设置…
      </Item>
      <Item testId="menu-quit" shortcut="⌘Q" onClick={() => void Windows.quit()}>
        退出 MeetKeep
      </Item>
    </div>
  );
}

function IdleTop() {
  const settings = useQuery(settingsQuery);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const sources = settings.data?.sources;

  const start = async () => {
    if (!settings.data) return;
    setStarting(true);
    setError(null);
    try {
      await Recorder.start({
        title: "",
        microphone: settings.data.sources.microphone,
        system: settings.data.sources.system,
        apps: settings.data.sources.apps,
        showPanel: settings.data.recording.showPanel,
      });
      await Windows.closeMenu();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setStarting(false);
    }
  };

  return (
    <div className="flex flex-col gap-2.5 px-2.5 pt-2.5 pb-3">
      <Button
        variant="primary"
        data-testid="menu-start"
        disabled={!settings.data || starting}
        aria-busy={starting}
        className="h-10 gap-2 text-body"
        onClick={() => void start()}
      >
        {starting ? (
          <LoaderCircle className="size-3.5 animate-spin" />
        ) : (
          <span aria-hidden className="size-2 shrink-0 rounded-full bg-rec-strong" />
        )}
        开始录制
      </Button>
      {error && (
        <p role="alert" data-testid="menu-error" data-selectable className="m-0 text-label break-words text-destructive">
          {error}
        </p>
      )}
      <button
        type="button"
        data-testid="menu-change-sources"
        onClick={() => void Windows.showMain("/?new=1")}
        className="flex h-[30px] items-center gap-2 rounded-sm px-0.5 text-label text-secondary-foreground outline-none"
      >
        {sources && (
          <SourceSummary
            microphone={sources.microphone}
            system={sources.system}
            apps={sources.apps}
            iconSize={13}
            className="gap-2"
          />
        )}
        <span className="flex-1" />
        <span className="text-muted-foreground">更改</span>
        <ChevronRight className="size-[13px] shrink-0 text-placeholder" />
      </button>
    </div>
  );
}

function RecordingTop({ meeting }: { meeting: Meeting }) {
  const controls = useControls();
  const elapsed = useElapsed(controls.status);
  const levels = useRecorder((s) => s.levels);
  const detail = useQuery(meetingQuery(meeting.id));

  return (
    <div className="flex flex-col gap-2.5 px-2.5 pt-2.5 pb-3">
      <div className="flex flex-col gap-px">
        <span className="truncate text-row font-semibold" data-testid="menu-title">
          {detail.data?.meeting.title ?? meeting.title}
        </span>
        <span className="inline-flex items-center gap-[7px] text-display font-medium tabular-nums" data-testid="menu-elapsed">
          <RecDot paused={controls.paused} className="size-2" />
          {clock(elapsed)}
        </span>
      </div>
      <div className="flex items-center gap-3.5 text-meta text-muted-foreground">
        {meeting.microphone && (
          <span className="inline-flex items-center gap-[5px]">
            <Mic className="size-[13px]" />我
            <LevelMeter level={levels.mic} on="#3d3d3d" off="#d9d9d9" />
          </span>
        )}
        {meeting.system && (
          <span className="inline-flex items-center gap-[5px]">
            <Volume2 className="size-[13px]" />对方
            <LevelMeter level={levels.system} on="#0284c7" off="#d9d9d9" />
          </span>
        )}
      </div>
      <LastLine
        meetingId={meeting.id}
        speakerClassName={{ self: "font-semibold", other: "font-semibold" }}
        className="line-clamp-2 rounded-sm bg-muted px-2.5 py-2 text-label leading-[1.6] text-secondary-foreground"
      />
      {controls.error && (
        <p role="alert" data-testid="menu-error" data-selectable className="m-0 text-label break-words text-destructive">
          {controls.error}
        </p>
      )}
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="lg"
          data-testid="menu-pause"
          className="flex-1"
          disabled={controls.stopping || controls.pausing}
          onClick={() => void controls.togglePause()}
        >
          {controls.paused ? <Play className="size-3.5" /> : <Pause className="size-3.5" />}
          {controls.paused ? "继续" : "暂停"}
        </Button>
        <Button
          variant="primary"
          size="lg"
          data-testid="menu-stop"
          className="flex-1"
          disabled={controls.stopping}
          aria-busy={controls.stopping}
          onClick={() => void controls.stop()}
        >
          {controls.stopping ? (
            <LoaderCircle className="size-[13px] animate-spin" />
          ) : (
            <span aria-hidden className="size-[9px] rounded-[2px] bg-rec" />
          )}
          停止
        </Button>
      </div>
    </div>
  );
}

function Recent({ exclude }: { exclude: string | undefined }) {
  const meetings = useQuery(meetingsQuery);
  const recent = (meetings.data ?? []).filter((m) => m.id !== exclude && m.status === "done").slice(0, 3);
  if (recent.length === 0) return null;
  return (
    <>
      <Separator />
      <div className="px-2.5 pt-1 pb-0.5 text-tiny font-medium text-muted-foreground">最近</div>
      {recent.map((meeting) => (
        <Item
          key={meeting.id}
          testId={`menu-meeting-${meeting.id}`}
          shortcut={dayLabel(meeting.startedAt)}
          onClick={() => void Windows.showMain(`/meetings/${meeting.id}`)}
        >
          {meeting.title}
        </Item>
      ))}
    </>
  );
}

function Separator() {
  return <div aria-hidden className="mx-2.5 my-1.5 h-px shrink-0 bg-line" />;
}

function Item({
  icon,
  shortcut,
  testId,
  children,
  className,
  ...props
}: { icon?: ReactNode; shortcut?: string; testId: string } & ComponentProps<"button">) {
  return (
    <button
      type="button"
      data-testid={testId}
      className={cn(
        "flex h-[30px] w-full items-center gap-2.5 rounded-[7px] px-2.5 text-left text-ui text-foreground outline-none transition-colors hover:bg-hover focus-visible:bg-hover disabled:opacity-50 disabled:hover:bg-transparent",
        className,
      )}
      {...props}
    >
      {icon && <span className="flex text-subtle [&_svg]:size-[15px]">{icon}</span>}
      <span className="min-w-0 flex-1 truncate">{children}</span>
      {shortcut && <span className="shrink-0 text-meta text-muted-foreground">{shortcut}</span>}
    </button>
  );
}
