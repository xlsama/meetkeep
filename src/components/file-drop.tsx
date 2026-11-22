import { Upload } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { isMyGo, onFileDrop } from "mygo-runtime";
import { useEffect, useEffectEvent, useState } from "react";
import { useImport } from "@/lib/importer";

function carriesFiles(event: DragEvent) {
  return event.dataTransfer?.types.includes("Files") ?? false;
}

export function FileDrop() {
  const { drop } = useImport();
  const [over, setOver] = useState(false);
  const dropped = useEffectEvent((paths: string[]) => {
    setOver(false);
    void drop(paths);
  });

  useEffect(() => {
    if (!isMyGo()) return;
    return onFileDrop(({ paths }) => dropped(paths));
  }, []);

  useEffect(() => {
    let depth = 0;
    const enter = (event: DragEvent) => {
      if (!carriesFiles(event)) return;
      depth += 1;
      setOver(true);
    };
    const leave = (event: DragEvent) => {
      if (!carriesFiles(event)) return;
      depth = Math.max(0, depth - 1);
      if (depth === 0) setOver(false);
    };
    const hover = (event: DragEvent) => {
      if (!carriesFiles(event)) return;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
    };
    const end = (event: DragEvent) => {
      if (carriesFiles(event)) event.preventDefault();
      depth = 0;
      setOver(false);
    };
    window.addEventListener("dragenter", enter);
    window.addEventListener("dragleave", leave);
    window.addEventListener("dragover", hover);
    window.addEventListener("drop", end);
    return () => {
      window.removeEventListener("dragenter", enter);
      window.removeEventListener("dragleave", leave);
      window.removeEventListener("dragover", hover);
      window.removeEventListener("drop", end);
    };
  }, []);

  return (
    <AnimatePresence>
      {over && (
        <motion.div
          data-testid="file-drop"
          className="pointer-events-none fixed inset-0 z-[60] p-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.12 }}
        >
          <div className="flex h-full flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed border-foreground/35 bg-canvas/85 backdrop-blur-sm">
            <span className="inline-flex size-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
              <Upload className="size-5" />
            </span>
            <span className="text-lead font-semibold text-foreground">松开导入</span>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
