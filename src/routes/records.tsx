import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { ChevronRight, LayoutGrid, List, LoaderCircle, Upload } from "lucide-react";
import { useEffect, useState } from "react";
import { HeaderActions, PageTitle } from "@/components/header-slot";
import { cardGrid, MeetingCard, MeetingState, Speakers, useNow } from "@/components/meeting-card";
import { MeetingMenu, type OnError } from "@/components/meeting-menu";
import { Notice } from "@/components/notice";
import { RecDot } from "@/components/rec-dot";
import { SourceSummary } from "@/components/source-summary";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import type { Card } from "@/mygo";
import { dayGroup, type DayGroup } from "@/lib/format";
import { useImport, useImporter } from "@/lib/importer";
import { cardsQuery, settingsQuery } from "@/lib/queries";
import { isActive, useRecorder } from "@/lib/recorder";
import { useSheet, useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { useWorkspace } from "@/lib/workspace";

const order: DayGroup[] = ["今天", "昨天", "本周", "更早"];

export function RecordsPage() {
  useWindowTitle("MeetKeep");
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const setSheetOpen = useSheet((s) => s.setOpen);
  const view = useWorkspace((s) => s.view);
  const setView = useWorkspace((s) => s.setView);
  const cards = useQuery(cardsQuery).data ?? [];
  const now = useNow();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (search.new !== 1) return;
    setSheetOpen(true);
    void navigate({ to: "/", search: {}, replace: true });
  }, [search.new, setSheetOpen, navigate]);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain" data-testid="records">
      <PageTitle title="记录" />
      {cards.length > 0 && (
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
      )}
      <Capture />
      <Notice testId="records-error" message={error} onDismiss={() => setError(null)} className="mx-4 mt-4" />
      {cards.length === 0 ? null : view === "list" ? (
        <Rows cards={cards} now={now} onError={setError} />
      ) : (
        <div className="flex flex-col gap-5 px-4 pt-4 pb-5">
          <Grid cards={cards} now={now} onError={setError} />
        </div>
      )}
    </div>
  );
}

function Capture() {
  const navigate = useNavigate();
  const settings = useQuery(settingsQuery);
  const status = useRecorder((s) => s.status);
  const setSheetOpen = useSheet((s) => s.setOpen);
  const sources = settings.data?.sources;
  const importing = useImporter((s) => s.importing);
  const { pick } = useImport();
  const live = isActive(status) && status.meeting ? status.meeting : null;

  return (
    <div className="sticky top-0 z-20 border-b border-divider bg-background">
      <div className="flex h-[64px] items-center gap-2 pr-2.5 pl-4" data-testid="capture">
        <button
          type="button"
          data-testid="start-recording"
          onClick={() => (live ? void navigate({ to: "/meetings/$id", params: { id: live.id } }) : setSheetOpen(true))}
          className="inline-flex h-10 shrink-0 items-center gap-2 rounded-full bg-primary pr-5 pl-4 text-ui font-semibold text-primary-foreground transition-colors outline-none hover:bg-[#262626] focus-visible:ring-[3px] focus-visible:ring-black/15"
        >
          {live ? (
            <RecDot paused={status.phase === "paused"} color="bg-rec" className="size-2" />
          ) : (
            <span aria-hidden className="size-2 rounded-full bg-rec" />
          )}
          {live ? "回到录制" : "开始录制"}
        </button>
        {!live && (
          <button
            type="button"
            data-testid="capture-sources"
            onClick={() => setSheetOpen(true)}
            className="inline-flex h-8 min-w-0 items-center gap-[7px] rounded-md px-2 text-ui text-subtle transition-colors outline-none hover:bg-hover hover:text-secondary-foreground focus-visible:bg-hover"
          >
            {sources && (
              <SourceSummary microphone={sources.microphone} system={sources.system} apps={sources.apps} className="min-w-0" />
            )}
            <ChevronRight className="size-[13px] shrink-0 text-placeholder" />
          </button>
        )}
        <button
          type="button"
          data-testid="capture-import"
          disabled={importing}
          aria-busy={importing}
          onClick={() => void pick()}
          className="ml-auto inline-flex h-9 shrink-0 items-center gap-2 rounded-full px-3.5 text-ui font-medium text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:bg-hover disabled:opacity-60"
        >
          {importing ? <LoaderCircle className="size-4 animate-spin" /> : <Upload className="size-4" />}
          导入录音或视频
        </button>
      </div>
    </div>
  );
}

function Grid({ cards, now, onError }: { cards: Card[]; now: Date; onError: OnError }) {
  const sections = order
    .map((label) => ({ label, items: cards.filter((card) => dayGroup(card.meeting.startedAt, now) === label) }))
    .filter((section) => section.items.length > 0);
  return sections.map((section) => (
    <section key={section.label} className="flex flex-col gap-2">
      <h2 className="m-0 px-0.5 text-label font-medium text-muted-foreground">{section.label}</h2>
      <div className={cardGrid}>
        {section.items.map((card) => (
          <MeetingCard key={card.meeting.id} card={card} now={now} onError={onError} />
        ))}
      </div>
    </section>
  ));
}

const columns = "grid grid-cols-[minmax(0,1fr)_120px_150px_36px] items-center gap-x-4 px-4";

function Rows({ cards, now, onError }: { cards: Card[]; now: Date; onError: OnError }) {
  return (
    <div data-testid="records-list">
      <div className={cn(columns, "h-10 border-b border-divider text-label text-muted-foreground")}>
        <span>标题</span>
        <span>说话人</span>
        <span>时间</span>
        <span />
      </div>
      <div>
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
