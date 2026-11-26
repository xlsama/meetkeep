import type { ReactNode } from "react";
import { useSample } from "@/lib/sample";
import { cn } from "@/lib/utils";

export function SampleButton({
  sampleKey,
  label,
  testId,
  size,
  disabled,
  onClick,
  children,
}: {
  sampleKey: string;
  label: string;
  testId: string;
  size: number;
  disabled?: boolean;
  onClick: () => void;
  children?: ReactNode;
}) {
  const playing = useSample((s) => s.playing === sampleKey);
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={playing}
      data-testid={testId}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "press inline-flex shrink-0 items-center justify-center rounded-full border border-input bg-background text-secondary-foreground outline-none focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-40",
        children
          ? "h-7 gap-1.5 pr-2.5 pl-2 text-label tabular-nums hover:border-border-strong hover:text-foreground aria-pressed:border-primary aria-pressed:bg-primary aria-pressed:text-primary-foreground"
          : "size-[30px] hover:bg-hover",
      )}
    >
      <svg aria-hidden width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
        {playing ? <rect x="6" y="6" width="12" height="12" rx="1.5" /> : <path d="M6 4l14 8-14 8z" />}
      </svg>
      {children}
    </button>
  );
}
