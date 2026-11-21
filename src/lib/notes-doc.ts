import type { Notes, Point, Todo, Topic } from "@/mygo";

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

function todoBlocks(todos: Todo[]) {
  return todos.length > 0 ? [heading(3, "待办"), ...todos.map(todoItem)] : [];
}

export function notesToDoc(notes: Notes | null): DocNode {
  if (!notes) return { type: "doc", content: [paragraph([])] };
  const live = !notes.final;
  const content = live
    ? [...notes.topics.flatMap((topic) => topicBlocks(topic, true)), ...todoBlocks(notes.todos)]
    : [
        ...(notes.headline.trim() !== "" ? [heading(1, notes.headline)] : []),
        ...todoBlocks(notes.todos),
        ...(notes.open.length > 0
          ? [heading(3, "还没定"), ...notes.open.map((point) => item("bullet", [...text(point.text), space, time(point)]))]
          : []),
        ...notes.topics.flatMap((topic) => topicBlocks(topic, false)),
      ];
  return { type: "doc", content: content.length > 0 ? content : [paragraph([])] };
}
