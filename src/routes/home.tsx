import { useQuery } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { ChevronRight, LoaderCircle, Upload } from "lucide-react";
import { useEffect, useState } from "react";
import { SourceSummary } from "@/components/source-summary";
import { Library } from "@/mygo";
import { errorMessage, settingsQuery } from "@/lib/queries";
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
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

  const importFile = async () => {
    setImporting(true);
    setError(null);
    try {
      const meeting = await Library.import();
      if (meeting) void navigate({ to: "/meetings/$id", params: { id: meeting.id } });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setImporting(false);
    }
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
          data-testid="import"
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
        {error && (
          <p role="alert" data-testid="import-error" data-selectable className="m-0 max-w-[360px] text-center text-label break-words text-destructive">
            {error}
          </p>
        )}
      </div>
    </div>
  );
}
