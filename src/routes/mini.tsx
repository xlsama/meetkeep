import { LoaderCircle, PanelRightOpen } from "lucide-react";
import { LastLine } from "@/components/last-line";
import { RecDot } from "@/components/rec-dot";
import { Windows } from "@/mygo";
import { useControls } from "@/lib/controls";
import { clock } from "@/lib/format";
import { useElapsed } from "@/lib/recorder";

export function MiniPage() {
  const controls = useControls();
  const elapsed = useElapsed(controls.status);
  const meeting = controls.status.meeting;

  return (
    <div
      role="toolbar"
      aria-label="录制小条"
      title={controls.error ?? undefined}
      className="drag flex h-full w-full items-center gap-3 overflow-hidden rounded-full bg-ink pr-1.5 pl-4 text-ink-foreground"
    >
      <span className="inline-flex shrink-0 items-center gap-2 text-body font-semibold tabular-nums" data-testid="mini-elapsed">
        <RecDot paused={controls.paused} />
        {clock(elapsed)}
      </span>
      <span className="min-w-0 flex-1 truncate text-ui text-ink-muted">
        {meeting && (
          <LastLine meetingId={meeting.id} speakerClassName={{ mic: "text-ink-foreground", system: "text-them-soft" }} />
        )}
      </span>
      <button
        type="button"
        aria-label="展开侧边面板"
        data-testid="mini-expand"
        onClick={() => void Windows.showPanel()}
        className="no-drag inline-flex size-8 shrink-0 items-center justify-center rounded-full bg-ink-raised text-ink-foreground transition-colors hover:bg-ink-line"
      >
        <PanelRightOpen className="size-[15px]" />
      </button>
      <button
        type="button"
        aria-label="停止录制"
        data-testid="mini-stop"
        disabled={!meeting || controls.stopping}
        aria-busy={controls.stopping}
        onClick={() => void controls.stop()}
        className="no-drag inline-flex size-8 shrink-0 items-center justify-center rounded-full bg-ink-foreground transition-colors hover:bg-white disabled:opacity-70"
      >
        {controls.stopping ? (
          <LoaderCircle className="size-3.5 animate-spin text-subtle" />
        ) : (
          <span aria-hidden className="size-2.5 rounded-[2px] bg-rec-strong" />
        )}
      </button>
    </div>
  );
}
