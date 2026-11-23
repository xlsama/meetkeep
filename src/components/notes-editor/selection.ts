import { definePlugin } from "prosekit/core";
import { Plugin } from "prosekit/pm/state";

const name = "notes-selection";

/**
 * Paints the selection over the text it covers alone. WebKit fills the lines of a selection to the edges of the
 * document and leaves the list markers out, as holes: the style sheet hides its painting where this one is at work.
 */
export function defineSelectionHighlight() {
  return definePlugin(
    new Plugin({
      view: (view) => {
        if (!("highlights" in CSS)) return {};
        const highlight = new Highlight();
        CSS.highlights.set(name, highlight);
        const paint = () => {
          highlight.clear();
          const selection = document.getSelection();
          if (!selection || selection.isCollapsed || selection.rangeCount === 0) return;
          const range = selection.getRangeAt(0);
          if (view.dom.contains(range.commonAncestorContainer)) highlight.add(new StaticRange(range));
        };
        document.addEventListener("selectionchange", paint);
        return {
          update: paint,
          destroy: () => {
            document.removeEventListener("selectionchange", paint);
            CSS.highlights.delete(name);
          },
        };
      },
    }),
  );
}
