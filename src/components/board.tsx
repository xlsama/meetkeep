import { AppWindow } from "lucide-react";
import { lazy, Suspense, useRef } from "react";
import type { Topic } from "@/mygo";
import { edgePoints, layoutBoard, type BoardNode } from "@/lib/board";
import type { Jump } from "@/lib/notes";
import { cn } from "@/lib/utils";

const BoardCanvas = lazy(() => import("@/components/board-canvas"));

export function BoardPane({
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
  return (
    <div
      data-testid="board"
      className="relative min-h-0 flex-1 overflow-hidden bg-muted bg-[radial-gradient(#d9d9d9_1px,transparent_1px)] bg-size-[20px_20px]"
    >
      <Suspense fallback={null}>
        <BoardCanvas key={meetingId} meetingId={meetingId} title={title} topics={topics} onJump={onJump} />
      </Suspense>
    </div>
  );
}

const padX = 18;
const padY = 20;

const nodeStyle: Record<BoardNode["kind"], string> = {
  root: "bg-primary font-semibold text-primary-foreground",
  topic: "border-[1.5px] border-foreground bg-background font-semibold",
  phrase: "border border-input bg-background",
};

export function BoardPreview({ title, topics, onOpen }: { title: string; topics: Topic[]; onOpen: () => void }) {
  const layout = layoutBoard(title, topics, { topicGap: 16 });
  const byId = new Map(layout.nodes.map((node) => [node.id, node]));
  const drag = useRef<{ x: number; y: number; left: number; top: number } | null>(null);

  return (
    <div className="relative min-h-0 flex-1">
      <div
        data-testid="panel-board"
        className="absolute inset-0 cursor-grab touch-none overflow-auto overscroll-contain bg-muted bg-[radial-gradient(#d9d9d9_1px,transparent_1px)] bg-size-[18px_18px] bg-local active:cursor-grabbing"
        onPointerDown={(event) => {
          if (event.button !== 0) return;
          event.currentTarget.setPointerCapture(event.pointerId);
          const el = event.currentTarget;
          drag.current = { x: event.clientX, y: event.clientY, left: el.scrollLeft, top: el.scrollTop };
        }}
        onPointerMove={(event) => {
          const start = drag.current;
          if (!start) return;
          event.currentTarget.scrollLeft = start.left - (event.clientX - start.x);
          event.currentTarget.scrollTop = start.top - (event.clientY - start.y);
        }}
        onPointerUp={(event) => {
          drag.current = null;
          event.currentTarget.releasePointerCapture(event.pointerId);
        }}
        onPointerCancel={() => {
          drag.current = null;
        }}
      >
        <div className="relative" style={{ width: layout.width + padX * 2, height: layout.height + padY * 2 }}>
          <svg aria-hidden className="absolute top-0 left-0 overflow-visible" width={1} height={1}>
            {layout.edges.map((edge) => {
              const from = byId.get(edge.from);
              const to = byId.get(edge.to);
              if (!from || !to) return null;
              return (
                <polyline
                  key={edge.to}
                  points={edgePoints(from, to, edge.kind)
                    .map(([x, y]) => `${x + padX + 0.75},${y + padY}`)
                    .join(" ")}
                  fill="none"
                  stroke={edge.kind === "trunk" ? "#b5b5b5" : "#d4d4d4"}
                  strokeWidth={1.5}
                />
              );
            })}
          </svg>
          {layout.nodes.map((node) => (
            <span
              key={node.id}
              className={cn(
                "absolute box-border inline-flex items-center rounded-md px-[11px] text-ui leading-[1.45] whitespace-nowrap",
                nodeStyle[node.kind],
                node.uncertain && "border-dashed border-faint text-subtle",
              )}
              style={{ left: node.x + padX, top: node.y + padY, height: node.height }}
            >
              {node.text}
            </span>
          ))}
        </div>
      </div>
      <button
        type="button"
        data-testid="panel-board-open"
        onClick={onOpen}
        className="absolute right-3.5 bottom-3.5 inline-flex h-8 items-center gap-1.5 rounded-full border border-input bg-background px-3 text-label font-medium text-secondary-foreground shadow-[0_4px_14px_rgba(0,0,0,0.08)] transition-colors hover:bg-hover"
      >
        <AppWindow className="size-3.5" />
        在主窗口打开
      </button>
    </div>
  );
}
