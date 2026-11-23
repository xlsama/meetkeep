import { Check, ChevronDown, LoaderCircle } from "lucide-react";
import { Popover } from "radix-ui";
import { useId, useRef, useState, type ComponentProps } from "react";
import { Input } from "@/components/ui/input";
import { fuzzyFilter } from "@/lib/fuzzy";
import { cn } from "@/lib/utils";

function reveal(item: HTMLElement | null | undefined) {
  const list = item?.parentElement;
  if (!item || !list) return;
  const top = item.offsetTop - 6;
  const bottom = item.offsetTop + item.offsetHeight + 6;
  if (top < list.scrollTop) list.scrollTop = top;
  else if (bottom > list.scrollTop + list.clientHeight) list.scrollTop = bottom - list.clientHeight;
}

function revealSelected(list: HTMLDivElement | null) {
  reveal(list?.querySelector<HTMLElement>("[aria-selected=true]"));
}

function Combobox({
  value,
  options,
  loading = false,
  testId,
  onValueChange,
  onPick,
  onBlur,
  className,
  ...props
}: Omit<ComponentProps<"input">, "value" | "onChange"> & {
  value: string;
  options: string[];
  loading?: boolean;
  testId: string;
  onValueChange: (value: string) => void;
  onPick: (value: string) => void;
}) {
  const listId = useId();
  const anchor = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState(false);
  const [cursor, setCursor] = useState<number | null>(null);

  const shown = typed ? fuzzyFilter(options, value) : options;
  const visible = open && shown.length > 0;
  const active = cursor ?? (typed ? 0 : shown.indexOf(value));

  const show = () => {
    setOpen(true);
    setTyped(false);
    setCursor(null);
  };

  const move = (step: number) => {
    const next = Math.min(Math.max(active + step, 0), shown.length - 1);
    setCursor(next);
    reveal(document.getElementById(`${listId}-${next}`));
  };

  const pick = (option: string) => {
    setOpen(false);
    onPick(option);
  };

  return (
    <Popover.Root open={visible} onOpenChange={setOpen}>
      <Popover.Anchor asChild>
        <div ref={anchor} className={cn("relative", className)}>
          <Input
            ref={input}
            role="combobox"
            aria-expanded={visible}
            aria-controls={listId}
            aria-autocomplete="list"
            aria-activedescendant={visible && active >= 0 ? `${listId}-${active}` : undefined}
            data-testid={testId}
            className="pr-8 font-mono text-label"
            value={value}
            onChange={(event) => {
              onValueChange(event.target.value);
              setOpen(true);
              setTyped(true);
              setCursor(null);
            }}
            onClick={() => {
              if (!open) show();
            }}
            onBlur={(event) => {
              setOpen(false);
              onBlur?.(event);
            }}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return;
              if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                if (shown.length === 0) return;
                event.preventDefault();
                if (visible) move(event.key === "ArrowDown" ? 1 : -1);
                else show();
              } else if (event.key === "Enter" && visible && shown[active] !== undefined) {
                event.preventDefault();
                pick(shown[active]);
              }
            }}
            {...props}
          />
          {loading && options.length === 0 ? (
            <LoaderCircle className="pointer-events-none absolute top-1/2 right-2.5 size-3.5 -translate-y-1/2 animate-spin text-muted-foreground" />
          ) : (
            options.length > 0 && (
              <button
                type="button"
                tabIndex={-1}
                aria-label="展开"
                data-testid={`${testId}-toggle`}
                className="absolute top-1/2 right-1 flex size-6 -translate-y-1/2 items-center justify-center rounded-[5px] text-muted-foreground outline-none hover:text-foreground"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => {
                  input.current?.focus();
                  if (open) setOpen(false);
                  else show();
                }}
              >
                <ChevronDown className="size-3.5" />
              </button>
            )
          )}
        </div>
      </Popover.Anchor>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={4}
          collisionPadding={12}
          onOpenAutoFocus={(event) => event.preventDefault()}
          onCloseAutoFocus={(event) => event.preventDefault()}
          onInteractOutside={(event) => {
            if (event.target instanceof Node && anchor.current?.contains(event.target)) event.preventDefault();
          }}
          className="z-50 w-(--radix-popover-trigger-width) origin-(--radix-popover-content-transform-origin) overflow-hidden rounded-lg bg-popover text-popover-foreground shadow-[0_12px_36px_rgba(0,0,0,0.12),0_0_0_0.5px_rgba(0,0,0,0.12)] outline-none data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"
        >
          <div
            ref={revealSelected}
            id={listId}
            role="listbox"
            data-testid={`${testId}-list`}
            className="relative max-h-[min(264px,var(--radix-popover-content-available-height))] overflow-y-auto p-1.5"
            onMouseDown={(event) => event.preventDefault()}
          >
            {shown.map((option, index) => (
              <div
                key={option}
                id={`${listId}-${index}`}
                role="option"
                aria-selected={option === value}
                data-active={index === active || undefined}
                data-testid={`${testId}-option`}
                className="flex h-[30px] items-center gap-2.5 rounded-[7px] px-2.5 font-mono text-label text-foreground select-none data-[active]:bg-hover"
                onMouseMove={(event) => {
                  if (index !== active && (event.movementX !== 0 || event.movementY !== 0)) setCursor(index);
                }}
                onClick={() => pick(option)}
              >
                <span className="min-w-0 flex-1 truncate">{option}</span>
                {option === value && <Check className="size-[15px] shrink-0 text-subtle" />}
              </div>
            ))}
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

export { Combobox };
