import { useSearch } from "@tanstack/react-router";
import { LoaderCircle, Mic, X } from "lucide-react";
import { useState } from "react";
import { Recorder, Windows } from "@/mygo";
import { errorMessage, queryClient, settingsQuery } from "@/lib/queries";

export function DetectPage() {
  const { app = "", name = "" } = useSearch({ strict: false });
  const [busy, setBusy] = useState<"start" | "ignore" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const run = async (kind: "start" | "ignore", action: () => Promise<unknown>) => {
    setBusy(kind);
    setError(null);
    try {
      await action();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(null);
    }
  };

  const start = () =>
    run("start", async () => {
      const settings = await queryClient.fetchQuery(settingsQuery);
      await Recorder.start({
        title: "",
        microphone: true,
        system: true,
        apps: [{ id: app, name }],
        showPanel: settings.recording.showPanel,
      });
      await Windows.closeNotice();
    });

  return (
    <div className="flex h-full w-full flex-col justify-between rounded-3xl border-[0.5px] border-black/12 bg-background py-3.5 pr-3.5 pl-4 leading-normal">
      <div className="flex items-center gap-3">
        <span
          aria-hidden
          className="inline-flex size-[38px] shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground"
        >
          <Mic className="size-[18px]" />
        </span>
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-row font-semibold" data-testid="detect-title">
            {name}正在开会
          </span>
          {error ? (
            <span role="alert" title={error} className="truncate text-label text-destructive" data-testid="detect-error">
              {error}
            </span>
          ) : (
            <span className="truncate text-label text-subtle">录制麦克风和{name}的声音</span>
          )}
        </div>
        <button
          type="button"
          aria-label="关闭"
          data-testid="detect-close"
          onClick={() => void Windows.closeNotice()}
          className="inline-flex size-[30px] shrink-0 items-center justify-center rounded-sm text-subtle transition-colors hover:bg-hover hover:text-foreground"
        >
          <X className="size-4" />
        </button>
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          data-testid="detect-start"
          disabled={busy !== null}
          aria-busy={busy === "start"}
          onClick={() => void start()}
          className="inline-flex h-[34px] flex-1 items-center justify-center gap-2 rounded-full bg-primary text-ui font-semibold text-primary-foreground transition-colors hover:bg-[#262626] disabled:opacity-70"
        >
          {busy === "start" ? (
            <LoaderCircle className="size-[13px] animate-spin" />
          ) : (
            <span aria-hidden className="size-2 rounded-full bg-rec" />
          )}
          开始录制
        </button>
        <button
          type="button"
          data-testid="detect-ignore"
          disabled={busy !== null}
          onClick={() => void run("ignore", () => Windows.ignoreApp({ id: app, name }))}
          className="h-[34px] rounded-full px-3 text-ui text-secondary-foreground transition-colors hover:bg-hover disabled:opacity-50"
        >
          这个 App 不再提醒
        </button>
      </div>
    </div>
  );
}
