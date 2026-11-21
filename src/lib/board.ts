import type { Topic } from "@/mygo";

export type NodeKind = "root" | "topic" | "phrase";

export type BoardNode = {
  id: string;
  kind: NodeKind;
  text: string;
  x: number;
  y: number;
  width: number;
  height: number;
  uncertain: boolean;
  seq: number | null;
  atMs: number | null;
};

export type BoardEdge = { from: string; to: string; kind: "trunk" | "branch" };

export type BoardLayout = { nodes: BoardNode[]; edges: BoardEdge[]; width: number; height: number };

export type Box = { x: number; y: number; width: number; height: number };

export type Point = [x: number, y: number];

export const nodeHeight: Record<NodeKind, number> = { root: 33, topic: 36, phrase: 35 };
const nodeBorder: Record<NodeKind, number> = { root: 0, topic: 1.5, phrase: 1 };
const padX = 11;
const topicX = 40;
const phraseX = 32;
const trunkX = 18;
const trunkGap = 14;
const branchX = 14;
const branchGap = 8;
const phraseGap = 6;
const tickGap = 6;

export function estimateWidth(text: string) {
  let width = 0;
  for (const char of text) width += char.charCodeAt(0) > 0x2e80 ? 13 : 7.2;
  return width;
}

export function phraseId(topicId: string, index: number) {
  return `${topicId}.${index}`;
}

export function layoutBoard(
  title: string,
  topics: Topic[],
  { measure = estimateWidth, topicGap = 14 }: { measure?: (text: string) => number; topicGap?: number } = {},
): BoardLayout {
  const node = (
    id: string,
    kind: NodeKind,
    text: string,
    x: number,
    y: number,
    source: { seq: number; atMs: number } | null,
    uncertain = false,
  ): BoardNode => ({
    id,
    kind,
    text,
    x,
    y,
    width: Math.ceil(measure(text)) + padX * 2 + nodeBorder[kind] * 2,
    height: nodeHeight[kind],
    uncertain,
    seq: source?.seq ?? null,
    atMs: source?.atMs ?? null,
  });

  const root = node("root", "root", title, 0, 0, null);
  const nodes: BoardNode[] = [root];
  const edges: BoardEdge[] = [];
  let y = root.height + trunkGap + 2;

  topics.forEach((topic, index) => {
    if (index > 0) y += topicGap;
    const first = topic.points[0];
    const head = node(topic.id, "topic", topic.title, topicX, y, {
      seq: first?.seq ?? -1,
      atMs: topic.atMs,
    });
    nodes.push(head);
    edges.push({ from: root.id, to: head.id, kind: "trunk" });
    y += head.height;
    topic.nodes.forEach((phrase, i) => {
      y += i === 0 ? branchGap : phraseGap;
      const leaf = node(phraseId(topic.id, i), "phrase", phrase.text, topicX + phraseX, y, phrase, phrase.uncertain);
      nodes.push(leaf);
      edges.push({ from: head.id, to: leaf.id, kind: "branch" });
      y += leaf.height;
    });
  });

  const width = Math.max(...nodes.map((n) => n.x + n.width));
  return { nodes, edges, width, height: y };
}

export function edgePoints(parent: Box, child: Box, kind: BoardEdge["kind"]): Point[] {
  const x = parent.x + (kind === "trunk" ? trunkX : branchX);
  const top = parent.y + parent.height + (kind === "trunk" ? trunkGap : branchGap);
  const middle = child.y + child.height / 2;
  if (kind === "branch" && Math.abs(child.x - (x + phraseX - branchX)) < 0.5 && child.y >= top) {
    return [
      [x, top],
      [x, child.y + child.height],
    ];
  }
  return [
    [x, top],
    [x, middle],
    [child.x - tickGap, middle],
  ];
}
