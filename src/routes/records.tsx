import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { LayoutGrid, List } from "lucide-react";
import { useState } from "react";
import { HeaderActions, PageTitle } from "@/components/header-slot";
import { cardGrid, MeetingCard, MeetingState, Speakers, useNow } from "@/components/meeting-card";
import { MeetingMenu, type OnError } from "@/components/meeting-menu";
import { Notice } from "@/components/notice";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import type { Card } from "@/mygo";
import { dayGroup, type DayGroup } from "@/lib/format";
import { cardsQuery } from "@/lib/queries";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { useWorkspace } from "@/lib/workspace";

const order: DayGroup[] = ["今天", "昨天", "本周", "更早"];

export function RecordsPage() {
  useWindowTitle("MeetKeep");
  const view = useWorkspace((s) => s.view);
  const setView = useWorkspace((s) => s.setView);
  const cards = useQuery(cardsQuery);
  const now = useNow();
  const [error, setError] = useState<string | null>(null);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2" data-testid="records">
      <PageTitle title="记录" />
      <HeaderActions>
        <Segmented
          aria-label="视图"
          value={view}
          onValueChange={(value) => setView(value === "list" ? "list" : "cards")}
          className="bg-black/[0.055]"
        >
          <SegmentedItem value="cards" aria-label="卡片" title="卡片" data-testid="records-view-cards" className="px-2">
            <LayoutGrid className="size-4" />
          </SegmentedItem>
          <SegmentedItem value="list" aria-label="列表" title="列表" data-testid="records-view-list" className="px-2">
            <List className="size-4" />
          </SegmentedItem>
        </Segmented>
      </HeaderActions>
      <Notice testId="records-error" message={error} onDismiss={() => setError(null)} />
      {!cards.data ? null : cards.data.length === 0 ? (
        <div className="flex flex-1 items-center justify-center text-ui text-muted-foreground" data-testid="records-empty">
          还没有记录
        </div>
      ) : view === "list" ? (
        <Rows cards={cards.data} now={now} onError={setError} />
      ) : (
        <Grid cards={cards.data} now={now} onError={setError} />
      )}
    </div>
  );
}

function Grid({ cards, now, onError }: { cards: Card[]; now: Date; onError: OnError }) {
  const sections = order
    .map((label) => ({ label, items: cards.filter((card) => dayGroup(card.meeting.startedAt, now) === label) }))
    .filter((section) => section.items.length > 0);
  return (
    <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <div className="flex flex-col gap-5 pb-2">
        {sections.map((section) => (
          <section key={section.label} className="flex flex-col gap-2">
            <h2 className="m-0 px-1.5 text-label font-medium text-muted-foreground">{section.label}</h2>
            <div className={cardGrid}>
              {section.items.map((card) => (
                <MeetingCard key={card.meeting.id} card={card} now={now} onError={onError} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

const columns = "grid grid-cols-[minmax(0,1fr)_120px_150px_36px] items-center gap-x-4 px-4";

function Rows({ cards, now, onError }: { cards: Card[]; now: Date; onError: OnError }) {
  return (
    <div className="card flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="records-list">
      <div className={cn(columns, "h-10 shrink-0 border-b border-divider text-label text-muted-foreground")}>
        <span>标题</span>
        <span>说话人</span>
        <span>时间</span>
        <span />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        {cards.map((card) => (
          <div key={card.meeting.id} className="group/meeting relative border-b border-border-subtle last:border-b-0">
            <Link
              to="/meetings/$id"
              params={{ id: card.meeting.id }}
              data-testid={`meeting-row-${card.meeting.id}`}
              className={cn(
                columns,
                "h-[56px] text-foreground transition-colors outline-none hover:bg-hover focus-visible:bg-hover",
              )}
            >
              <span className="flex min-w-0 flex-col">
                <span className="truncate text-ui leading-[1.45] font-medium">{card.meeting.title}</span>
                {(card.headline || card.opening) && (
                  <span className="truncate text-label text-muted-foreground">{card.headline || card.opening}</span>
                )}
              </span>
              <Speakers speakers={card.speakers} />
              <MeetingState meeting={card.meeting} now={now} />
              <span />
            </Link>
            <MeetingMenu
              meeting={card.meeting}
              onError={onError}
              className="absolute top-1/2 right-6 -translate-y-1/2"
            />
          </div>
        ))}
      </div>
    </div>
  );
}
