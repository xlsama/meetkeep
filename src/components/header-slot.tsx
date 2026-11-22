import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { create } from "zustand";

type Slot = HTMLElement | null;

export const useHeader = create<{
  title: Slot;
  actions: Slot;
  setTitle: (slot: Slot) => void;
  setActions: (slot: Slot) => void;
}>()((set) => ({
  title: null,
  actions: null,
  setTitle: (title) => set({ title }),
  setActions: (actions) => set({ actions }),
}));

export function HeaderTitle({ children }: { children: ReactNode }) {
  const slot = useHeader((s) => s.title);
  return slot ? createPortal(children, slot) : null;
}

export function HeaderActions({ children }: { children: ReactNode }) {
  const slot = useHeader((s) => s.actions);
  return slot ? createPortal(children, slot) : null;
}

export function PageTitle({ title, meta }: { title: string; meta?: ReactNode }) {
  return (
    <HeaderTitle>
      <h1 className="m-0 max-w-full truncate text-title leading-[1.35] font-semibold" data-selectable>
        {title}
      </h1>
      {meta && <span className="max-w-full truncate text-label text-muted-foreground tabular-nums">{meta}</span>}
    </HeaderTitle>
  );
}
