import { CaptureUpdateAction, Excalidraw, FONT_FAMILY, hashElementsVersion } from "@excalidraw/excalidraw";
import type { ExcalidrawElement, OrderedExcalidrawElement } from "@excalidraw/excalidraw/element/types";
import type {
  AppState,
  ExcalidrawImperativeAPI,
  ExcalidrawInitialDataState,
  NormalizedZoomValue,
  ToolType,
  UIOptions,
} from "@excalidraw/excalidraw/types";
import "@excalidraw/excalidraw/index.css";
import { ArrowUpRight, MousePointer, Pen, Square, Type } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";
import "@/components/board-canvas.css";
import { Library, type Topic } from "@/mygo";
import { layoutBoard, type BoardLayout } from "@/lib/board";
import { isAuto, measureText, reconcile, sourceOf } from "@/lib/board-scene";
import { clock } from "@/lib/format";
import type { Jump } from "@/lib/notes";
import { cn } from "@/lib/utils";

const tools = [
  { type: "selection", label: "选择", Icon: MousePointer },
  { type: "rectangle", label: "矩形", Icon: Square },
  { type: "arrow", label: "箭头", Icon: ArrowUpRight },
  { type: "text", label: "文字", Icon: Type },
  { type: "freedraw", label: "画笔", Icon: Pen },
] as const;

const uiOptions: Partial<UIOptions> = {
  canvasActions: {
    changeViewBackgroundColor: false,
    clearCanvas: false,
    export: false,
    loadScene: false,
    saveAsImage: false,
    saveToActiveFile: false,
    toggleTheme: false,
  },
  tools: { image: false },
};

const defaults = {
  viewBackgroundColor: "transparent",
  currentItemFontFamily: FONT_FAMILY.Helvetica,
  currentItemFontSize: 16,
  currentItemRoughness: 0,
  currentItemRoundness: "round",
  currentItemStrokeColor: "#1f1f1f",
  currentItemStrokeWidth: 1.5,
  currentItemArrowType: "sharp",
} satisfies Partial<AppState>;

const Saved = z.object({
  elements: z.array(z.record(z.string(), z.unknown())),
  appState: z.object({ scrollX: z.number(), scrollY: z.number(), zoom: z.number() }).partial().default({}),
  removed: z.array(z.string()).default([]),
});

const scenes = new Map<string, string>();
const grid = 20;

function parse(raw: string) {
  try {
    const result = Saved.safeParse(JSON.parse(raw));
    return result.success ? result.data : null;
  } catch {
    return null;
  }
}

type View = {
  tool: (tool: string) => void;
  zoom: (zoom: number) => void;
  source: (key: string) => void;
};

class BoardSession {
  private api: ExcalidrawImperativeAPI | null = null;
  private layout: BoardLayout | null = null;
  private merged: BoardLayout | null = null;
  private removed = new Set<string>();
  private ready = false;
  private syncing = false;
  private key = "";
  private timer = 0;
  private pending: (() => string) | null = null;
  private grid: HTMLDivElement | null = null;

  constructor(
    private readonly meetingId: string,
    private readonly view: View,
  ) {}

  async load(): Promise<ExcalidrawInitialDataState> {
    const raw = scenes.get(this.meetingId) ?? (await Library.board(this.meetingId).catch(() => ""));
    const saved = raw === "" ? null : parse(raw);
    this.removed = new Set(saved?.removed ?? []);
    return {
      elements: (saved?.elements ?? []) as unknown as ExcalidrawElement[],
      appState: {
        ...defaults,
        scrollX: saved?.appState.scrollX ?? 28,
        scrollY: saved?.appState.scrollY ?? 66,
        zoom: { value: (saved?.appState.zoom ?? 1) as NormalizedZoomValue },
      },
    };
  }

  attach(api: ExcalidrawImperativeAPI) {
    this.api = api;
  }

  setGrid(el: HTMLDivElement | null) {
    this.grid = el;
  }

  private paintGrid(x: number, y: number, zoom: number) {
    if (!this.grid) return;
    this.grid.style.backgroundPosition = `${x * zoom}px ${y * zoom}px`;
    this.grid.style.backgroundSize = `${grid * zoom}px ${grid * zoom}px`;
  }

  setLayout(layout: BoardLayout) {
    this.layout = layout;
    this.sync();
  }

  change(elements: readonly OrderedExcalidrawElement[], state: AppState) {
    const api = this.api;
    if (!api || state.isLoading) return;
    if (state.openDialog || state.openSidebar) {
      api.updateScene({ appState: { openDialog: null, openSidebar: null }, captureUpdate: CaptureUpdateAction.NEVER });
    }
    this.view.tool(state.activeTool.type);
    this.view.zoom(state.zoom.value);
    this.paintGrid(state.scrollX, state.scrollY, state.zoom.value);
    const source = sourceOf(elements, state.selectedElementIds);
    this.view.source(source ? `${source.atMs}:${source.seq}` : "");
    for (const el of elements) {
      if (!isAuto(el)) continue;
      if (el.isDeleted) this.removed.add(el.id);
      else this.removed.delete(el.id);
    }
    if (!this.ready) {
      this.ready = true;
      this.key = this.keyOf(api);
    }
    this.sync(state);
    this.schedule();
  }

  private sync(state?: AppState) {
    const api = this.api;
    if (!api || !this.layout || !this.ready || this.syncing) return;
    const current = state ?? api.getAppState();
    if (current.editingTextElement || current.newElement || current.multiElement) return;
    const nodes = this.layout !== this.merged && !current.selectedElementsAreBeingDragged && !current.resizingElement;
    const next = reconcile(api.getSceneElementsIncludingDeleted(), this.layout, this.removed, nodes);
    if (nodes) this.merged = this.layout;
    if (!next) return;
    this.syncing = true;
    try {
      api.updateScene({ elements: next, captureUpdate: CaptureUpdateAction.NEVER });
    } finally {
      this.syncing = false;
    }
    if (!state) this.schedule();
  }

  private keyOf(api: ExcalidrawImperativeAPI) {
    const state = api.getAppState();
    return [
      hashElementsVersion(api.getSceneElements()),
      state.scrollX,
      state.scrollY,
      state.zoom.value,
      [...this.removed].join(","),
    ].join("|");
  }

  private schedule() {
    const api = this.api;
    if (!api) return;
    const key = this.keyOf(api);
    if (key === this.key) return;
    this.key = key;
    const elements = api.getSceneElements();
    const { scrollX, scrollY, zoom } = api.getAppState();
    const removed = [...this.removed];
    this.pending = () =>
      JSON.stringify({
        elements: elements.filter((el) => !el.isDeleted),
        appState: { scrollX, scrollY, zoom: zoom.value },
        removed,
      });
    window.clearTimeout(this.timer);
    this.timer = window.setTimeout(() => this.flush(), 1000);
  }

  flush() {
    window.clearTimeout(this.timer);
    const pending = this.pending;
    this.pending = null;
    if (!pending) return;
    const scene = pending();
    scenes.set(this.meetingId, scene);
    Library.saveBoard(this.meetingId, scene).catch((error: unknown) => console.error(error));
  }

  tool(type: ToolType) {
    this.api?.setActiveTool({ type: type === "image" ? "selection" : type });
  }

  zoomBy(step: number) {
    const api = this.api;
    if (!api) return;
    const state = api.getAppState();
    const zoom = state.zoom.value;
    const next = Math.min(30, Math.max(0.1, Math.round((zoom + step) * 10) / 10));
    const cx = state.width / 2;
    const cy = state.height / 2;
    api.updateScene({
      appState: {
        zoom: { value: next as NormalizedZoomValue },
        scrollX: cx / next - (cx / zoom - state.scrollX),
        scrollY: cy / next - (cy / zoom - state.scrollY),
      },
      captureUpdate: CaptureUpdateAction.NEVER,
    });
  }
}

export default function BoardCanvas({
  meetingId,
  title,
  topics,
  onJump,
}: {
  meetingId: string;
  title: string;
  topics: Topic[];
  onJump: (jump: Jump) => void;
}) {
  "use no memo";
  const [tool, setTool] = useState("selection");
  const [zoom, setZoom] = useState(1);
  const [source, setSource] = useState("");
  const [session] = useState(() => new BoardSession(meetingId, { tool: setTool, zoom: setZoom, source: setSource }));
  const [initialData] = useState(() => session.load());
  const layout = useMemo(() => layoutBoard(title, topics, { measure: measureText }), [title, topics]);

  useEffect(() => session.setLayout(layout), [session, layout]);
  useEffect(() => () => session.flush(), [session]);

  const [atMs = Number.NaN, seq = Number.NaN] = source === "" ? [] : source.split(":").map(Number);

  return (
    <>
      <div
        ref={(el) => session.setGrid(el)}
        className="absolute inset-0 bg-muted bg-[radial-gradient(#d9d9d9_1px,transparent_1px)] bg-size-[20px_20px]"
      />
      <div className="mk-board absolute inset-0 isolate">
        <Excalidraw
          excalidrawAPI={(api) => session.attach(api)}
          initialData={initialData}
          onChange={(elements, state) => session.change(elements, state)}
          UIOptions={uiOptions}
          langCode="zh-CN"
          validateEmbeddable={false}
          aiEnabled={false}
          onPaste={(data) => !data.files || Object.keys(data.files).length === 0}
        />
      </div>
      <div className="pointer-events-none absolute inset-x-0 top-3.5 z-10 flex justify-center">
        <div className="pointer-events-auto relative">
          <div
            role="toolbar"
            aria-label="画板工具"
            className="flex gap-0.5 rounded-lg border border-border bg-background p-[3px] shadow-[0_4px_14px_rgba(0,0,0,0.08)]"
          >
            {tools.map(({ type, label, Icon }) => (
              <button
                key={type}
                type="button"
                aria-label={label}
                aria-pressed={tool === type}
                data-testid={`board-tool-${type}`}
                onClick={() => session.tool(type)}
                className={cn(
                  "inline-flex size-[30px] items-center justify-center rounded-[9px] transition-colors",
                  tool === type ? "bg-accent text-foreground" : "text-subtle hover:bg-hover hover:text-foreground",
                )}
              >
                <Icon className="size-[15px]" />
              </button>
            ))}
          </div>
          {source !== "" && (
            <button
              type="button"
              aria-label={`跳到 ${clock(atMs)}`}
              data-testid="board-source"
              onClick={() => onJump({ seq, atMs })}
              className="absolute inset-y-0 left-full ml-1.5 inline-flex items-center rounded-lg border border-border bg-background px-2.5 font-mono text-tiny text-subtle tabular-nums shadow-[0_4px_14px_rgba(0,0,0,0.08)] transition-colors hover:text-foreground"
            >
              {clock(atMs)}
            </button>
          )}
        </div>
      </div>
      <div className="absolute bottom-3.5 left-3.5 z-10 flex items-center gap-0.5 rounded-md border border-border bg-background p-0.5">
        <button
          type="button"
          aria-label="缩小"
          data-testid="board-zoom-out"
          onClick={() => session.zoomBy(-0.1)}
          className="size-[26px] rounded-sm text-[15px] text-subtle transition-colors hover:bg-hover hover:text-foreground"
        >
          −
        </button>
        <span className="px-1 text-meta text-secondary-foreground tabular-nums">{Math.round(zoom * 100)}%</span>
        <button
          type="button"
          aria-label="放大"
          data-testid="board-zoom-in"
          onClick={() => session.zoomBy(0.1)}
          className="size-[26px] rounded-sm text-[15px] text-subtle transition-colors hover:bg-hover hover:text-foreground"
        >
          +
        </button>
      </div>
    </>
  );
}
