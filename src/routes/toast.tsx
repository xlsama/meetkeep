import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { Mic } from "lucide-react";
import { Windows } from "@/mygo";
import { doubts, useNotes } from "@/lib/notes";
import { meetingQuery } from "@/lib/queries";

export function ToastPage() {
  const { meeting = "" } = useSearch({ strict: false });
  const detail = useQuery({ ...meetingQuery(meeting), enabled: meeting !== "" });
  const notes = useNotes(meeting)?.notes ?? null;
  const title = detail.data?.meeting.title;
  if (title === undefined) return null;

  const todos = notes?.todos.length ?? 0;
  const doubt = notes ? doubts(notes) : 0;
  const summary = [todos > 0 && `${todos} 个待办`, doubt > 0 && `${doubt} 处存疑`].filter(Boolean).join("，");

  return (
    <div className="flex h-full w-full items-center gap-3 rounded-3xl border-[0.5px] border-black/12 bg-background py-3.5 pr-3.5 pl-4 leading-normal">
      <span
        aria-hidden
        className="inline-flex size-[38px] shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground"
      >
        <Mic className="size-[18px]" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-row font-semibold" data-testid="toast-title">
          {title} · 纪要好了
        </span>
        {summary !== "" && (
          <span className="truncate text-label text-subtle" data-testid="toast-summary">
            {summary}
          </span>
        )}
      </div>
      <button
        type="button"
        data-testid="toast-view"
        onClick={async () => {
          await Windows.showMain(`/meetings/${meeting}`);
          await Windows.closeToast();
        }}
        className="inline-flex h-[30px] shrink-0 items-center rounded-full bg-secondary px-3.5 text-ui font-medium text-foreground transition-colors hover:bg-selected"
      >
        查看
      </button>
    </div>
  );
}
