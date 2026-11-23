import { DragDropProvider, useDraggable, useDroppable } from "@dnd-kit/react";
import { GripHorizontal } from "lucide-react";
import type { ReactNode } from "react";
import { Group, Panel, Separator, useDefaultLayout } from "react-resizable-panels";
import { cn } from "@/lib/utils";
import { useWorkspace, type CardId } from "@/lib/workspace";

export function Workspace({ cards }: { cards: Record<CardId, ReactNode> }) {
  const [first, second] = useWorkspace((s) => s.order);
  const swap = useWorkspace((s) => s.swap);
  const { defaultLayout, onLayoutChanged } = useDefaultLayout({ id: "meeting-cards", storage: localStorage });

  return (
    <DragDropProvider
      onDragEnd={(event) => {
        const { source, target } = event.operation;
        if (!event.canceled && source && target && source.data.card !== target.id) swap();
      }}
    >
      <Group
        orientation="horizontal"
        defaultLayout={defaultLayout}
        onLayoutChanged={onLayoutChanged}
        className="min-h-0 flex-1"
      >
        <Panel key={first} id={first} minSize={320} defaultSize="48.7%">
          <WorkspaceCard id={first}>{cards[first]}</WorkspaceCard>
        </Panel>
        <Separator key="separator" />
        <Panel key={second} id={second} minSize={320}>
          <WorkspaceCard id={second}>{cards[second]}</WorkspaceCard>
        </Panel>
      </Group>
    </DragDropProvider>
  );
}

function WorkspaceCard({ id, children }: { id: CardId; children: ReactNode }) {
  const { ref: droppable, isDropTarget } = useDroppable({ id });
  const { ref: draggable, isDragSource } = useDraggable({ id: `grip-${id}`, data: { card: id } });

  return (
    <div
      ref={droppable}
      data-testid={`card-${id}`}
      className={cn(
        "group/card card relative flex h-full min-w-0 flex-col overflow-hidden transition-[opacity,box-shadow]",
        isDragSource && "opacity-60",
        isDropTarget && !isDragSource && "shadow-[inset_0_0_0_2px_var(--color-foreground)]",
      )}
    >
      <div className="pointer-events-none absolute inset-x-0 top-1 z-10 flex justify-center">
        <button
          ref={draggable}
          type="button"
          aria-label="拖动换位置"
          data-testid={`card-grip-${id}`}
          className="pointer-events-auto inline-flex h-3.5 w-9 cursor-grab items-center justify-center rounded-full text-faint opacity-0 transition-opacity outline-none group-hover/card:opacity-100 hover:bg-hover hover:text-muted-foreground focus-visible:opacity-100 active:cursor-grabbing"
        >
          <GripHorizontal className="size-3.5" />
        </button>
      </div>
      {children}
    </div>
  );
}
