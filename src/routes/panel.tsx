import { useQuery } from "@tanstack/react-query";
import { AppWindow, LoaderCircle, Mic, PanelRightClose, Pause, Play, Volume2, X } from "lucide-react";
import { useRef, useState, type ReactNode } from "react";
import { BoardPreview } from "@/components/board";
import { LevelMeter } from "@/components/level-meter";
import { FaultNotice, Notice } from "@/components/notice";
import { PanelNotes } from "@/components/notes";
import { RecDot } from "@/components/rec-dot";
import { Transcript } from "@/components/transcript";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Windows } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock, systemLabel } from "@/lib/format";
import { seqOf, useNotes, type Jump } from "@/lib/notes";
import { meetingQuery } from "@/lib/queries";
import { useElapsed, useRecorder } from "@/lib/recorder";
import { useTranscript } from "@/lib/transcript";

export function PanelPage() {
  const status = useRecorder((s) => s.status);
  const meeting = status.meeting;
  const detail = useQuery({ ...meetingQuery(meeting?.id ?? ""), enabled: meeting !== null });
  const title = detail.data?.meeting.title ?? meeting?.title ?? "";
  const meetingId = meeting?.id ?? "";
  const notes = useNotes(meetingId);
  const { lines } = useTranscript(meetingId, true);
  const [tab, setTab] = useState("transcript");
  const [focus, setFocus] = useState<{ seq: number; at: number } | null>(null);

  const jump = (target: Jump) => {
    const seq = seqOf(lines, target);
    if (seq === null) return;
    setFocus({ seq, at: Date.now() });
    setTab("transcript");
  };

  return (
    <div className="relative flex h-full flex-col border-l border-edge bg-background">
      <ResizeHandle />
      <div className="drag flex h-10 shrink-0 items-center gap-1 pt-1 pr-2 pl-[18px]">
        <span className="min-w-0 flex-1 truncate text-row font-semibold" data-testid="panel-title">
          {title}
        </span>
        <IconButton
          label="打开主窗口"
          testId="panel-open-main"
          onClick={() => void Windows.showMain(meeting ? `/meetings/${meeting.id}` : "")}
        >
          <AppWindow className="size-4" />
        </IconButton>
        <IconButton label="收起成小条" testId="panel-collapse" onClick={() => void Windows.collapsePanel()}>
          <PanelRightClose className="size-4" />
        </IconButton>
        <IconButton label="关闭" testId="panel-close" onClick={() => void Windows.hidePanel()}>
          <X className="size-4" />
        </IconButton>
      </div>
      <Controls />
      <Tabs value={tab} onValueChange={setTab} className="flex-1">
        <TabsList className="h-[38px] gap-5 border-line px-[18px]">
          <TabsTrigger value="transcript" data-testid="panel-tab-transcript" className="text-ui">
            文字稿
          </TabsTrigger>
          <TabsTrigger value="summary" data-testid="panel-tab-summary" className="text-ui">
            纪要
          </TabsTrigger>
          <TabsTrigger value="board" data-testid="panel-tab-board" className="text-ui">
            白板
          </TabsTrigger>
        </TabsList>
        <FaultNotice className="mx-[18px] mt-2.5" />
        <TabsContent value="transcript" forceMount className="flex flex-col">
          {meeting && (
            <Transcript
              meetingId={meeting.id}
              live
              variant="panel"
              paddingEnd={18}
              activeSeq={focus?.seq ?? null}
              revealSeq={focus?.seq ?? null}
              revealKey={focus?.at}
            />
          )}
        </TabsContent>
        <TabsContent value="summary" className="flex flex-col">
          <PanelNotes state={notes} onJump={jump} />
        </TabsContent>
        <TabsContent value="board" className="flex flex-col">
          {meeting && (
            <BoardPreview
              title={title}
              topics={notes?.notes?.topics ?? []}
              onOpen={() => void Windows.showMain(`/meetings/${meeting.id}?tab=board`)}
            />
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

function Controls() {
  const controls = useControls();
  const elapsed = useElapsed(controls.status);
  const levels = useRecorder((s) => s.levels);
  const meeting = controls.status.meeting;

  return (
    <>
      <div className="drag flex h-[46px] shrink-0 items-center gap-2.5 pr-3.5 pl-[18px]">
        <span className="inline-flex items-center gap-2 text-lead font-semibold tabular-nums" data-testid="panel-elapsed">
          <RecDot paused={controls.paused} />
          {clock(elapsed)}
        </span>
        {meeting?.microphone && (
          <span className="ml-1 inline-flex items-center gap-[5px] text-muted-foreground" title="我（麦克风）">
            <Mic className="size-[13px]" />
            <LevelMeter level={levels.mic} on="#3d3d3d" off="#d9d9d9" />
          </span>
        )}
        {meeting?.system && (
          <span
            className="inline-flex items-center gap-[5px] text-muted-foreground"
            title={`对方（${systemLabel(meeting.apps)}）`}
          >
            <Volume2 className="size-[13px]" />
            <LevelMeter level={levels.system} on="#0284c7" off="#d9d9d9" />
          </span>
        )}
        <span className="flex-1" />
        <Button
          variant="outline"
          size="icon"
          aria-label={controls.paused ? "继续" : "暂停"}
          data-testid="rec-pause"
          className="no-drag text-secondary-foreground"
          disabled={!meeting || controls.stopping || controls.pausing}
          onClick={() => void controls.togglePause()}
        >
          {controls.paused ? <Play className="size-3.5" /> : <Pause className="size-3.5" />}
        </Button>
        <Button
          variant="primary"
          size="md"
          data-testid="rec-stop"
          className="no-drag pr-3.5 pl-3"
          disabled={!meeting || controls.stopping}
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
      <Notice testId="rec-error" message={controls.error} onDismiss={controls.clearError} className="mx-[18px] mb-2" />
    </>
  );
}

function IconButton({
  label,
  testId,
  onClick,
  children,
}: {
  label: string;
  testId: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon-sm" aria-label={label} data-testid={testId} className="no-drag" onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}

function ResizeHandle() {
  const dragging = useRef(false);
  const frame = useRef(0);
  const width = useRef(0);

  return (
    <div
      aria-hidden
      data-testid="panel-resize"
      className="no-drag group absolute inset-y-0 left-0 z-10 flex w-1.5 cursor-col-resize items-center"
      onPointerDown={(event) => {
        event.currentTarget.setPointerCapture(event.pointerId);
        dragging.current = true;
      }}
      onPointerMove={(event) => {
        if (!dragging.current) return;
        width.current = window.screen.availWidth - event.screenX;
        if (frame.current) return;
        frame.current = requestAnimationFrame(() => {
          frame.current = 0;
          void Windows.resizePanel(Math.round(width.current));
        });
      }}
      onPointerUp={(event) => {
        dragging.current = false;
        event.currentTarget.releasePointerCapture(event.pointerId);
      }}
      onPointerCancel={() => {
        dragging.current = false;
      }}
    >
      <span className="ml-px h-10 w-1 rounded-[2px] bg-grip transition-colors group-hover:bg-border-strong group-active:bg-faint" />
    </div>
  );
}
