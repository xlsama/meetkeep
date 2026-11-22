import { useQuery } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { Popover } from "radix-ui";
import { useState, type RefObject } from "react";
import { Avatar } from "@/components/avatar";
import { Button } from "@/components/ui/button";
import { Library } from "@/mygo";
import { errorMessage, meetingQuery, peopleQuery } from "@/lib/queries";
import { speakerVoice, tones } from "@/lib/speakers";

export function SpeakerPopover({
  meetingId,
  speakerId,
  avatar,
  onDone,
}: {
  meetingId: string;
  speakerId: number;
  avatar: RefObject<HTMLButtonElement | null>;
  onDone: () => void;
}) {
  return (
    <Popover.Portal>
      <Popover.Content
        align="start"
        sideOffset={6}
        collisionPadding={12}
        data-testid="speaker-popover"
        onInteractOutside={(event) => {
          if (event.target instanceof Node && avatar.current?.contains(event.target)) event.preventDefault();
        }}
        className="z-50 flex w-[280px] flex-col overflow-hidden rounded-xl bg-popover text-popover-foreground shadow-[0_12px_36px_rgba(0,0,0,0.14),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"
      >
        <Body meetingId={meetingId} speakerId={speakerId} onDone={onDone} />
      </Popover.Content>
    </Popover.Portal>
  );
}

type Suggestion = { name: string; tone: string; here: boolean };

function Body({ meetingId, speakerId, onDone }: { meetingId: string; speakerId: number; onDone: () => void }) {
  const speakers = useQuery(meetingQuery(meetingId)).data?.speakers ?? [];
  const people = useQuery(peopleQuery).data?.people ?? [];
  const speaker = speakers.find((s) => s.id === speakerId);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  if (!speaker) return null;

  const others = speakers.filter((s) => s.id !== speaker.id && !s.self);
  const suggestions: Suggestion[] = [speaker, ...others].map((s) => ({
    name: s.name,
    tone: speakerVoice(s, speakers).avatar,
    here: true,
  }));
  for (const [index, person] of people.entries()) {
    if (suggestions.some((s) => s.name === person.name)) continue;
    suggestions.push({ name: person.name, tone: tones[index % tones.length] ?? "bg-them", here: false });
  }
  const name = draft.trim();
  const chosen = name || speaker.name;
  const shown = suggestions.filter((s) => name === "" || s.name === name || s.name.includes(name));
  const target = others.find((s) => s.name === name);
  const changed = name !== "" && name !== speaker.name;

  const confirm = async () => {
    if (!changed || busy) return;
    setBusy(true);
    setError(null);
    try {
      if (target) await Library.mergeSpeakers(meetingId, speaker.id, target.id);
      else await Library.renameSpeaker(meetingId, speaker.id, name);
      onDone();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <>
      <input
        autoFocus
        aria-label="说话人名字"
        data-testid="speaker-popover-input"
        placeholder={speaker.name}
        spellCheck={false}
        autoComplete="off"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.nativeEvent.isComposing) void confirm();
        }}
        className="h-11 shrink-0 border-0 border-b border-divider bg-transparent px-3.5 text-body text-foreground outline-none placeholder:text-placeholder"
      />
      {shown.length > 0 && (
        <div className="flex max-h-[216px] flex-col overflow-y-auto overscroll-contain p-1.5">
          {shown.map((suggestion) => (
            <button
              key={suggestion.name}
              type="button"
              data-testid={`speaker-suggestion-${suggestion.name}`}
              onClick={() => setDraft(suggestion.name === speaker.name ? "" : suggestion.name)}
              className="flex h-10 shrink-0 cursor-pointer items-center gap-2.5 rounded-lg px-2 text-left outline-none hover:bg-hover focus-visible:bg-hover"
            >
              <Avatar name={suggestion.name} tone={suggestion.tone} className="size-6 text-micro" />
              <span className="min-w-0 flex-1 truncate text-ui text-foreground">{suggestion.name}</span>
              {suggestion.name === chosen && <Check className="size-4 shrink-0 text-foreground" />}
            </button>
          ))}
        </div>
      )}
      <div className="flex min-h-12 shrink-0 items-center gap-2 border-t border-divider px-3 py-2">
        <span role="alert" className="min-w-0 flex-1 text-label leading-snug break-words text-destructive">
          {error}
        </span>
        <Button
          variant="primary"
          size="md"
          data-testid="speaker-popover-confirm"
          disabled={!changed || busy}
          onClick={() => void confirm()}
        >
          {target ? "合并" : "确认"}
        </Button>
      </div>
    </>
  );
}
