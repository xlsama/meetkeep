import { useTranscript } from "@/lib/transcript";

export function LastLine({
  meetingId,
  speakerClassName,
  className,
}: {
  meetingId: string;
  speakerClassName: { self: string; other: string };
  className?: string;
}) {
  const { last } = useTranscript(meetingId, true);
  if (!last) return null;
  return (
    <span data-testid="last-line" data-selectable className={className}>
      <span className={last.voice.self ? speakerClassName.self : speakerClassName.other}>{last.voice.name}</span>
      {"　"}
      {last.line.text}
    </span>
  );
}
