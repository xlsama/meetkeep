import { Feedback, PointerActivationConstraints } from "@dnd-kit/dom";
import { DragDropProvider, KeyboardSensor, PointerSensor, useDraggable } from "@dnd-kit/react";
import { ArrowLeftRight, Move } from "lucide-react";
import { AnimatePresence, motion, useMotionValue } from "motion/react";
import { createContext, use, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Group, Panel, Separator, useDefaultLayout } from "react-resizable-panels";
import { useWorkspace, type CardId } from "@/lib/workspace";

const labels: Record<CardId, string> = { transcript: "文字稿", notes: "纪要" };
const controls = "button, a, input, textarea, select, label, [role=tab], [contenteditable=true]";
const sensors = [
  PointerSensor.configure({
    activationConstraints: () => [new PointerActivationConstraints.Distance({ value: 4 })],
    preventActivation: (event) => event.target instanceof Element && event.target.closest(controls) !== null,
  }),
  KeyboardSensor,
];
const still = [Feedback.configure({ feedback: "none" })];

type Handle = (element: Element | null) => void;

const HandleContext = createContext<Handle | null>(null);

export function usePaneHandle() {
  return use(HandleContext) ?? undefined;
}

export function usePaneMovable() {
  return use(HandleContext) !== null;
}

export function PaneGrip() {
  if (!usePaneMovable()) return null;
  return (
    <Move
      aria-hidden
      className="pointer-events-none absolute top-1/2 left-[7px] size-3.5 -translate-y-1/2 scale-75 text-placeholder opacity-0 transition-[opacity,scale] duration-150 ease-out group-hover/handle:scale-100 group-hover/handle:opacity-100 group-has-[button:hover,input:hover]/handle:opacity-0"
    />
  );
}

function cardOf(value: unknown): CardId | null {
  return value === "transcript" || value === "notes" ? value : null;
}

function paneAt(point: { x: number; y: number }) {
  return cardOf(document.elementFromPoint(point.x, point.y)?.closest<HTMLElement>("[data-pane]")?.dataset.pane);
}

export function Workspace({ cards }: { cards: Record<CardId, ReactNode> }) {
  const [first, second] = useWorkspace((s) => s.order);
  const swap = useWorkspace((s) => s.swap);
  const { defaultLayout, onLayoutChanged } = useDefaultLayout({ id: "meeting-cards", storage: localStorage });
  const [dragging, setDragging] = useState<CardId | null>(null);
  const [over, setOver] = useState<CardId | null>(null);
  const x = useMotionValue(0);
  const y = useMotionValue(0);

  return (
    <DragDropProvider
      sensors={sensors}
      onDragStart={(event) => {
        const { current } = event.operation.position;
        x.set(current.x);
        y.set(current.y);
        setDragging(cardOf(event.operation.source?.data.card));
      }}
      onDragMove={(event) => {
        const point = event.to ?? event.operation.position.current;
        x.set(point.x);
        y.set(point.y);
        setOver(paneAt(point));
      }}
      onDragEnd={(event) => {
        const target = paneAt(event.operation.position.current);
        setDragging(null);
        setOver(null);
        if (!event.canceled && dragging && target && target !== dragging) swap();
      }}
    >
      <Group
        orientation="horizontal"
        defaultLayout={defaultLayout}
        onLayoutChanged={onLayoutChanged}
        className="min-h-0 flex-1"
      >
        <Panel key={first} id={first} minSize={320} defaultSize="48.7%">
          <WorkspaceCard id={first} slot={0} dragging={dragging} over={over}>
            {cards[first]}
          </WorkspaceCard>
        </Panel>
        <Separator key="separator" />
        <Panel key={second} id={second} minSize={320}>
          <WorkspaceCard id={second} slot={1} dragging={dragging} over={over}>
            {cards[second]}
          </WorkspaceCard>
        </Panel>
      </Group>
      {createPortal(
        <AnimatePresence>
          {dragging && (
            <motion.div
              key="chip"
              data-testid="card-drag-chip"
              style={{ x, y }}
              initial={{ opacity: 0, scale: 0.85 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.85 }}
              transition={{ duration: 0.14, ease: "easeOut" }}
              className="pointer-events-none fixed top-3 left-3 z-[70] inline-flex h-8 items-center gap-1.5 rounded-full bg-primary pr-3 pl-2.5 text-ui font-medium text-primary-foreground shadow-[0_8px_24px_rgba(0,0,0,0.22)]"
            >
              <Move className="size-3.5" />
              {labels[dragging]}
            </motion.div>
          )}
        </AnimatePresence>,
        document.body,
      )}
    </DragDropProvider>
  );
}

function WorkspaceCard({
  id,
  slot,
  dragging,
  over,
  children,
}: {
  id: CardId;
  slot: number;
  dragging: CardId | null;
  over: CardId | null;
  children: ReactNode;
}) {
  const element = useRef<HTMLDivElement>(null);
  const { handleRef } = useDraggable({ id: `pane-${id}`, data: { card: id }, plugins: still, element });
  const source = dragging === id;
  const target = over === id && dragging !== null && !source;

  return (
    <motion.div
      ref={element}
      layout="position"
      layoutDependency={slot}
      animate={{ opacity: source ? 0.5 : 1, scale: source ? 0.985 : 1 }}
      transition={{
        layout: { type: "spring", duration: 0.5, bounce: 0.15 },
        default: { duration: 0.15, ease: "easeOut" },
      }}
      data-testid={`card-${id}`}
      data-pane={id}
      className="card relative flex h-full min-w-0 flex-col overflow-hidden"
    >
      <HandleContext value={handleRef}>{children}</HandleContext>
      <AnimatePresence>
        {target && (
          <motion.div
            key="target"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.12, ease: "easeOut" }}
            className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center rounded-[12px] bg-drag/[0.06] shadow-[inset_0_0_0_2px_var(--color-drag)]"
          >
            <motion.span
              initial={{ scale: 0.6 }}
              animate={{ scale: 1 }}
              transition={{ type: "spring", duration: 0.3, bounce: 0.3 }}
              className="inline-flex size-11 items-center justify-center rounded-full bg-background text-drag shadow-[0_6px_20px_rgba(0,0,0,0.12),0_0_0_0.5px_rgba(0,0,0,0.08)]"
            >
              <ArrowLeftRight className="size-5" />
            </motion.span>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
