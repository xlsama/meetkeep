import { FONT_FAMILY, newElementWith, restoreElements, ROUNDNESS } from "@excalidraw/excalidraw";
import type { ExcalidrawElement, ExcalidrawTextElement } from "@excalidraw/excalidraw/element/types";
import { edgePoints, estimateWidth, type BoardEdge, type BoardLayout, type BoardNode, type Box, type Point } from "@/lib/board";

export const fontSize = 13;
const lineHeight = 1.15;
const font = `${fontSize}px Helvetica, sans-serif, Segoe UI Emoji`;

let context: CanvasRenderingContext2D | null | undefined;

export function measureText(text: string) {
  context ??= document.createElement("canvas").getContext("2d");
  if (!context) return estimateWidth(text);
  context.font = font;
  return context.measureText(text).width;
}

type Gen = { x: number; y: number; w: number; h: number; text: string };
type NodeMeta = { mk: "node"; node: string; gen: Gen; atMs: number | null; seq: number | null };
type EdgeMeta = { mk: "edge"; from: string; to: string; gen: Point[] };

export const nodeKey = (id: string) => `mk-${id}`;
const textKey = (id: string) => `mk-${id}-text`;
const edgeKey = (id: string) => `mk-edge-${id}`;

export function nodeMeta(el: ExcalidrawElement | undefined): NodeMeta | null {
  const data = el?.customData;
  if (!el || data?.mk !== "node" || typeof data.node !== "string" || el.id !== nodeKey(data.node)) return null;
  return data as NodeMeta;
}

function edgeMeta(el: ExcalidrawElement | undefined): EdgeMeta | null {
  const data = el?.customData;
  if (!el || el.type !== "line" || data?.mk !== "edge" || typeof data.to !== "string" || el.id !== edgeKey(data.to)) return null;
  return data as EdgeMeta;
}

export function isAuto(el: ExcalidrawElement) {
  return nodeMeta(el) !== null || edgeMeta(el) !== null;
}

const looks = {
  root: { bg: "#0d0d0d", stroke: "#0d0d0d", width: 1, text: "#ffffff" },
  topic: { bg: "#ffffff", stroke: "#0d0d0d", width: 1.5, text: "#0d0d0d" },
  phrase: { bg: "#ffffff", stroke: "#e3e3e3", width: 1, text: "#0d0d0d" },
  doubt: { bg: "#ffffff", stroke: "#c4c4c4", width: 1, text: "#5d5d5d" },
};

function look(node: BoardNode) {
  const style = node.uncertain ? looks.doubt : looks[node.kind];
  return {
    strokeColor: style.stroke,
    backgroundColor: style.bg,
    strokeWidth: style.width,
    strokeStyle: node.uncertain ? ("dashed" as const) : ("solid" as const),
    text: style.text,
  };
}

function textBox(node: BoardNode) {
  const width = measureText(node.text);
  const height = fontSize * lineHeight;
  return { x: node.x + (node.width - width) / 2, y: node.y + (node.height - height) / 2, width, height };
}

function meta(node: BoardNode): NodeMeta {
  return {
    mk: "node",
    node: node.id,
    gen: { x: node.x, y: node.y, w: node.width, h: node.height, text: node.text },
    atMs: node.atMs,
    seq: node.seq,
  };
}

function restore(elements: object[]) {
  return restoreElements(elements as unknown as ExcalidrawElement[], null);
}

function nodeElements(node: BoardNode) {
  const { text, ...style } = look(node);
  return restore([
    {
      id: nodeKey(node.id),
      type: "rectangle",
      x: node.x,
      y: node.y,
      width: node.width,
      height: node.height,
      ...style,
      fillStyle: "solid",
      roughness: 0,
      roundness: { type: ROUNDNESS.ADAPTIVE_RADIUS, value: 10 },
      boundElements: [{ type: "text", id: textKey(node.id) }],
      customData: meta(node),
    },
    {
      id: textKey(node.id),
      type: "text",
      ...textBox(node),
      strokeColor: text,
      backgroundColor: "transparent",
      roughness: 0,
      fontSize,
      fontFamily: FONT_FAMILY.Helvetica,
      lineHeight,
      text: node.text,
      originalText: node.text,
      textAlign: "center",
      verticalAlign: "middle",
      containerId: nodeKey(node.id),
      autoResize: true,
      customData: { mk: "text", node: node.id },
    },
  ]);
}

function edgeElement(edge: BoardEdge, points: Point[]) {
  const [x0, y0] = points[0] ?? [0, 0];
  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const data: EdgeMeta = { mk: "edge", from: edge.from, to: edge.to, gen: points };
  const [element] = restore([
    {
      id: edgeKey(edge.to),
      type: "line",
      x: x0,
      y: y0,
      width: Math.max(...xs) - Math.min(...xs),
      height: Math.max(...ys) - Math.min(...ys),
      points: points.map(([x, y]) => [x - x0, y - y0]),
      strokeColor: edge.kind === "trunk" ? "#b5b5b5" : "#d4d4d4",
      backgroundColor: "transparent",
      strokeWidth: 1.5,
      roughness: 0,
      roundness: null,
      locked: true,
      startArrowhead: null,
      endArrowhead: null,
      customData: data,
    },
  ]);
  return element;
}

const near = (a: number, b: number) => Math.abs(a - b) < 0.01;

function untouchedNode(container: ExcalidrawElement, text: ExcalidrawTextElement) {
  const gen = nodeMeta(container)?.gen;
  return (
    gen !== undefined &&
    near(container.x, gen.x) &&
    near(container.y, gen.y) &&
    near(container.width, gen.w) &&
    near(container.height, gen.h) &&
    container.angle === 0 &&
    !text.isDeleted &&
    text.originalText === gen.text
  );
}

function absolute(el: ExcalidrawElement): Point[] {
  return "points" in el ? el.points.map(([x, y]): Point => [el.x + x, el.y + y]) : [];
}

function samePoints(a: Point[], b: Point[]) {
  return a.length === b.length && a.every(([x, y], i) => near(x, b[i]?.[0] ?? Number.NaN) && near(y, b[i]?.[1] ?? Number.NaN));
}

function untouchedEdge(el: ExcalidrawElement) {
  const gen = edgeMeta(el)?.gen;
  return gen !== undefined && el.angle === 0 && samePoints(absolute(el), gen);
}

function updateNode(container: ExcalidrawElement, text: ExcalidrawTextElement, node: BoardNode) {
  const current = nodeMeta(container);
  const style = look(node);
  const same =
    current !== null &&
    near(container.x, node.x) &&
    near(container.y, node.y) &&
    near(container.width, node.width) &&
    near(container.height, node.height) &&
    text.originalText === node.text &&
    container.strokeColor === style.strokeColor &&
    container.strokeStyle === style.strokeStyle &&
    current.atMs === node.atMs &&
    current.seq === node.seq;
  if (same) return null;
  const { text: color, ...box } = style;
  return [
    newElementWith(container, {
      x: node.x,
      y: node.y,
      width: node.width,
      height: node.height,
      ...box,
      customData: meta(node),
    }),
    newElementWith(text, { ...textBox(node), text: node.text, originalText: node.text, strokeColor: color }),
  ] as const;
}

function syncEdges(list: readonly ExcalidrawElement[], layout: BoardLayout, removed: ReadonlySet<string>) {
  const byId = new Map(list.map((el) => [el.id, el]));
  const boxes = new Map<string, Box>();
  for (const el of list) {
    const data = nodeMeta(el);
    if (data && !el.isDeleted) boxes.set(data.node, el);
  }
  const wanted = new Set<string>();
  const replace = new Map<string, ExcalidrawElement>();
  const drop = new Set<string>();
  const added: ExcalidrawElement[] = [];

  for (const edge of layout.edges) {
    const key = edgeKey(edge.to);
    wanted.add(key);
    const el = byId.get(key);
    if (el?.isDeleted || removed.has(key)) continue;
    const from = boxes.get(edge.from);
    const to = boxes.get(edge.to);
    if (!from || !to) {
      if (el && untouchedEdge(el)) drop.add(key);
      continue;
    }
    const points = edgePoints(from, to, edge.kind);
    if (!el) {
      const created = edgeElement(edge, points);
      if (created) added.push(created);
    } else if (untouchedEdge(el) && !samePoints(edgeMeta(el)?.gen ?? [], points)) {
      const fresh = edgeElement(edge, points);
      if (fresh) replace.set(key, { ...fresh, index: el.index, version: el.version + 1, versionNonce: el.versionNonce + 1 });
    }
  }
  for (const el of list) {
    if (edgeMeta(el) && !el.isDeleted && !wanted.has(el.id) && untouchedEdge(el)) drop.add(el.id);
  }
  if (replace.size === 0 && drop.size === 0 && added.length === 0) return null;
  return [...added, ...list.filter((el) => !drop.has(el.id)).map((el) => replace.get(el.id) ?? el)];
}

export function reconcile(
  all: readonly ExcalidrawElement[],
  layout: BoardLayout,
  removed: ReadonlySet<string>,
  nodes: boolean,
): ExcalidrawElement[] | null {
  let list: readonly ExcalidrawElement[] = all;
  let changed = false;

  if (nodes) {
    const byId = new Map(all.map((el) => [el.id, el]));
    const wanted = new Set(layout.nodes.map((node) => nodeKey(node.id)));
    const replace = new Map<string, ExcalidrawElement>();
    const drop = new Set<string>();
    const added: ExcalidrawElement[] = [];

    for (const node of layout.nodes) {
      const key = nodeKey(node.id);
      if (removed.has(key)) continue;
      const container = byId.get(key);
      if (!container) {
        added.push(...nodeElements(node));
        continue;
      }
      const text = byId.get(textKey(node.id));
      if (container.isDeleted || text?.type !== "text" || !untouchedNode(container, text)) continue;
      const next = updateNode(container, text, node);
      if (next) {
        replace.set(container.id, next[0]);
        replace.set(text.id, next[1]);
      }
    }
    for (const el of all) {
      const data = nodeMeta(el);
      if (!data || el.isDeleted || wanted.has(el.id)) continue;
      const text = byId.get(textKey(data.node));
      if (text?.type === "text" && untouchedNode(el, text)) {
        drop.add(el.id);
        drop.add(text.id);
      }
    }
    if (replace.size > 0 || drop.size > 0 || added.length > 0) {
      list = [...all.filter((el) => !drop.has(el.id)).map((el) => replace.get(el.id) ?? el), ...added];
      changed = true;
    }
  }

  const edges = syncEdges(list, layout, removed);
  if (edges) return edges;
  return changed ? [...list] : null;
}

export function sourceOf(all: readonly ExcalidrawElement[], selected: Readonly<Record<string, true>>) {
  const ids = Object.keys(selected);
  if (ids.length === 0) return null;
  const byId = new Map(all.map((el) => [el.id, el]));
  let found: NodeMeta | null = null;
  for (const id of ids) {
    const el = byId.get(id);
    const data = nodeMeta(el) ?? (el?.type === "text" && el.containerId ? nodeMeta(byId.get(el.containerId)) : null);
    if (!data || (found && found.node !== data.node)) return null;
    found = data;
  }
  return found && found.atMs !== null ? { atMs: found.atMs, seq: found.seq ?? -1 } : null;
}
