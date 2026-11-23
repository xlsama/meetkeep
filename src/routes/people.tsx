import { useQuery } from "@tanstack/react-query";
import {
  createColumnHelper,
  createSortedRowModel,
  metaHelper,
  rowSortingFeature,
  tableFeatures,
  useTable,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, Ellipsis, LoaderCircle, Pencil, Trash2 } from "lucide-react";
import { createContext, use, useEffect, useMemo, useRef, useState, type Ref, type RefObject } from "react";
import { Avatar } from "@/components/avatar";
import { PageTitle } from "@/components/header-slot";
import { Notice } from "@/components/notice";
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

type Entry = { id: string; name: string; tone: string; meetings: number; lastAt: number; person: Person | null };

type Editor = {
  editing: number | null;
  setEditing: (id: number | null) => void;
  input: RefObject<HTMLInputElement | null>;
  act: (action: () => Promise<unknown>) => Promise<boolean>;
};

const EditorContext = createContext<Editor | null>(null);

function useEditor() {
  const editor = use(EditorContext);
  if (!editor) throw new Error("EditorContext is missing");
  return editor;
}

const features = tableFeatures({
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
  columnMeta: metaHelper<{ className: string }>(),
});
const helper = createColumnHelper<typeof features, Entry>();
const columns = helper.columns([
  helper.accessor("name", {
    header: "名字",
    sortFn: (a, b) => a.original.name.localeCompare(b.original.name, "zh-Hans-CN"),
    cell: ({ row }) => <NameCell entry={row.original} />,
  }),
  helper.accessor("meetings", {
    header: "会议",
    sortDescFirst: true,
    meta: { className: "w-[120px]" },
    cell: ({ row }) => `${row.original.meetings} 场`,
  }),
  helper.accessor("lastAt", {
    header: "最近",
    sortDescFirst: true,
    meta: { className: "w-[170px]" },
    cell: ({ row }) => (row.original.lastAt > 0 ? meetingDate(row.original.lastAt) : null),
  }),
  helper.display({
    id: "samples",
    header: "声音样本",
    meta: { className: "w-[150px]" },
    cell: ({ row }) => <SampleCell entry={row.original} />,
  }),
  helper.display({
    id: "actions",
    header: () => <span className="sr-only">操作</span>,
    meta: { className: "w-[60px]" },
    cell: ({ row }) => (row.original.person ? <ActionsCell person={row.original.person} /> : null),
  }),
]);

export function PeoplePage() {
  useWindowTitle("MeetKeep");
  const known = useQuery(peopleQuery);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<number | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const entries = useMemo<Entry[]>(
    () =>
      known.data
        ? [
            {
              id: "self",
              name: "我",
              tone: "bg-secondary-foreground",
              meetings: known.data.selfMeetings,
              lastAt: known.data.selfLastAt,
              person: null,
            },
            ...known.data.people.map((person, index) => ({
              id: String(person.id),
              name: person.name,
              tone: tones[index % tones.length] ?? "bg-them",
              meetings: person.meetings,
              lastAt: person.lastAt,
              person,
            })),
          ]
        : [],
    [known.data],
  );

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
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain" data-testid="people">
      <PageTitle title="人物" meta={`已记住 ${known.data.people.length} 人的声音`} />
      <Notice testId="people-error" message={error} onDismiss={() => setError(null)} className="mx-4 mt-4" />
      <EditorContext value={{ editing, setEditing, input, act }}>
        <PeopleTable entries={entries} />
      </EditorContext>
    </div>
  );
}

function PeopleTable({ entries }: { entries: Entry[] }) {
  const table = useTable({ features, columns, data: entries, getRowId: (entry) => entry.id });

  return (
    <table className="w-full table-fixed border-separate border-spacing-0" data-testid="people-table">
      <thead>
        {table.getHeaderGroups().map((group) => (
          <tr key={group.id}>
            {group.headers.map((header) => {
              const sorted = header.column.getIsSorted();
              return (
                <th
                  key={header.id}
                  scope="col"
                  aria-sort={sorted === "asc" ? "ascending" : sorted === "desc" ? "descending" : undefined}
                  className={cn(
                    "sticky top-0 z-10 h-10 border-b border-divider bg-background px-2 text-left text-label font-normal text-muted-foreground first:pl-4 last:pr-4",
                    header.column.columnDef.meta?.className,
                  )}
                >
                  {header.column.getCanSort() ? (
                    <button
                      type="button"
                      data-testid={`people-sort-${header.column.id}`}
                      onClick={header.column.getToggleSortingHandler()}
                      className={cn(
                        "-mx-1.5 inline-flex h-7 items-center gap-1 rounded-xs px-1.5 transition-colors outline-none hover:text-foreground focus-visible:bg-hover",
                        sorted && "text-foreground",
                      )}
                    >
                      <table.FlexRender header={header} />
                      {sorted === "asc" && <ArrowUp className="size-3" />}
                      {sorted === "desc" && <ArrowDown className="size-3" />}
                    </button>
                  ) : (
                    <table.FlexRender header={header} />
                  )}
                </th>
              );
            })}
          </tr>
        ))}
      </thead>
      <tbody className="[&>tr:last-child>td]:border-b-0">
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id} data-testid={`people-row-${row.id}`} className="group/person">
            {row.getAllCells().map((cell) => (
              <td
                key={cell.id}
                className="h-14 border-b border-border-subtle px-2 text-ui text-secondary-foreground tabular-nums transition-colors group-hover/person:bg-hover first:pl-4 last:pr-4"
              >
                <table.FlexRender cell={cell} />
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function NameCell({ entry }: { entry: Entry }) {
  const { editing, setEditing, input, act } = useEditor();
  const person = entry.person;

  return (
    <span className="flex min-w-0 items-center gap-2.5">
      <Avatar name={entry.name} tone={entry.tone} className="size-7 text-meta" />
      {person && editing === person.id ? (
        <NameInput
          ref={input}
          person={person}
          onDone={(name) => {
            setEditing(null);
            if (name !== null) void act(() => rename(person, name));
          }}
        />
      ) : (
        <span className="truncate font-medium text-foreground">{entry.name}</span>
      )}
    </span>
  );
}

function SampleCell({ entry }: { entry: Entry }) {
  const { act } = useEditor();
  const person = entry.person;

  if (!person) return <span className="text-muted-foreground">麦克风</span>;
  return (
    <SampleButton
      sampleKey={`person-${person.id}`}
      label={`试听${person.name}`}
      testId={`person-listen-${person.id}`}
      size={9}
      disabled={person.sampleMeeting === "" || person.sampleEndMs <= person.sampleStartMs}
      onClick={() =>
        void act(() =>
          toggleSample(`person-${person.id}`, person.sampleMeeting, person.sampleStartMs, person.sampleEndMs),
        )
      }
    >
      {person.samples} 段
    </SampleButton>
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

function ActionsCell({ person }: { person: Person }) {
  const { setEditing, input, act } = useEditor();
  const renaming = useRef(false);
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const remove = async () => {
    setDeleting(true);
    const ok = await act(() => People.delete(person.id));
    setDeleting(false);
    if (ok) setConfirming(false);
  };

  return (
    <span className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label="更多"
            data-testid={`person-more-${person.id}`}
            className="inline-flex size-7 items-center justify-center rounded-[8px] text-muted-foreground opacity-0 transition-[opacity,background-color] outline-none group-hover/person:opacity-100 hover:bg-black/[0.06] focus-visible:opacity-100 focus-visible:shadow-[0_0_0_2px_rgba(0,0,0,0.12)] data-[state=open]:bg-black/[0.06] data-[state=open]:opacity-100"
          >
            <Ellipsis className="size-4" />
          </button>
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
              setEditing(person.id);
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
    </span>
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
      className="h-8 w-[220px] max-w-full rounded-md border border-input bg-background px-2.5 text-ui font-medium text-foreground outline-none transition-[border-color,box-shadow] focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5"
    />
  );
}
