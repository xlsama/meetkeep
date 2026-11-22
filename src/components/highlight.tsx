export function Highlight({ text, query }: { text: string; query: string }) {
  if (query === "") return text;
  const lower = text.toLowerCase();
  const needle = query.toLowerCase();
  const parts: { text: string; mark: boolean; at: number }[] = [];
  let from = 0;
  for (let at = lower.indexOf(needle); at !== -1; at = lower.indexOf(needle, from)) {
    if (at > from) parts.push({ text: text.slice(from, at), mark: false, at: from });
    parts.push({ text: text.slice(at, at + needle.length), mark: true, at });
    from = at + needle.length;
  }
  if (from < text.length) parts.push({ text: text.slice(from), mark: false, at: from });
  return parts.map((part) =>
    part.mark ? (
      <mark key={part.at} className="rounded-[3px] bg-mark px-px text-inherit">
        {part.text}
      </mark>
    ) : (
      <span key={part.at}>{part.text}</span>
    ),
  );
}
