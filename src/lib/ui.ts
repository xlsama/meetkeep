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
