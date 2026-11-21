import { Bold, Code, Italic, Link, Strikethrough, Unlink } from "lucide-react";
import type { Editor } from "prosekit/core";
import type { LinkAttrs } from "prosekit/extensions/link";
import { useEditor, useEditorDerivedValue } from "prosekit/react";
import { InlinePopoverPopup, InlinePopoverPositioner, InlinePopoverRoot } from "prosekit/react/inline-popover";
import { useState, type ReactNode } from "react";
import type { NotesExtension } from "@/components/notes-editor/extension";
import { cn } from "@/lib/utils";

function menuState(editor: Editor<NotesExtension>) {
  const { $from } = editor.state.selection;
  const link = ($from.marksAcross($from) ?? []).find((mark) => mark.type.name === "link");
  return {
    bold: editor.marks.bold.isActive(),
    italic: editor.marks.italic.isActive(),
    strike: editor.marks.strike.isActive(),
    code: editor.marks.code.isActive(),
    link: editor.marks.link.isActive(),
    canLink: editor.commands.addLink.canExec({ href: "" }),
    href: link ? (link.attrs as LinkAttrs).href : "",
  };
}

const popup =
  "flex rounded-lg border border-border bg-popover shadow-[0_6px_20px_rgba(0,0,0,0.1)] transition-[opacity,scale] duration-150 data-[state=closed]:scale-95 data-[state=closed]:opacity-0 starting:scale-95 starting:opacity-0";

function Tool({
  label,
  active,
  onClick,
  children,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={active}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onClick}
      className={cn(
        "inline-flex size-7 items-center justify-center rounded-[7px] transition-colors",
        active ? "bg-accent text-foreground" : "text-subtle hover:bg-hover hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}

export function InlineMenu() {
  const editor = useEditor<NotesExtension>();
  const state = useEditorDerivedValue(menuState);
  const [linking, setLinking] = useState(false);

  const applyLink = (href: string) => {
    if (href === "") editor.commands.removeLink();
    else editor.commands.addLink({ href });
    setLinking(false);
    editor.focus();
  };

  return (
    <>
      <InlinePopoverRoot onOpenChange={(event) => !event.detail && setLinking(false)}>
        <InlinePopoverPositioner className="z-50 block h-min w-min overflow-visible">
          <InlinePopoverPopup data-testid="notes-toolbar" className={cn(popup, "gap-0.5 p-[3px]")}>
            <Tool label="加粗" active={state.bold} onClick={() => editor.commands.toggleBold()}>
              <Bold className="size-[15px]" />
            </Tool>
            <Tool label="斜体" active={state.italic} onClick={() => editor.commands.toggleItalic()}>
              <Italic className="size-[15px]" />
            </Tool>
            <Tool label="删除线" active={state.strike} onClick={() => editor.commands.toggleStrike()}>
              <Strikethrough className="size-[15px]" />
            </Tool>
            <Tool label="行内代码" active={state.code} onClick={() => editor.commands.toggleCode()}>
              <Code className="size-[15px]" />
            </Tool>
            {state.canLink && (
              <Tool
                label="链接"
                active={state.link}
                onClick={() => {
                  editor.commands.expandLink();
                  setLinking((open) => !open);
                }}
              >
                <Link className="size-[15px]" />
              </Tool>
            )}
          </InlinePopoverPopup>
        </InlinePopoverPositioner>
      </InlinePopoverRoot>
      <InlinePopoverRoot defaultOpen={false} open={linking} onOpenChange={(event) => setLinking(event.detail)}>
        <InlinePopoverPositioner placement="bottom" className="z-50 block h-min w-min overflow-visible">
          <InlinePopoverPopup className={cn(popup, "items-center gap-1 p-1")}>
            {linking && (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  const input = event.currentTarget.elements.namedItem("href");
                  applyLink(input instanceof HTMLInputElement ? input.value.trim() : "");
                }}
              >
                <input
                  name="href"
                  aria-label="链接"
                  placeholder="https://"
                  defaultValue={state.href}
                  autoComplete="off"
                  spellCheck={false}
                  className="h-7 w-60 rounded-[7px] border-0 bg-transparent px-2 text-ui text-foreground outline-none"
                />
              </form>
            )}
            {state.link && (
              <Tool label="移除链接" active={false} onClick={() => applyLink("")}>
                <Unlink className="size-[15px]" />
              </Tool>
            )}
          </InlinePopoverPopup>
        </InlinePopoverPositioner>
      </InlinePopoverRoot>
    </>
  );
}
