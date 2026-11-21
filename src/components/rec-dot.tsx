import { cn } from "@/lib/utils";

export function RecDot({
  paused = false,
  color = "bg-rec-strong",
  className,
}: {
  paused?: boolean;
  color?: string;
  className?: string;
}) {
  return (
    <span
      aria-hidden
      className={cn("size-[7px] shrink-0 rounded-full", className, paused ? "bg-ink-muted" : ["animate-pulse-dot", color])}
    />
  );
}
