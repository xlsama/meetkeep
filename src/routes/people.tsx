import { useQuery } from "@tanstack/react-query";
import { Ellipsis, LoaderCircle, Pencil, Trash2 } from "lucide-react";
import { useEffect, useRef, useState, type Ref } from "react";
import { Avatar } from "@/components/avatar";
import { PageTitle } from "@/components/header-slot";
import { SampleButton } from "@/components/sample-button";
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
import { People, type Person } from "@/mygo";
import { meetingDate } from "@/lib/format";
import { errorMessage, peopleQuery, queryClient } from "@/lib/queries";
import { stopSample, toggleSample } from "@/lib/sample";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";

const tones = ["bg-them-2", "bg-them", "bg-them-3"] as const;
const head = "h-[34px] border-b border-line px-3 text-meta font-medium text-muted-foreground";
const cell = "h-14 border-b border-border-subtle px-3 text-ui text-secondary-foreground tabular-nums";

export function PeoplePage() {
  useWindowTitle("MeetKeep");
  const known = useQuery(peopleQuery);
  const [error, setError] = useState<string | null>(null);
  const people = known.data?.people ?? [];

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

  if (!known.data) return null;

  return (
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <div className="flex w-full max-w-[880px] flex-col gap-6 px-10 pt-6 pb-10">
        <PageTitle title="人物" meta={`已记住 ${people.length} 人的声音`} />
        {error && (
          <p role="alert" data-testid="people-error" data-selectable className="m-0 -mt-3 text-label break-words text-destructive">
            {error}
          </p>
        )}
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th scope="col" className={cn(head, "text-left")}>
                名字
              </th>
              <th scope="col" className={cn(head, "text-left")}>
                会议
              </th>
              <th scope="col" className={cn(head, "text-left")}>
                最近
              </th>
              <th scope="col" className={cn(head, "text-left")}>
                声音样本
              </th>
              <th scope="col" className={cn(head, "text-right")}>
                <span className="sr-only">操作</span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr data-testid="people-row-self">
              <td className={cell}>
                <Name name="我" tone="bg-secondary-foreground" />
              </td>
              <td className={cell}>{known.data.selfMeetings} 场</td>
              <td className={cell}>{known.data.selfLastAt > 0 && meetingDate(known.data.selfLastAt)}</td>
              <td className={cell}>麦克风</td>
              <td className={cell} />
            </tr>
            {people.map((person, index) => (
              <PersonRow
                key={person.id}
                person={person}
                tone={tones[index % tones.length] ?? "bg-them"}
                onListen={() =>
                  void act(() =>
                    toggleSample(`person-${person.id}`, person.sampleMeeting, person.sampleStartMs, person.sampleEndMs),
                  )
                }
                onRename={(name) => act(() => rename(person, name))}
                onDelete={() => act(() => People.delete(person.id))}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Name({ name, tone }: { name: string; tone: string }) {
  return (
    <span className="inline-flex items-center gap-2.5 text-body font-medium text-foreground">
      <Avatar name={name} tone={tone} className="size-[30px] text-label" />
      {name}
    </span>
  );
}

function patchPerson(id: number, name: string) {
  queryClient.setQueryData(peopleQuery.queryKey, (known) =>
    known ? { ...known, people: known.people.map((p) => (p.id === id ? { ...p, name } : p)) } : known,
  );
}

async function rename(person: Person, name: string) {
  patchPerson(person.id, name);
  try {
    await People.rename(person.id, name);
  } catch (err) {
    patchPerson(person.id, person.name);
    throw err;
  }
}

function PersonRow({
  person,
  tone,
  onListen,
  onRename,
  onDelete,
}: {
  person: Person;
  tone: string;
  onListen: () => void;
  onRename: (name: string) => Promise<boolean>;
  onDelete: () => Promise<boolean>;
}) {
  const [editing, setEditing] = useState(false);
  const renaming = useRef(false);
  const input = useRef<HTMLInputElement>(null);
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const remove = async () => {
    setDeleting(true);
    const ok = await onDelete();
    setDeleting(false);
    if (ok) setConfirming(false);
  };

  return (
    <tr data-testid={`people-row-${person.id}`}>
      <td className={cell}>
        {editing ? (
          <span className="inline-flex items-center gap-2.5">
            <Avatar name={person.name} tone={tone} className="size-[30px] text-label" />
            <NameInput
              ref={input}
              person={person}
              onDone={(name) => {
                setEditing(false);
                if (name !== null) void onRename(name);
              }}
            />
          </span>
        ) : (
          <Name name={person.name} tone={tone} />
        )}
      </td>
      <td className={cell}>{person.meetings} 场</td>
      <td className={cell}>{person.lastAt > 0 && meetingDate(person.lastAt)}</td>
      <td className={cell}>{person.samples} 段</td>
      <td className={cn(cell, "text-right")}>
        <span className="inline-flex gap-1">
          <SampleButton
            sampleKey={`person-${person.id}`}
            label={`试听${person.name}`}
            testId={`person-listen-${person.id}`}
            size={11}
            disabled={person.sampleMeeting === "" || person.sampleEndMs <= person.sampleStartMs}
            onClick={onListen}
          />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm" aria-label="更多" data-testid={`person-more-${person.id}`}>
                <Ellipsis className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="end"
              onCloseAutoFocus={(event) => {
                if (!renaming.current) return;
                renaming.current = false;
                event.preventDefault();
                input.current?.focus();
              }}
            >
              <DropdownMenuItem
                data-testid={`person-rename-${person.id}`}
                onSelect={() => {
                  renaming.current = true;
                  setEditing(true);
                }}
              >
                <Pencil />
                改名
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                data-testid={`person-delete-${person.id}`}
                onSelect={() => setConfirming(true)}
              >
                <Trash2 />
                删除
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </span>
        <AlertDialog open={confirming} onOpenChange={(open) => !deleting && setConfirming(open)}>
          <AlertDialogContent data-testid="person-delete-dialog">
            <AlertDialogTitle>删除「{person.name}」的声音？</AlertDialogTitle>
            <AlertDialogDescription>会议里的名字会保留。</AlertDialogDescription>
            <AlertDialogFooter>
              <AlertDialogCancel asChild>
                <Button variant="outline" size="md" data-testid="person-delete-cancel" disabled={deleting}>
                  取消
                </Button>
              </AlertDialogCancel>
              <AlertDialogAction asChild>
                <Button
                  variant="destructive"
                  size="md"
                  data-testid="person-delete-confirm"
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
      </td>
    </tr>
  );
}

function NameInput({
  ref,
  person,
  onDone,
}: {
  ref: Ref<HTMLInputElement>;
  person: Person;
  onDone: (name: string | null) => void;
}) {
  const [draft, setDraft] = useState(person.name);
  const cancelled = useRef(false);

  return (
    <input
      ref={ref}
      aria-label="名字"
      data-testid={`person-name-${person.id}`}
      spellCheck={false}
      autoComplete="off"
      value={draft}
      onChange={(event) => setDraft(event.target.value)}
      onFocus={(event) => event.currentTarget.select()}
      onBlur={() => {
        const next = draft.trim();
        onDone(cancelled.current || next === "" || next === person.name ? null : next);
      }}
      onKeyDown={(event) => {
        if (event.key === "Enter") event.currentTarget.blur();
        if (event.key === "Escape") {
          event.preventDefault();
          cancelled.current = true;
          event.currentTarget.blur();
        }
      }}
      className="h-[34px] w-[200px] rounded-md border border-input bg-background px-2.5 text-body text-foreground outline-none transition-[border-color,box-shadow] focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5"
    />
  );
}
