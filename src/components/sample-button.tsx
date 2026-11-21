import { useSample } from "@/lib/sample";

export function SampleButton({
  sampleKey,
  label,
  testId,
  size,
  disabled,
  onClick,
}: {
  sampleKey: string;
  label: string;
  testId: string;
  size: number;
  disabled?: boolean;
  onClick: () => void;
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
      className="inline-flex size-[30px] shrink-0 items-center justify-center rounded-full border border-input bg-background text-secondary-foreground transition-colors outline-none hover:bg-hover focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-40"
    >
      <svg aria-hidden width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
        {playing ? <rect x="6" y="6" width="12" height="12" rx="1.5" /> : <path d="M6 4l14 8-14 8z" />}
      </svg>
    </button>
  );
}
