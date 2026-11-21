import { create } from "zustand";
import type { Progress } from "@/mygo";

export const useProgress = create<{ items: Record<string, Progress> }>()(() => ({ items: {} }));

export function applyProgress(progress: Progress) {
  useProgress.setState((state) => ({ items: { ...state.items, [progress.meetingId]: progress } }));
}

export function clearProgress(meetingId: string) {
  useProgress.setState((state) => {
    if (!(meetingId in state.items)) return state;
    const { [meetingId]: _, ...items } = state.items;
    return { items };
  });
}
