import { useQuery } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { ChevronRight, LoaderCircle, Upload } from "lucide-react";
import { useEffect } from "react";
import { PageTitle } from "@/components/header-slot";
import { cardGrid, MeetingCard, useNow } from "@/components/meeting-card";
import { SourceSummary } from "@/components/source-summary";
import { useImport, useImporter } from "@/lib/importer";
import { cardsQuery, settingsQuery } from "@/lib/queries";
import { isActive, useRecorder } from "@/lib/recorder";
import { useSheet, useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";

export function Home() {
  useWindowTitle("MeetKeep");
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const settings = useQuery(settingsQuery);
  const recent = (useQuery(cardsQuery).data ?? []).slice(0, 4);
  const now = useNow();
  const status = useRecorder((s) => s.status);
  const setSheetOpen = useSheet((s) => s.setOpen);
  const sources = settings.data?.sources;
  const importing = useImporter((s) => s.importing);
  const importFile = useImport();

  useEffect(() => {
    if (search.new !== 1) return;
    setSheetOpen(true);
    void navigate({ to: "/", search: {}, replace: true });
  }, [search.new, setSheetOpen, navigate]);

  const start = () => {
    if (isActive(status) && status.meeting) {
      void navigate({ to: "/meetings/$id", params: { id: status.meeting.id } });
      return;
    }
    setSheetOpen(true);
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain">
      <PageTitle title="首页" />
      <div className="flex min-h-[300px] flex-1 flex-col items-center justify-center gap-[18px] px-6 py-10">
        <button
          type="button"
          data-testid="start-recording"
          onClick={start}
          className="inline-flex h-12 items-center justify-center gap-2.5 rounded-full bg-primary pr-7 pl-6 text-lead font-semibold text-primary-foreground transition-colors outline-none hover:bg-[#262626] focus-visible:ring-[3px] focus-visible:ring-black/15"
        >
          <span aria-hidden className="size-2.5 rounded-full bg-rec" />
          开始录制
        </button>
        <button
          type="button"
          data-testid="home-sources"
          onClick={() => setSheetOpen(true)}
          className="inline-flex h-5 items-center gap-[7px] text-ui text-subtle transition-colors outline-none hover:text-secondary-foreground"
        >
          {sources && <SourceSummary microphone={sources.microphone} system={sources.system} apps={sources.apps} />}
          <ChevronRight className="size-[13px] shrink-0 text-placeholder" />
        </button>
        <span aria-hidden className="my-1.5 h-px w-8 bg-border" />
        <button
          type="button"
          data-testid="home-import"
          disabled={importing}
          aria-busy={importing}
          onClick={() => void importFile()}
          className="inline-flex h-8 items-center gap-[7px] rounded-sm px-3 text-ui text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-50"
        >
          {importing ? (
            <LoaderCircle className="size-3.5 shrink-0 animate-spin" />
          ) : (
            <Upload className="size-3.5 shrink-0" />
          )}
          导入录音或视频
        </button>
      </div>
      {recent.length > 0 && (
        <section className="flex shrink-0 flex-col gap-2">
          <h2 className="m-0 px-1.5 text-label font-medium text-muted-foreground">最近</h2>
          <div className={cn(cardGrid, "auto-rows-[0px] grid-rows-[auto] gap-y-0 overflow-hidden")} data-testid="home-recent">
            {recent.map((card) => (
              <MeetingCard key={card.meeting.id} card={card} now={now} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
