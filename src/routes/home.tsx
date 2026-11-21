import { useQuery } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { ChevronRight, Upload } from "lucide-react";
import { useEffect } from "react";
import { SourceSummary } from "@/components/source-summary";
import { settingsQuery } from "@/lib/queries";
import { isActive, useRecorder } from "@/lib/recorder";
import { useSheet, useWindowTitle } from "@/lib/ui";

export function Home() {
  useWindowTitle("MeetKeep");
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const settings = useQuery(settingsQuery);
  const status = useRecorder((s) => s.status);
  const setSheetOpen = useSheet((s) => s.setOpen);
  const sources = settings.data?.sources;

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
    <div className="flex flex-1 items-center justify-center px-6 py-10">
      <div className="flex flex-col items-center gap-[18px]">
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
          disabled
          data-testid="import"
          className="inline-flex h-8 items-center gap-[7px] rounded-sm px-3 text-ui text-secondary-foreground disabled:opacity-50"
        >
          <Upload className="size-3.5 shrink-0" />
          导入录音或视频
        </button>
      </div>
    </div>
  );
}
