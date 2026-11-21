import type { ComponentProps } from "react";
import { Switch as SwitchPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "relative inline-flex h-[22px] w-9 shrink-0 rounded-full transition-colors outline-none",
        "bg-border-strong data-[state=checked]:bg-primary",
        "focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="pointer-events-none absolute top-0.5 left-0.5 block size-[18px] rounded-full bg-white shadow-[0_1px_2px_rgba(0,0,0,0.12)] transition-transform data-[state=checked]:translate-x-3.5"
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
