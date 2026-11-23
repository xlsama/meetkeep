const boundary = /[^a-z0-9]/;

function score(text: string, term: string): number | null {
  const at = text.indexOf(term);
  if (at >= 0) return 1000 - at + (at === 0 || boundary.test(text[at - 1] ?? "") ? 200 : 0);
  let first = -1;
  let from = 0;
  for (const char of term) {
    const found = text.indexOf(char, from);
    if (found < 0) return null;
    if (first < 0) first = found;
    from = found + 1;
  }
  return 500 - (from - first - term.length) - first;
}

export function fuzzyFilter(items: string[], query: string): string[] {
  const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (terms.length === 0) return items;
  const matches: { item: string; score: number; index: number }[] = [];
  for (const [index, item] of items.entries()) {
    const text = item.toLowerCase();
    let total = 0;
    for (const term of terms) {
      const s = score(text, term);
      if (s === null) {
        total = Number.NaN;
        break;
      }
      total += s;
    }
    if (!Number.isNaN(total)) matches.push({ item, score: total, index });
  }
  return matches.sort((a, b) => b.score - a.score || a.index - b.index).map((m) => m.item);
}
