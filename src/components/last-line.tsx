import type { Track } from "@/mygo";
import { speaker } from "@/lib/format";
import { useTranscript } from "@/lib/transcript";

export function LastLine({
  meetingId,
  speakerClassName,
  className,
}: {
  meetingId: string;
  speakerClassName: Record<Track, string>;
  className?: string;
}) {
  const { last } = useTranscript(meetingId, true);
  if (!last) return null;
  return (
    <span data-testid="last-line" data-selectable className={className}>
      <span className={speakerClassName[last.track]}>{speaker[last.track]}</span>
      {"　"}
      {last.text}
    </span>
  );
}
