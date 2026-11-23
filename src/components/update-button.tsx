import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Updates } from "@/mygo";
import { updateQuery } from "@/lib/queries";
import { useRecorder } from "@/lib/recorder";

export function UpdateButton() {
  const version = useQuery(updateQuery).data;
  const recording = useRecorder((s) => s.status.phase !== "idle");
  const [restarting, setRestarting] = useState(false);
  if (!version || recording) return null;
  return (
    <button
      type="button"
      data-testid="update"
      title={`更新到 ${version} 并重启`}
      disabled={restarting}
      className="no-drag inline-flex h-6 shrink-0 items-center rounded-full bg-info px-2.5 text-label font-semibold text-white transition-colors outline-none hover:bg-[#1e40af] focus-visible:ring-[3px] focus-visible:ring-info/25 disabled:opacity-60"
      onClick={() => {
        setRestarting(true);
        Updates.restart().catch(() => setRestarting(false));
      }}
    >
      更新
    </button>
  );
}
