import { currentWindow, isMyGo } from "mygo-runtime";
import { useEffect } from "react";
import { create } from "zustand";

export const useSheet = create<{ open: boolean; setOpen: (open: boolean) => void }>()((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));

export function useWindowTitle(title: string) {
  useEffect(() => {
    document.title = title;
    if (isMyGo()) void currentWindow.setTitle(title);
  }, [title]);
}

const mac = navigator.userAgent.includes("Mac");

/** Whether a key is pressed with the key of the platform's shortcuts, alone: ⌘ on macOS, Ctrl elsewhere. */
export function withCommand(event: KeyboardEvent) {
  return (mac ? event.metaKey && !event.ctrlKey : event.ctrlKey && !event.metaKey) && !event.shiftKey && !event.altKey;
}
