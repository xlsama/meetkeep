import type { ComponentProps } from "react";
import { Switch as SwitchPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "relative inline-flex h-[20px] w-[34px] shrink-0 cursor-pointer rounded-full transition-colors duration-200 outline-none",
        "bg-border-strong data-[state=checked]:bg-primary",
        "focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:cursor-default disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="pointer-events-none absolute top-[2px] left-[2px] block size-[16px] rounded-full bg-white shadow-[0_1px_2px_rgba(0,0,0,0.2)] transition-transform duration-200 ease-out data-[state=checked]:translate-x-[14px]"
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
