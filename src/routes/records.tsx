import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { LayoutGrid, List } from "lucide-react";
import { PageTitle } from "@/components/header-slot";
import { cardGrid, MeetingCard, MeetingState, Speakers, StarButton, useNow } from "@/components/meeting-card";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import type { Card } from "@/mygo";
import { dayGroup, type DayGroup } from "@/lib/format";
import { cardsQuery } from "@/lib/queries";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { useWorkspace } from "@/lib/workspace";

const order: DayGroup[] = ["今天", "昨天", "本周", "更早"];

const tabs = [
  { id: "all", label: "全部" },
  { id: "starred", label: "收藏" },
] as const;

export function RecordsPage() {
  useWindowTitle("MeetKeep");
  const navigate = useNavigate();
  const { show } = useSearch({ strict: false });
  const tab = show ?? "all";
  const view = useWorkspace((s) => s.view);
  const setView = useWorkspace((s) => s.setView);
  const cards = useQuery(cardsQuery);
  const now = useNow();
  const shown = (cards.data ?? []).filter((card) => tab !== "starred" || card.meeting.starred);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2" data-testid="records">
      <PageTitle title="记录" />
      <div className="flex shrink-0 items-center gap-1 pr-0.5">
        {tabs.map((item) => (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={tab === item.id}
            data-testid={`records-tab-${item.id}`}
            onClick={() =>
              void navigate({ to: "/meetings", search: item.id === "starred" ? { show: "starred" } : {}, replace: true })
            }
            className={cn(
              "h-8 rounded-[10px] px-3 text-ui transition-colors outline-none focus-visible:shadow-[inset_0_0_0_2px_rgba(0,0,0,0.12)]",
              tab === item.id
                ? "bg-background font-medium text-foreground"
                : "text-subtle hover:bg-black/[0.05]",
            )}
          >
            {item.label}
          </button>
        ))}
        <Segmented
          aria-label="视图"
          value={view}
          onValueChange={(value) => setView(value === "list" ? "list" : "cards")}
          className="ml-auto bg-black/[0.055]"
        >
          <SegmentedItem value="cards" aria-label="卡片" title="卡片" data-testid="records-view-cards" className="px-2">
            <LayoutGrid className="size-4" />
          </SegmentedItem>
          <SegmentedItem value="list" aria-label="列表" title="列表" data-testid="records-view-list" className="px-2">
            <List className="size-4" />
          </SegmentedItem>
        </Segmented>
      </div>
      {cards.data && shown.length === 0 ? (
        <div className="flex flex-1 items-center justify-center text-ui text-muted-foreground" data-testid="records-empty">
          {tab === "starred" ? "还没有收藏" : "还没有记录"}
        </div>
      ) : view === "list" ? (
        <Rows cards={shown} now={now} />
      ) : (
        <Grid cards={shown} now={now} />
      )}
    </div>
  );
}

function Grid({ cards, now }: { cards: Card[]; now: Date }) {
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
                <MeetingCard key={card.meeting.id} card={card} now={now} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

const columns = "grid grid-cols-[minmax(0,1fr)_120px_150px_36px] items-center gap-x-4 px-4";

function Rows({ cards, now }: { cards: Card[]; now: Date }) {
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
          <Link
            key={card.meeting.id}
            to="/meetings/$id"
            params={{ id: card.meeting.id }}
            data-testid={`meeting-row-${card.meeting.id}`}
            className={cn(
              columns,
              "group/meeting h-[56px] border-b border-border-subtle text-foreground transition-colors outline-none last:border-b-0 hover:bg-hover focus-visible:bg-hover",
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
            <StarButton meeting={card.meeting} />
          </Link>
        ))}
      </div>
    </div>
  );
}
