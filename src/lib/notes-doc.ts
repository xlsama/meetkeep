import type { Notes, NotesState, Point, Todo, Topic } from "@/mygo";

export type DocNode = {
  type: string;
  attrs?: Record<string, unknown>;
  content?: DocNode[];
  text?: string;
};

const text = (value: string): DocNode[] => (value.trim() === "" ? [] : [{ type: "text", text: value.trim() }]);
const space: DocNode = { type: "text", text: " " };
const time = (point: { atMs: number; seq: number }): DocNode => ({
  type: "timestamp",
  attrs: { ms: point.atMs, seq: point.seq },
});
const paragraph = (content: DocNode[]): DocNode => (content.length > 0 ? { type: "paragraph", content } : { type: "paragraph" });
const item = (kind: "bullet" | "task", content: DocNode[]): DocNode => ({
  type: "list",
  attrs: kind === "task" ? { kind, checked: false } : { kind },
  content: [paragraph(content)],
});
const heading = (level: number, value: string, updating = false): DocNode => ({
  type: "heading",
  attrs: { level, updating },
  content: text(value),
});

function todoItem(todo: Todo) {
  const owner = todo.owner.trim();
  const due = todo.due.trim();
  return item("task", [
    ...text(todo.text),
    space,
    ...(owner !== "" || due !== "" ? [{ type: "assignee", attrs: { owner, due } }] : []),
    time(todo),
  ]);
}

function pointItem(point: Point) {
  return item("bullet", [...text(point.text), space, ...(point.uncertain ? [{ type: "doubt" }] : []), time(point)]);
}

function topicBlocks(topic: Topic, live: boolean) {
  return [heading(2, topic.title, live && topic.updating), ...topic.points.map(pointItem)];
}

export const todosHeading = "待办";
export const mine = "我";

function todoBlocks(todos: Todo[]) {
  const own = (todo: Todo) => todo.owner.trim() === mine;
  const ordered = [...todos.filter(own), ...todos.filter((todo) => !own(todo))];
  return ordered.length > 0 ? [heading(3, todosHeading), ...ordered.map(todoItem)] : [];
}

export function notesToDoc(notes: Notes | null): DocNode {
  if (!notes) return { type: "doc", content: [paragraph([])] };
  const live = !notes.final;
  const content = live
    ? [...notes.topics.flatMap((topic) => topicBlocks(topic, true)), ...todoBlocks(notes.todos)]
    : [
        ...(notes.headline.trim() !== "" ? [heading(1, notes.headline)] : []),
        ...((notes.gist ?? "").trim() !== "" ? [paragraph(text(notes.gist))] : []),
        ...(notes.conclusions ?? []).map((point) => item("bullet", [...text(point.text), space, time(point)])),
        ...todoBlocks(notes.todos),
        ...(notes.open.length > 0
          ? [heading(3, "还没定"), ...notes.open.map((point) => item("bullet", [...text(point.text), space, time(point)]))]
          : []),
        ...notes.topics.flatMap((topic) => topicBlocks(topic, false)),
      ];
  return { type: "doc", content: content.length > 0 ? content : [paragraph([])] };
}

export function stateDoc(state: NotesState): DocNode {
  if (state.doc !== "") {
    try {
      return JSON.parse(state.doc) as DocNode;
    } catch {
      return notesToDoc(state.notes);
    }
  }
  return notesToDoc(state.notes);
}

export type DocTodo = { text: string; owner: string; due: string; checked: boolean };

export function todosIn(doc: DocNode) {
  const todos: DocTodo[] = [];
  const attr = (node: DocNode | undefined, name: string) => {
    const value = node?.attrs?.[name];
    return typeof value === "string" ? value.trim() : "";
  };
  const walk = (node: DocNode) => {
    if (node.type === "list" && node.attrs?.kind === "task") {
      const line = node.content?.[0]?.content ?? [];
      const assignee = line.find((child) => child.type === "assignee");
      todos.push({
        text: line
          .map((child) => (child.type === "text" ? (child.text ?? "") : ""))
          .join("")
          .replaceAll(/[ \u2060]+/g, " ")
          .trim(),
        owner: attr(assignee, "owner"),
        due: attr(assignee, "due"),
        checked: node.attrs.checked === true,
      });
    }
    node.content?.forEach(walk);
  };
  walk(doc);
  return todos;
}

function inline(nodes: DocNode[] | undefined): string {
  return (nodes ?? [])
    .map((node) => {
      if (node.type === "text") return node.text ?? "";
      if (node.type === "assignee") {
        const label = [node.attrs?.owner, node.attrs?.due].filter((part) => typeof part === "string" && part !== "").join(" · ");
        return label === "" ? "" : `（${label}）`;
      }
      if (node.type === "doubt") return "（存疑）";
      if (node.type === "hardBreak") return "\n";
      return "";
    })
    .join("")
    .replaceAll(/[ \u2060]+/g, " ")
    .replaceAll(" （", "（")
    .trim();
}

function block(node: DocNode): string {
  if (node.type === "heading") return `${Number(node.attrs?.level) === 1 ? "#" : "##"} ${inline(node.content)}`;
  if (node.type === "list") {
    const [first, ...rest] = node.content ?? [];
    const mark = node.attrs?.kind === "task" ? (node.attrs.checked ? "- [x]" : "- [ ]") : "-";
    const nested = rest.map((child) => block(child).replaceAll(/^/gm, "  ")).filter((line) => line.trim() !== "");
    return [`${mark} ${first ? block(first) : ""}`, ...nested].join("\n");
  }
  if (node.content?.some((child) => child.type !== "text" && child.content)) return (node.content ?? []).map(block).join("\n");
  return inline(node.content);
}

/** The head of a notes document as Markdown: what comes before its first topic. */
export function summaryMarkdown(doc: DocNode) {
  const blocks = doc.content ?? [];
  const end = blocks.findIndex((node) => node.type === "heading" && Number(node.attrs?.level) === 2);
  const lines: string[] = [];
  let previous = "";
  for (const node of end < 0 ? blocks : blocks.slice(0, end)) {
    const text = block(node);
    if (text.trim() === "") continue;
    if (lines.length > 0 && !(node.type === "list" && previous === "list")) lines.push("");
    lines.push(text);
    previous = node.type;
  }
  return lines.join("\n");
}
