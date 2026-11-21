import type { ComponentProps } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-1.5 whitespace-nowrap font-medium transition-colors outline-none focus-visible:ring-[3px] focus-visible:ring-black/10 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        primary: "rounded-full bg-primary font-semibold text-primary-foreground hover:bg-[#262626]",
        outline: "rounded-full border border-input bg-background text-secondary-foreground hover:bg-hover",
        ghost: "rounded-sm bg-transparent text-subtle hover:bg-hover hover:text-foreground",
        ink: "rounded-full bg-ink-button text-white hover:bg-[#333333]",
        light: "rounded-full bg-white font-semibold text-foreground hover:bg-[#ededed]",
        destructive: "rounded-full bg-destructive font-semibold text-white hover:bg-[#9f1f15]",
      },
      size: {
        sm: "h-7 px-2.5 text-ui",
        md: "h-8 px-3.5 text-ui",
        lg: "h-[34px] px-4 text-ui",
        xl: "h-9 px-3.5 text-ui",
        "icon-sm": "size-[30px]",
        icon: "size-8",
        "icon-lg": "size-9",
      },
    },
    defaultVariants: { variant: "outline", size: "md" },
  },
);

function Button({
  className,
  variant,
  size,
  asChild = false,
  type = "button",
  ...props
}: ComponentProps<"button"> & VariantProps<typeof buttonVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      type={asChild ? undefined : type}
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}

export { Button, buttonVariants };
