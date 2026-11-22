import { create } from "zustand";
import { persist } from "zustand/middleware";

export type CardId = "transcript" | "notes";
export type RecordsView = "cards" | "list";

export const useWorkspace = create<{
  order: [CardId, CardId];
  sidebar: boolean;
  view: RecordsView;
  swap: () => void;
  toggleSidebar: () => void;
  setView: (view: RecordsView) => void;
}>()(
  persist(
    (set) => ({
      order: ["transcript", "notes"],
      sidebar: true,
      view: "cards",
      swap: () => set((s) => ({ order: [s.order[1], s.order[0]] })),
      toggleSidebar: () => set((s) => ({ sidebar: !s.sidebar })),
      setView: (view) => set({ view }),
    }),
    { name: "meetkeep.workspace" },
  ),
);
