import { initial } from "@/lib/speakers";
import { cn } from "@/lib/utils";

export function Avatar({ name, tone, className }: { name: string; tone: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white",
        tone,
        className,
      )}
    >
      {initial(name)}
    </span>
  );
}
