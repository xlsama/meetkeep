import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

function Input({ className, type = "text", ...props }: ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      spellCheck={false}
      autoComplete="off"
      className={cn(
        "h-[34px] w-full min-w-0 rounded-md border border-input bg-background px-3 text-ui text-foreground outline-none transition-[border-color,box-shadow]",
        "focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5",
        "aria-invalid:border-invalid aria-invalid:focus-visible:ring-invalid/10",
        "disabled:cursor-default disabled:opacity-60",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
