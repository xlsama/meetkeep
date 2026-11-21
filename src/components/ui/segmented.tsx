import type { ComponentProps } from "react";
import { RadioGroup } from "radix-ui";
import { cn } from "@/lib/utils";

function Segmented({ className, ...props }: ComponentProps<typeof RadioGroup.Root>) {
  return (
    <RadioGroup.Root
      data-slot="segmented"
      orientation="horizontal"
      className={cn("inline-flex gap-0.5 rounded-md bg-secondary p-0.5", className)}
      {...props}
    />
  );
}

function SegmentedItem({ className, ...props }: ComponentProps<typeof RadioGroup.Item>) {
  return (
    <RadioGroup.Item
      data-slot="segmented-item"
      className={cn(
        "h-7 rounded-sm px-3 text-ui text-subtle transition-[color,background-color,box-shadow] outline-none",
        "data-[state=checked]:bg-background data-[state=checked]:font-medium data-[state=checked]:text-foreground data-[state=checked]:shadow-[0_1px_2px_rgba(0,0,0,0.08)]",
        "focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

export { Segmented, SegmentedItem };
