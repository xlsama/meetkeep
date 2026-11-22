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
  const { pick } = useImport();

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
      <div className="flex min-h-[250px] flex-1 items-center justify-center px-6 py-8">
        <div className="grid w-full max-w-[620px] grid-cols-2 gap-2">
          <div className="card flex h-[200px] min-w-0 flex-col items-center justify-center gap-4 px-5">
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
              className="inline-flex h-7 max-w-full items-center gap-[7px] rounded-md px-2 text-ui text-subtle transition-colors outline-none hover:bg-hover hover:text-secondary-foreground focus-visible:bg-hover"
            >
              {sources && (
                <SourceSummary
                  microphone={sources.microphone}
                  system={sources.system}
                  apps={sources.apps}
                  className="min-w-0"
                />
              )}
              <ChevronRight className="size-[13px] shrink-0 text-placeholder" />
            </button>
          </div>
          <button
            type="button"
            data-testid="home-import"
            disabled={importing}
            aria-busy={importing}
            onClick={() => void pick()}
            className="flex h-[200px] min-w-0 cursor-pointer flex-col items-center justify-center gap-3 rounded-xl border-[1.5px] border-dashed border-faint px-5 transition-colors outline-none hover:border-placeholder hover:bg-background/60 focus-visible:border-placeholder focus-visible:bg-background/60 disabled:cursor-default disabled:opacity-60"
          >
            <span className="inline-flex size-10 items-center justify-center rounded-full bg-black/[0.06] text-secondary-foreground">
              {importing ? <LoaderCircle className="size-[18px] animate-spin" /> : <Upload className="size-[18px]" />}
            </span>
            <span className="flex flex-col items-center gap-0.5">
              <span className="text-body font-medium text-foreground">导入录音或视频</span>
              <span className="text-label text-muted-foreground">拖进窗口，或点击选择</span>
            </span>
          </button>
        </div>
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
