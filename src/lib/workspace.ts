import { create } from "zustand";
import { persist } from "zustand/middleware";

export type CardId = "transcript" | "notes";

export const useWorkspace = create<{ order: [CardId, CardId]; swap: () => void }>()(
  persist(
    (set) => ({
      order: ["transcript", "notes"],
      swap: () => set((s) => ({ order: [s.order[1], s.order[0]] })),
    }),
    { name: "meetkeep.workspace" },
  ),
);
