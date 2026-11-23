import { useQuery } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { Popover } from "radix-ui";
import { useState, type RefObject } from "react";
import { Avatar } from "@/components/avatar";
import { Button } from "@/components/ui/button";
import { Library } from "@/mygo";
import { span } from "@/lib/format";
import { useNotes } from "@/lib/notes";
import { stateDoc, todosIn } from "@/lib/notes-doc";
import { errorMessage, meetingQuery, peopleQuery } from "@/lib/queries";
import { speakerNamed, speakerVoice, tones } from "@/lib/speakers";
import { cn } from "@/lib/utils";

export function SpeakerPopover({
  meetingId,
  speakerId,
  avatar,
  rename = false,
  onDone,
}: {
  meetingId: string;
  speakerId: number;
  avatar: RefObject<HTMLButtonElement | null>;
  rename?: boolean;
  onDone: () => void;
}) {
  const [renaming, setRenaming] = useState(rename);
  return (
    <Popover.Portal>
      <Popover.Content
        align="start"
        sideOffset={6}
        collisionPadding={12}
        data-testid="speaker-popover"
        onOpenAutoFocus={(event) => {
          if (!renaming) event.preventDefault();
        }}
        onInteractOutside={(event) => {
          if (event.target instanceof Node && avatar.current?.contains(event.target)) event.preventDefault();
        }}
        className="z-50 flex w-[280px] flex-col overflow-hidden rounded-xl bg-popover text-popover-foreground shadow-[0_12px_36px_rgba(0,0,0,0.14),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"
      >
        {renaming ? (
          <Body meetingId={meetingId} speakerId={speakerId} onDone={onDone} />
        ) : (
          <Card meetingId={meetingId} speakerId={speakerId} onRename={() => setRenaming(true)} />
        )}
      </Popover.Content>
    </Popover.Portal>
  );
}

function Card({ meetingId, speakerId, onRename }: { meetingId: string; speakerId: number; onRename: () => void }) {
  const speakers = useQuery(meetingQuery(meetingId)).data?.speakers ?? [];
  const people = useQuery(peopleQuery).data?.people ?? [];
  const notes = useNotes(meetingId);
  const speaker = speakers.find((s) => s.id === speakerId);
  if (!speaker) return null;

  const total = speakers.reduce((sum, s) => sum + s.durationMs, 0);
  const meetings = people.find((p) => p.id === speaker.personId)?.meetings ?? 0;
  const facts = [
    ...(total > 0 ? [`发言 ${span(speaker.durationMs)}`, `${Math.round((speaker.durationMs / total) * 100)}%`] : []),
    ...(meetings > 1 ? [`${meetings} 场会议`] : []),
  ];
  const todos = notes ? todosIn(stateDoc(notes)).filter((todo) => speakerNamed(todo.owner, speakers)?.id === speaker.id) : [];

  return (
    <div data-testid="speaker-card" className="flex flex-col">
      <div className="flex items-center gap-3 px-3.5 py-3.5">
        <Avatar name={speaker.name} tone={speakerVoice(speaker, speakers).avatar} className="size-10 text-body" />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="truncate text-body font-semibold text-foreground">{speaker.name}</span>
          {facts.length > 0 && (
            <span className="truncate text-label text-muted-foreground tabular-nums">{facts.join(" · ")}</span>
          )}
        </div>
        <Button variant="outline" size="md" data-testid="speaker-card-rename" onClick={onRename}>
          改名
        </Button>
      </div>
      {todos.length > 0 && (
        <div className="flex max-h-[220px] flex-col gap-2 overflow-y-auto overscroll-contain border-t border-divider px-3.5 py-3">
          <span className="text-meta text-muted-foreground">待办 {todos.length}</span>
          {todos.map((todo, index) => (
            <div key={index} className="flex items-start gap-2 text-ui leading-snug">
              <span
                className={cn("min-w-0 flex-1 text-foreground", todo.checked && "text-placeholder line-through")}
              >
                {todo.text}
              </span>
              {todo.due !== "" && <span className="notes-chip shrink-0">{todo.due}</span>}
            </div>
          ))}
        </div>
      )}
    </div>
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
