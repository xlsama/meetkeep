import type { ComponentProps } from "react";
import { DropdownMenu as DropdownMenuPrimitive } from "radix-ui";
import { cn } from "@/lib/utils";

const DropdownMenu = DropdownMenuPrimitive.Root;
const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;

function DropdownMenuContent({
  className,
  sideOffset = 4,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        data-slot="dropdown-menu-content"
        sideOffset={sideOffset}
        className={cn(
          "z-50 min-w-[180px] origin-(--radix-dropdown-menu-content-transform-origin) rounded-lg bg-popover p-1.5 text-ui text-popover-foreground shadow-[0_12px_36px_rgba(0,0,0,0.12),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none",
          "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95",
          className,
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}

function DropdownMenuItem({
  className,
  variant = "default",
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Item> & { variant?: "default" | "destructive" }) {
  return (
    <DropdownMenuPrimitive.Item
      data-slot="dropdown-menu-item"
      data-variant={variant}
      className={cn(
        "flex h-[30px] items-center gap-2.5 rounded-[7px] px-2.5 text-ui text-foreground outline-none select-none",
        "focus:bg-hover data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        "[&_svg]:size-[15px] [&_svg]:shrink-0 [&_svg]:text-subtle",
        "data-[variant=destructive]:text-destructive data-[variant=destructive]:[&_svg]:text-destructive",
        className,
      )}
      {...props}
    />
  );
}

function DropdownMenuSeparator({ className, ...props }: ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return (
    <DropdownMenuPrimitive.Separator className={cn("mx-2.5 my-1.5 h-px bg-line", className)} {...props} />
  );
}

export { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger };
