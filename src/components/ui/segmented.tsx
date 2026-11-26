import { createContext, use, useId, type ComponentProps } from "react";
import { motion } from "motion/react";
import { RadioGroup } from "radix-ui";
import { cn } from "@/lib/utils";

const SegmentedContext = createContext({ value: "", id: "" });

function Segmented({ className, value, ...props }: ComponentProps<typeof RadioGroup.Root> & { value: string }) {
  const id = useId();
  return (
    <SegmentedContext value={{ value, id }}>
      <RadioGroup.Root
        data-slot="segmented"
        orientation="horizontal"
        value={value}
        className={cn("inline-flex gap-0.5 rounded-md bg-secondary p-0.5", className)}
        {...props}
      />
    </SegmentedContext>
  );
}

function SegmentedItem({ className, children, value, ...props }: ComponentProps<typeof RadioGroup.Item>) {
  const group = use(SegmentedContext);
  return (
    <RadioGroup.Item
      data-slot="segmented-item"
      value={value}
      className={cn(
        "relative isolate h-7 rounded-sm px-3 text-ui text-subtle transition-colors outline-none",
        "data-[state=checked]:font-medium data-[state=checked]:text-foreground",
        "focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:opacity-50",
        className,
      )}
      {...props}
    >
      {group.value === value && (
        <motion.span
          aria-hidden
          layoutId={`${group.id}-thumb`}
          className="absolute inset-0 -z-10 rounded-sm bg-background shadow-[0_1px_2px_rgba(0,0,0,0.08)]"
          transition={{ type: "spring", duration: 0.3, bounce: 0 }}
        />
      )}
      {children}
    </RadioGroup.Item>
  );
}

export { Segmented, SegmentedItem };
