import { Ellipsis, LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { SampleButton } from "@/components/sample-button";
import { Button } from "@/components/ui/button";
import { DialogTitle, Sheet } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Library, type MeetingDetail, type Speaker } from "@/mygo";
import { span } from "@/lib/format";
import { clearProgress } from "@/lib/progress";
import { errorMessage, meetingQuery, queryClient } from "@/lib/queries";
import { stopSample, toggleSample } from "@/lib/sample";
import { ordered, speakerVoice } from "@/lib/speakers";
import { cn } from "@/lib/utils";

export function SpeakersSheet({
  meetingId,
  speakers,
  open,
  onOpenChange,
}: {
  meetingId: string;
  speakers: Speaker[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [busy, setBusy] = useState(false);
  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        if (!busy) onOpenChange(next);
      }}
      data-testid="speakers-sheet"
      className="w-[620px]"
    >
      <SpeakersForm meetingId={meetingId} speakers={speakers} onBusy={setBusy} onDone={() => onOpenChange(false)} />
    </Sheet>
  );
}

function SpeakersForm({
  meetingId,
  speakers,
  onBusy,
  onDone,
}: {
  meetingId: string;
  speakers: Speaker[];
  onBusy: (busy: boolean) => void;
  onDone: () => void;
}) {
  const [count, setCount] = useState(() => Math.max(1, speakers.filter((s) => !s.self).length));
  const [error, setError] = useState<string | null>(null);
  const [reidentifying, setReidentifying] = useState(false);
  const total = speakers.reduce((sum, s) => sum + s.durationMs, 0);
  const rows = ordered(speakers);

  useEffect(() => stopSample, []);

  const act = async (action: () => Promise<unknown>) => {
    setError(null);
    try {
      await action();
      return true;
    } catch (err) {
      setError(errorMessage(err));
      return false;
    }
  };

  const reidentify = async () => {
    setReidentifying(true);
    onBusy(true);
    clearProgress(meetingId);
    const ok = await act(() => Library.reidentify(meetingId, count));
    setReidentifying(false);
    onBusy(false);
    if (ok) onDone();
  };

  return (
    <div className="flex min-h-0 flex-col">
      <div className="flex min-h-0 flex-col gap-1 overflow-y-auto px-6 pt-[22px] pb-2">
        <DialogTitle className="m-0 mb-1.5 text-title font-semibold">说话人</DialogTitle>
        {rows.map((speaker) => (
          <SpeakerRow
            key={speaker.id}
            meetingId={meetingId}
            speaker={speaker}
            speakers={speakers}
            share={total > 0 ? Math.round((speaker.durationMs / total) * 100) : 0}
            onRename={(name) => act(() => rename(meetingId, speaker, name))}
            onMerge={(into) => void act(() => Library.mergeSpeakers(meetingId, speaker.id, into))}
            onListen={() =>
              void act(() =>
                toggleSample(`speaker-${speaker.id}`, meetingId, speaker.sampleStartMs, speaker.sampleEndMs),
              )
            }
            onRemember={(remember) => void act(() => setRemembered(meetingId, speaker, remember))}
          />
        ))}
        {error && (
          <p role="alert" data-testid="speakers-error" data-selectable className="m-0 pt-2 text-label break-words text-destructive">
            {error}
          </p>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-2.5 pt-3.5 pr-5 pb-4 pl-6">
        <span className="text-ui text-secondary-foreground">人数</span>
        <span className="inline-flex h-[30px] items-center rounded-full border border-input">
          <button
            type="button"
            aria-label="减少"
            data-testid="speakers-count-minus"
            disabled={count <= 1}
            onClick={() => setCount((n) => Math.max(1, n - 1))}
            className="h-7 w-[30px] text-lead text-subtle outline-none disabled:opacity-40"
          >
            −
          </button>
          <span className="w-[18px] text-center text-ui tabular-nums" data-testid="speakers-count">
            {count}
          </span>
          <button
            type="button"
            aria-label="增加"
            data-testid="speakers-count-plus"
            disabled={count >= 20}
            onClick={() => setCount((n) => Math.min(20, n + 1))}
            className="h-7 w-[30px] text-lead text-subtle outline-none disabled:opacity-40"
          >
            +
          </button>
        </span>
        <Button
          variant="outline"
          size="md"
          data-testid="speakers-reidentify"
          disabled={reidentifying}
          aria-busy={reidentifying}
          onClick={() => void reidentify()}
        >
          {reidentifying && <LoaderCircle className="size-[13px] animate-spin" />}
          重新识别
        </Button>
        <span className="flex-1" />
        <Button variant="primary" size="md" data-testid="speakers-done" disabled={reidentifying} onClick={onDone}>
          完成
        </Button>
      </div>
    </div>
  );
}

function patchSpeaker(meetingId: string, speakerId: number, patch: Partial<Speaker>) {
  queryClient.setQueryData<MeetingDetail>(meetingQuery(meetingId).queryKey, (detail) =>
    detail
      ? { ...detail, speakers: detail.speakers.map((s) => (s.id === speakerId ? { ...s, ...patch } : s)) }
      : detail,
  );
}

async function rename(meetingId: string, speaker: Speaker, name: string) {
  patchSpeaker(meetingId, speaker.id, { name });
  try {
    await Library.renameSpeaker(meetingId, speaker.id, name);
  } catch (err) {
    patchSpeaker(meetingId, speaker.id, { name: speaker.name });
    throw err;
  }
}

async function setRemembered(meetingId: string, speaker: Speaker, remembered: boolean) {
  patchSpeaker(meetingId, speaker.id, { remembered });
  try {
    await Library.rememberSpeaker(meetingId, speaker.id, remembered);
  } catch (err) {
    patchSpeaker(meetingId, speaker.id, { remembered: speaker.remembered });
    throw err;
  }
}

function SpeakerRow({
  meetingId,
  speaker,
  speakers,
  share,
  onRename,
  onMerge,
  onListen,
  onRemember,
}: {
  meetingId: string;
  speaker: Speaker;
  speakers: Speaker[];
  share: number;
  onRename: (name: string) => Promise<boolean>;
  onMerge: (into: number) => void;
  onListen: () => void;
  onRemember: (remember: boolean) => void;
}) {
  const voice = speakerVoice(speaker, speakers);
  const targets = ordered(speakers).filter((s) => s.id !== speaker.id);

  return (
    <div className="flex items-center gap-3 border-b border-border-subtle py-2.5" data-testid={`speaker-${speaker.id}`}>
      <Avatar name={speaker.name} tone={voice.avatar} className="size-7 text-meta" />
      <NameInput key={`${meetingId}-${speaker.id}-${speaker.name}`} speaker={speaker} onRename={onRename} />
      <span className="w-[84px] shrink-0 text-right text-label text-muted-foreground tabular-nums">
        {span(speaker.durationMs)} · {share}%
      </span>
      {speaker.sampleEndMs > 0 ? (
        <SampleButton
          sampleKey={`speaker-${speaker.id}`}
          label={`试听${speaker.name}`}
          testId={`speaker-listen-${speaker.id}`}
          size={12}
          onClick={onListen}
        />
      ) : (
        <span aria-hidden className="w-[30px] shrink-0" />
      )}
      {speaker.self ? (
        <span aria-hidden className="w-[118px] shrink-0" />
      ) : (
        <label className="inline-flex w-[118px] shrink-0 items-center justify-end gap-2 text-label text-secondary-foreground">
          记住声音
          <Switch
            aria-label={`记住${speaker.name}的声音`}
            data-testid={`speaker-remember-${speaker.id}`}
            checked={speaker.remembered}
            onCheckedChange={onRemember}
          />
        </label>
      )}
      {speaker.self ? (
        <span aria-hidden className="w-[30px] shrink-0" />
      ) : (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={`合并${speaker.name}`}
              data-testid={`speaker-merge-${speaker.id}`}
              disabled={targets.length === 0}
            >
              <Ellipsis className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {targets.map((target) => (
              <DropdownMenuItem
                key={target.id}
                data-testid={`speaker-merge-${speaker.id}-into-${target.id}`}
                onSelect={() => onMerge(target.id)}
              >
                合并到 {target.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

function NameInput({ speaker, onRename }: { speaker: Speaker; onRename: (name: string) => Promise<boolean> }) {
  const [draft, setDraft] = useState(speaker.name);
  const cancelled = useRef(false);

  const commit = () => {
    if (cancelled.current) {
      cancelled.current = false;
      setDraft(speaker.name);
      return;
    }
    const next = draft.trim();
    if (next === "" || next === speaker.name) {
      setDraft(speaker.name);
      return;
    }
    void onRename(next).then((ok) => {
      if (!ok) setDraft(speaker.name);
    });
  };

  return (
    <input
      aria-label="说话人名字"
      data-testid={`speaker-name-${speaker.id}`}
      spellCheck={false}
      autoComplete="off"
      readOnly={speaker.self}
      value={draft}
      onChange={(event) => setDraft(event.target.value)}
      onBlur={commit}
      onKeyDown={(event) => {
        if (event.key === "Enter") event.currentTarget.blur();
        if (event.key === "Escape") {
          event.preventDefault();
          event.stopPropagation();
          cancelled.current = true;
          event.currentTarget.blur();
        }
      }}
      className={cn(
        "h-[34px] min-w-0 flex-1 rounded-md border border-transparent bg-transparent px-2.5 text-body font-medium text-foreground outline-none transition-[border-color,background-color,box-shadow]",
        !speaker.self &&
          "hover:bg-hover focus-visible:border-border-strong focus-visible:bg-background focus-visible:ring-[3px] focus-visible:ring-black/5",
      )}
    />
  );
}
