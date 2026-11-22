import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { create } from "zustand";

export const useHeader = create<{ slot: HTMLElement | null; setSlot: (slot: HTMLElement | null) => void }>()((set) => ({
  slot: null,
  setSlot: (slot) => set({ slot }),
}));

export function HeaderSlot({ children }: { children: ReactNode }) {
  const slot = useHeader((s) => s.slot);
  return slot ? createPortal(children, slot) : null;
}
