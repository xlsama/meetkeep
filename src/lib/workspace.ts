import { create } from "zustand";
import { persist } from "zustand/middleware";

export type CardId = "transcript" | "notes";

export const useWorkspace = create<{
  order: [CardId, CardId];
  sidebar: boolean;
  swap: () => void;
  toggleSidebar: () => void;
}>()(
  persist(
    (set) => ({
      order: ["transcript", "notes"],
      sidebar: true,
      swap: () => set((s) => ({ order: [s.order[1], s.order[0]] })),
      toggleSidebar: () => set((s) => ({ sidebar: !s.sidebar })),
    }),
    { name: "meetkeep.workspace" },
  ),
);
