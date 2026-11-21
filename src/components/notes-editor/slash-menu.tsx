import { canUseRegexLookbehind } from "prosekit/core";
import { useEditor } from "prosekit/react";
import { AutocompleteItem, AutocompletePopup, AutocompletePositioner, AutocompleteRoot } from "prosekit/react/autocomplete";
import type { NotesExtension } from "@/components/notes-editor/extension";

const regex = new RegExp(`${canUseRegexLookbehind() ? String.raw`(?<!\S)` : ""}${String.raw`\/(\S.{0,90})?$`}`, "u");

function Item({ label, hint, value, onSelect }: { label: string; hint: string; value: string; onSelect: () => void }) {
  return (
    <AutocompleteItem
      value={value}
      onSelect={onSelect}
      className="flex h-8 cursor-default items-center justify-between gap-6 rounded-[7px] px-2.5 text-ui text-foreground outline-none data-highlighted:bg-hover"
    >
      <span>{label}</span>
      <kbd className="font-mono text-tiny text-placeholder">{hint}</kbd>
    </AutocompleteItem>
  );
}

export function SlashMenu() {
  const editor = useEditor<NotesExtension>();
  return (
    <AutocompleteRoot regex={regex}>
      <AutocompletePositioner className="z-50 block h-min w-min overflow-visible">
        <AutocompletePopup
          data-testid="notes-slash"
          className="flex max-h-80 min-w-48 flex-col overflow-y-auto overscroll-contain rounded-lg border border-border bg-popover p-1 shadow-[0_6px_20px_rgba(0,0,0,0.1)] transition-[opacity,scale] duration-150 select-none data-[state=closed]:scale-95 data-[state=closed]:opacity-0 starting:scale-95 starting:opacity-0"
        >
          <Item label="标题 1" hint="#" value="标题1 h1 heading" onSelect={() => editor.commands.setHeading({ level: 1 })} />
          <Item label="标题 2" hint="##" value="标题2 h2 heading" onSelect={() => editor.commands.setHeading({ level: 2 })} />
          <Item label="标题 3" hint="###" value="标题3 h3 heading" onSelect={() => editor.commands.setHeading({ level: 3 })} />
          <Item label="无序列表" hint="-" value="无序列表 列表 bullet list" onSelect={() => editor.commands.wrapInList({ kind: "bullet" })} />
          <Item label="有序列表" hint="1." value="有序列表 列表 ordered list" onSelect={() => editor.commands.wrapInList({ kind: "ordered" })} />
          <Item label="待办" hint="[]" value="待办 todo task" onSelect={() => editor.commands.wrapInList({ kind: "task" })} />
          <Item label="引用" hint=">" value="引用 quote" onSelect={() => editor.commands.setBlockquote()} />
          <Item label="分割线" hint="---" value="分割线 divider hr" onSelect={() => editor.commands.insertHorizontalRule()} />
        </AutocompletePopup>
      </AutocompletePositioner>
    </AutocompleteRoot>
  );
}
