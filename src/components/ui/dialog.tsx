import type { ComponentProps, ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Dialog as DialogPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

const Dialog = DialogPrimitive.Root;
const DialogTitle = DialogPrimitive.Title;
const DialogDescription = DialogPrimitive.Description;
const DialogClose = DialogPrimitive.Close;

function Sheet({
  open,
  onOpenChange,
  children,
  className,
  ...props
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
} & Omit<ComponentProps<typeof DialogPrimitive.Content>, "asChild" | "forceMount">) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open && (
          <DialogPrimitive.Portal forceMount>
            <DialogPrimitive.Overlay asChild forceMount>
              <motion.div
                className="fixed inset-0 z-50 bg-black/10"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
              />
            </DialogPrimitive.Overlay>
            <DialogPrimitive.Content asChild forceMount aria-describedby={undefined} {...props}>
              <motion.div
                data-slot="sheet"
                className={cn(
                  "fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100%-40px)] w-[480px] max-w-[calc(100%-40px)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-xl bg-background shadow-[0_24px_64px_rgba(0,0,0,0.18),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none",
                  className,
                )}
                initial={{ opacity: 0, scale: 0.97 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.98 }}
                transition={{ type: "spring", duration: 0.32, bounce: 0 }}
              >
                {children}
              </motion.div>
            </DialogPrimitive.Content>
          </DialogPrimitive.Portal>
        )}
      </AnimatePresence>
    </DialogPrimitive.Root>
  );
}

export { Dialog, DialogClose, DialogDescription, DialogTitle, Sheet };
