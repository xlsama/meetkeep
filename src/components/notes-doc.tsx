import { LoaderCircle } from "lucide-react";
import { lazy, Suspense, useState } from "react";
import { Avatar } from "@/components/avatar";
import { Notice } from "@/components/notice";
import type { NotesState, Speaker } from "@/mygo";
import type { Jump } from "@/lib/notes";
import { byTalk, speakerVoice } from "@/lib/speakers";
import { cn } from "@/lib/utils";

const NotesEditor = lazy(() => import("@/components/notes-editor/notes-editor"));

export function NotesDoc({
  meetingId,
  state,
  live,
  pending = false,
  speakers,
  onSpeakers,
  onJump,
  onRetry,
}: {
  meetingId: string;
  state: NotesState | null;
  live: boolean;
  pending?: boolean;
  speakers: Speaker[];
  onSpeakers: () => void;
  onJump: (jump: Jump) => void;
  onRetry?: () => void;
}) {
  const [saveError, setSaveError] = useState<string | null>(null);
  const editable = !live && state?.notes?.final === true;

  return (
    <div
      data-testid="notes"
      className={cn("min-h-0 flex-1 overflow-y-auto overscroll-contain px-7 pt-6", live ? "pb-[110px]" : "pb-10")}
    >
      <Notice
        testId="notes-error"
        message={state?.error || null}
        className="mb-5"
        action={
          onRetry && (
            <button
              type="button"
              data-testid="notes-retry"
              onClick={onRetry}
              className="-my-0.5 inline-flex h-[22px] shrink-0 items-center rounded-xs px-2 text-label font-semibold text-warning transition-colors hover:bg-warning-border/50"
            >
              重试
            </button>
          )
        }
      />
      {pending && state?.notes?.final !== true && (
        <div className="mb-5 flex items-center gap-2 text-label text-muted-foreground" data-testid="notes-pending">
          <LoaderCircle className="size-3.5 animate-spin" />
          正在生成完整纪要
        </div>
      )}
      <Notice testId="notes-save-error" message={saveError} onDismiss={() => setSaveError(null)} className="mb-5" />
      {speakers.length > 0 && (
        <button
          type="button"
          aria-label="说话人"
          data-testid="notes-speakers"
          onClick={onSpeakers}
          className="-mx-1.5 mb-3 flex flex-wrap items-center gap-x-3.5 gap-y-1.5 rounded-md px-1.5 py-1 text-left text-label text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:bg-hover"
        >
          {byTalk(speakers).map((speaker) => (
            <span key={speaker.id} className="inline-flex items-center gap-1.5">
              <Avatar name={speaker.name} tone={speakerVoice(speaker, speakers).avatar} className="size-[18px] text-micro" />
              {speaker.name}
            </span>
          ))}
        </button>
      )}
      {state && (state.notes !== null || state.doc !== "") && (
        <Suspense fallback={null}>
          <NotesEditor
            key={meetingId}
            meetingId={meetingId}
            state={state}
            editable={editable}
            onJump={onJump}
            onSaveError={setSaveError}
          />
        </Suspense>
      )}
    </div>
  );
}
