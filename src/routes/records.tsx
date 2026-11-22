import { useQuery } from "@tanstack/react-query";
import { PageTitle } from "@/components/header-slot";
import { cardGrid, MeetingCard, useNow } from "@/components/meeting-card";
import { dayGroup, type DayGroup } from "@/lib/format";
import { cardsQuery } from "@/lib/queries";
import { useWindowTitle } from "@/lib/ui";

const order: DayGroup[] = ["今天", "昨天", "本周", "更早"];

export function RecordsPage() {
  useWindowTitle("MeetKeep");
  const cards = useQuery(cardsQuery);
  const now = useNow();
  const sections = order
    .map((label) => ({
      label,
      items: (cards.data ?? []).filter((card) => dayGroup(card.meeting.startedAt, now) === label),
    }))
    .filter((section) => section.items.length > 0);

  if (cards.data?.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center text-ui text-muted-foreground" data-testid="records-empty">
        <PageTitle title="记录" />
        还没有记录
      </div>
    );
  }
  return (
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain" data-testid="records">
      <PageTitle title="记录" />
      <div className="flex flex-col gap-5 pb-2">
        {sections.map((section) => (
          <section key={section.label} className="flex flex-col gap-2">
            <h2 className="m-0 px-1.5 text-label font-medium text-muted-foreground">{section.label}</h2>
            <div className={cardGrid}>
              {section.items.map((card) => (
                <MeetingCard key={card.meeting.id} card={card} now={now} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
