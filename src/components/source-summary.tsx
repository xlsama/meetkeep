import { Mic, Volume2 } from "lucide-react";
import { Fragment } from "react";
import type { AppRef } from "@/mygo";
import { systemLabel } from "@/lib/format";
import { cn } from "@/lib/utils";

export function SourceSummary({
  microphone,
  system,
  apps,
  iconSize = 14,
  className,
}: {
  microphone: boolean;
  system: boolean;
  apps: AppRef[];
  iconSize?: number;
  className?: string;
}) {
  const parts = [
    microphone && { key: "mic", icon: Mic, label: "麦克风" },
    system && { key: "system", icon: Volume2, label: systemLabel(apps) },
  ].filter((part) => part !== false);
  return (
    <span className={cn("inline-flex min-w-0 items-center gap-[7px]", className)}>
      {parts.map((part, index) => (
        <Fragment key={part.key}>
          {index > 0 && <span className="text-faint">+</span>}
          <part.icon className="shrink-0" size={iconSize} />
          <span className="truncate">{part.label}</span>
        </Fragment>
      ))}
    </span>
  );
}
