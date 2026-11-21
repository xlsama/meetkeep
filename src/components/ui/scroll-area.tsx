import type { ComponentProps, Ref } from "react";
import { ScrollArea as ScrollAreaPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

function ScrollArea({
  className,
  viewportClassName,
  viewportRef,
  children,
  ...props
}: ComponentProps<typeof ScrollAreaPrimitive.Root> & { viewportClassName?: string; viewportRef?: Ref<HTMLDivElement> }) {
  return (
    <ScrollAreaPrimitive.Root data-slot="scroll-area" type="scroll" className={cn("relative overflow-hidden", className)} {...props}>
      <ScrollAreaPrimitive.Viewport
        ref={viewportRef}
        data-slot="scroll-area-viewport"
        className={cn("size-full rounded-[inherit] outline-none [&>div]:block!", viewportClassName)}
      >
        {children}
      </ScrollAreaPrimitive.Viewport>
      <ScrollAreaPrimitive.Scrollbar
        orientation="vertical"
        className="flex w-2 touch-none p-px transition-opacity select-none data-[state=hidden]:opacity-0"
      >
        <ScrollAreaPrimitive.Thumb className="relative flex-1 rounded-full bg-border-strong" />
      </ScrollAreaPrimitive.Scrollbar>
    </ScrollAreaPrimitive.Root>
  );
}

export { ScrollArea };
