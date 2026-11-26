import { createContext, use, useId, type ComponentProps } from "react";
import { motion } from "motion/react";
import { Tabs as TabsPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

const TabsContext = createContext({ value: "", id: "" });

function Tabs({ className, value, ...props }: ComponentProps<typeof TabsPrimitive.Root> & { value: string }) {
  const id = useId();
  return (
    <TabsContext value={{ value, id }}>
      <TabsPrimitive.Root data-slot="tabs" value={value} className={cn("flex min-h-0 flex-col", className)} {...props} />
    </TabsContext>
  );
}

function TabsList({ className, ...props }: ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cn("flex shrink-0 items-stretch border-b", className)}
      {...props}
    />
  );
}

function TabsTrigger({ className, children, value, ...props }: ComponentProps<typeof TabsPrimitive.Trigger>) {
  const tabs = use(TabsContext);
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      value={value}
      className={cn(
        "relative flex items-center text-muted-foreground transition-colors outline-none hover:text-foreground",
        "data-[state=active]:font-semibold data-[state=active]:text-foreground",
        "focus-visible:text-foreground",
        className,
      )}
      {...props}
    >
      {children}
      {tabs.value === value && (
        <motion.span
          aria-hidden
          layoutId={`${tabs.id}-line`}
          className="absolute inset-x-0 bottom-0 h-0.5 bg-foreground"
          transition={{ type: "spring", duration: 0.3, bounce: 0 }}
        />
      )}
    </TabsPrimitive.Trigger>
  );
}

function TabsContent({ className, ...props }: ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn("min-h-0 flex-1 outline-none data-[state=inactive]:hidden", className)}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
