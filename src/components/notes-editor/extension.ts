import {
  defineBaseCommands,
  defineBaseKeymap,
  defineHistory,
  defineNodeAttr,
  defineNodeSpec,
  union,
} from "prosekit/core";
import { defineBlockquote } from "prosekit/extensions/blockquote";
import { defineBold } from "prosekit/extensions/bold";
import { defineCode } from "prosekit/extensions/code";
import { defineDoc } from "prosekit/extensions/doc";
import { defineHardBreak } from "prosekit/extensions/hard-break";
import { defineHeadingCommands, defineHeadingKeymap, defineHeadingSpec } from "prosekit/extensions/heading";
import { defineHorizontalRule } from "prosekit/extensions/horizontal-rule";
import { defineTextBlockInputRule } from "prosekit/extensions/input-rule";
import { defineItalic } from "prosekit/extensions/italic";
import { defineLink } from "prosekit/extensions/link";
import { defineList } from "prosekit/extensions/list";
import { defineModClickPrevention } from "prosekit/extensions/mod-click-prevention";
import { defineParagraph } from "prosekit/extensions/paragraph";
import { defineStrike } from "prosekit/extensions/strike";
import { defineText } from "prosekit/extensions/text";
import { defineVirtualSelection } from "prosekit/extensions/virtual-selection";
import type { ProseMirrorNode } from "prosekit/pm/model";
import { defineTodos } from "@/components/notes-editor/todos";
import { clock } from "@/lib/format";

const glue = "\u2060";

function defineTimestamp() {
  return defineNodeSpec<"timestamp", { ms: number; seq: number }>({
    name: "timestamp",
    group: "inline",
    inline: true,
    atom: true,
    selectable: true,
    attrs: { ms: { default: 0 }, seq: { default: -1 } },
    leafText: () => "",
    parseDOM: [
      {
        tag: "span[data-timestamp]",
        getAttrs: (dom) => ({ ms: Number(dom.dataset.ms ?? 0), seq: Number(dom.dataset.seq ?? -1) }),
      },
    ],
    toDOM: (node) => {
      const ms = Number(node.attrs.ms);
      return [
        "span",
        {
          class: "notes-time",
          role: "button",
          "aria-label": `跳到 ${clock(ms)}`,
          "data-timestamp": "",
          "data-ms": String(ms),
          "data-seq": String(node.attrs.seq),
          "data-testid": "notes-timestamp",
        },
        clock(ms),
      ];
    },
  });
}

function assigneeLabel(node: ProseMirrorNode) {
  return [String(node.attrs.owner ?? ""), String(node.attrs.due ?? "")].filter((part) => part !== "").join(" · ");
}

function defineAssignee() {
  return defineNodeSpec<"assignee", { owner: string; due: string }>({
    name: "assignee",
    group: "inline",
    inline: true,
    atom: true,
    selectable: true,
    attrs: { owner: { default: "" }, due: { default: "" } },
    leafText: assigneeLabel,
    parseDOM: [
      {
        tag: "span[data-assignee]",
        getAttrs: (dom) => ({ owner: dom.dataset.owner ?? "", due: dom.dataset.due ?? "" }),
      },
    ],
    toDOM: (node) => [
      "span",
      { class: "notes-glue" },
      [
        "span",
        {
          class: "notes-chip",
          "data-assignee": "",
          "data-owner": String(node.attrs.owner ?? ""),
          "data-due": String(node.attrs.due ?? ""),
        },
        assigneeLabel(node),
      ],
      glue,
    ],
  });
}

function defineDoubt() {
  return defineNodeSpec<"doubt">({
    name: "doubt",
    group: "inline",
    inline: true,
    atom: true,
    selectable: true,
    leafText: () => "",
    parseDOM: [{ tag: "span[data-doubt]" }],
    toDOM: () => ["span", { class: "notes-glue" }, ["span", { class: "notes-doubt", "data-doubt": "" }, "存疑"], glue],
  });
}

export function defineNotesExtension() {
  return union(
    defineDoc(),
    defineText(),
    defineParagraph(),
    defineHeadingSpec(),
    defineHeadingCommands(),
    defineHeadingKeymap(),
    defineTextBlockInputRule({
      regex: /^(#{1,3})\s$/,
      type: "heading",
      attrs: (match) => ({ level: match[1]?.length ?? 1 }),
    }),
    defineNodeAttr<"heading", "updating", boolean>({
      type: "heading",
      attr: "updating",
      default: false,
      toDOM: (value) => (value ? ["data-updating", ""] : null),
      parseDOM: (dom) => dom.hasAttribute("data-updating"),
    }),
    defineList(),
    defineBlockquote(),
    defineHorizontalRule(),
    defineHardBreak(),
    defineBold(),
    defineItalic(),
    defineStrike(),
    defineCode(),
    defineLink(),
    defineTimestamp(),
    defineAssignee(),
    defineDoubt(),
    defineTodos(),
    defineBaseKeymap(),
    defineBaseCommands(),
    defineHistory(),
    defineVirtualSelection(),
    defineModClickPrevention(),
  );
}

export type NotesExtension = ReturnType<typeof defineNotesExtension>;

export function docText(doc: ProseMirrorNode) {
  const lines: string[] = [];
  doc.descendants((node) => {
    if (!node.isTextblock) return true;
    const line = node.textContent.trim();
    if (line !== "") lines.push(line);
    return false;
  });
  return lines.join("\n");
}
