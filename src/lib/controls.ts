import { useState } from "react";
import { Recorder } from "@/mygo";
import { errorMessage } from "@/lib/queries";
import { useRecorder } from "@/lib/recorder";

export function useControls() {
  const status = useRecorder((s) => s.status);
  const [busy, setBusy] = useState<"pause" | "stop" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const run = async (kind: "pause" | "stop", action: () => Promise<unknown>) => {
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

  const paused = status.phase === "paused";
  const stopping = status.phase === "stopping" || busy === "stop";

  return {
    status,
    paused,
    stopping,
    pausing: busy === "pause",
    error,
    clearError: () => setError(null),
    togglePause: () => run("pause", () => (paused ? Recorder.resume() : Recorder.pause())),
    stop: () => run("stop", () => Recorder.stop()),
  };
}
