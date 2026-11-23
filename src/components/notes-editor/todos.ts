import { definePlugin } from "prosekit/core";
import type { ProseMirrorNode } from "prosekit/pm/model";
import { Plugin, PluginKey } from "prosekit/pm/state";
import { Decoration, DecorationSet, type EditorView } from "prosekit/pm/view";
import { mine, todosHeading } from "@/lib/notes-doc";

const key = new PluginKey<boolean>("notes-todos");
const shown = 6;
const named = 3;

function ownerOf(node: ProseMirrorNode) {
  let owner = "";
  node.descendants((child) => {
    if (child.type.name === "assignee") owner = String(child.attrs.owner ?? "").trim();
    return owner === "";
  });
  return owner;
}

function toggle(open: boolean, label: string) {
  return (view: EditorView) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "notes-fold";
    button.dataset.testid = "notes-todos-fold";
    button.contentEditable = "false";
    button.textContent = label;
    button.addEventListener("mousedown", (event) => event.preventDefault());
    button.addEventListener("click", () => view.dispatch(view.state.tr.setMeta(key, !open)));
    return button;
  };
}

function decorate(doc: ProseMirrorNode, open: boolean) {
  const decorations: Decoration[] = [];
  const rest: { from: number; to: number; owner: string }[] = [];
  let stage: "before" | "inside" | "after" = "before";
  let count = 0;
  let foldAt = -1;
  doc.forEach((node, offset) => {
    if (stage === "before") {
      if (node.type.name === "heading" && node.textContent.trim() === todosHeading) stage = "inside";
      return;
    }
    if (stage === "after") return;
    if (node.type.name !== "list" || node.attrs.kind !== "task") {
      stage = "after";
      return;
    }
    count += 1;
    const end = offset + node.nodeSize;
    const owner = ownerOf(node);
    if (owner === mine) decorations.push(Decoration.node(offset, end, { class: "notes-mine" }));
    if (count === shown) foldAt = end;
    if (count > shown) rest.push({ from: offset, to: end, owner });
  });
  const last = rest.at(-1);
  if (last) {
    if (!open) for (const todo of rest) decorations.push(Decoration.node(todo.from, todo.to, { class: "notes-folded" }));
    const owners = [...new Set(rest.map((todo) => todo.owner).filter((owner) => owner !== ""))];
    const names = owners.length === 0 ? "" : `（${owners.slice(0, named).join("、")}${owners.length > named ? "等" : ""}）`;
    const label = open ? "收起" : `展开其余 ${rest.length} 条${names}`;
    decorations.push(
      Decoration.widget(open ? last.to : foldAt, toggle(open, label), { side: open ? 1 : -1, key: `${open}:${label}`, ignoreSelection: true }),
    );
  }
  return DecorationSet.create(doc, decorations);
}

/** Marks the user's own todos and folds those past the first few. */
export function defineTodos() {
  return definePlugin(
    new Plugin<boolean>({
      key,
      state: {
        init: () => false,
        apply: (tr, open) => {
          const meta: unknown = tr.getMeta(key);
          return typeof meta === "boolean" ? meta : open;
        },
      },
      props: {
        decorations: (state) => decorate(state.doc, key.getState(state) ?? false),
      },
    }),
  );
}
