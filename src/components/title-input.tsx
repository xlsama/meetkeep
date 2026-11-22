import { useRef, useState } from "react";
import { Library, type Card, type Meeting, type MeetingDetail } from "@/mygo";
import { cardsQuery, meetingQuery, meetingsQuery, queryClient } from "@/lib/queries";
import { cn } from "@/lib/utils";

function retitle(id: string, title: string) {
  queryClient.setQueryData<MeetingDetail>(meetingQuery(id).queryKey, (detail) =>
    detail ? { ...detail, meeting: { ...detail.meeting, title } } : detail,
  );
  queryClient.setQueryData<Meeting[]>(meetingsQuery.queryKey, (list) =>
    list?.map((meeting) => (meeting.id === id ? { ...meeting, title } : meeting)),
  );
  queryClient.setQueryData<Card[]>(cardsQuery.queryKey, (list) =>
    list?.map((card) => (card.meeting.id === id ? { ...card, meeting: { ...card.meeting, title } } : card)),
  );
}

export function TitleInput({ id, title, className }: { id: string; title: string; className?: string }) {
  const [draft, setDraft] = useState<string | null>(null);
  const cancelled = useRef(false);

  const commit = async (value: string | null) => {
    setDraft(null);
    if (cancelled.current || value === null) {
      cancelled.current = false;
      return;
    }
    const next = value.trim();
    if (next === "" || next === title) return;
    retitle(id, next);
    try {
      await Library.rename(id, next);
    } catch {
      retitle(id, title);
    }
  };

  return (
    <input
      aria-label="会议标题"
      data-testid="meeting-title"
      spellCheck={false}
      autoComplete="off"
      value={draft ?? title}
      onChange={(event) => setDraft(event.target.value)}
      onBlur={() => void commit(draft)}
      onKeyDown={(event) => {
        if (event.key === "Enter") event.currentTarget.blur();
        if (event.key === "Escape") {
          cancelled.current = true;
          event.currentTarget.blur();
        }
      }}
      className={cn(
        "max-w-full min-w-[4ch] truncate rounded-xs border-0 bg-transparent text-foreground outline-none [field-sizing:content]",
        className,
      )}
    />
  );
}
