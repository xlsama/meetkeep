import "prosekit/basic/style.css";
import "@/components/notes-editor/notes-editor.css";
import { createEditor, type Editor, type NodeJSON } from "prosekit/core";
import { definePlaceholder } from "prosekit/extensions/placeholder";
import { defineReadonly } from "prosekit/extensions/readonly";
import { ProseKit, useDocChange, useExtension } from "prosekit/react";
import { useCallback, useEffect, useState } from "react";
import { defineNotesExtension, docText, type NotesExtension } from "@/components/notes-editor/extension";
import { InlineMenu } from "@/components/notes-editor/inline-menu";
import { SlashMenu } from "@/components/notes-editor/slash-menu";
import { Library, type NotesState } from "@/mygo";
import { notesToDoc } from "@/lib/notes-doc";
import type { Jump } from "@/lib/notes";
import { errorMessage, notesQuery, queryClient } from "@/lib/queries";

class DocSession {
  private key = "";
  private notes = "";
  private doc = "";
  private timer = 0;

  constructor(
    readonly editor: Editor<NotesExtension>,
    private readonly meetingId: string,
    private readonly onSaveError: (error: string | null) => void,
  ) {}

  private current() {
    return JSON.stringify(this.editor.getDocJSON());
  }

  private apply(state: NotesState) {
    window.clearTimeout(this.timer);
    let content: NodeJSON = notesToDoc(state.notes);
    if (state.doc !== "") {
      try {
        content = JSON.parse(state.doc) as NodeJSON;
      } catch {
        content = notesToDoc(state.notes);
      }
    }
    try {
      this.editor.setContent(content);
    } catch {
      this.editor.setContent(notesToDoc(state.notes));
    }
    this.doc = state.doc !== "" ? state.doc : this.current();
  }

  receive(state: NotesState) {
    const notes = JSON.stringify(state.notes);
    const key = state.doc !== "" ? `doc:${state.doc}` : `notes:${notes}`;
    if (key === this.key) return;
    const first = this.key === "";
    this.key = key;
    const notesChanged = notes !== this.notes;
    this.notes = notes;
    if (first || (state.doc === "" && notesChanged) || (state.doc !== "" && state.doc !== this.doc)) this.apply(state);
  }

  changed(editable: boolean) {
    if (!editable) return;
    window.clearTimeout(this.timer);
    this.timer = window.setTimeout(() => this.flush(), 800);
  }

  flush() {
    window.clearTimeout(this.timer);
    const doc = this.current();
    if (doc === this.doc) return;
    this.doc = doc;
    this.key = `doc:${doc}`;
    queryClient.setQueryData(notesQuery(this.meetingId).queryKey, (state) => (state ? { ...state, doc } : state));
    this.onSaveError(null);
    Library.saveNotesDoc(this.meetingId, doc, docText(this.editor.state.doc)).catch((error: unknown) =>
      this.onSaveError(errorMessage(error)),
    );
  }
}

export default function NotesEditor({
  meetingId,
  state,
  editable,
  onJump,
  onSaveError,
}: {
  meetingId: string;
  state: NotesState;
  editable: boolean;
  onJump: (jump: Jump) => void;
  onSaveError: (error: string | null) => void;
}) {
  "use no memo";
  const [session] = useState(() => {
    const created = new DocSession(createEditor({ extension: defineNotesExtension() }), meetingId, onSaveError);
    created.receive(state);
    return created;
  });
  const [readonly] = useState(() => defineReadonly());
  const [placeholder] = useState(() => definePlaceholder({ placeholder: "输入 / 插入" }));
  const editor = session.editor;
  const mount = useCallback((element: HTMLDivElement | null) => editor.mount(element), [editor]);

  useExtension(editable ? null : readonly, { editor });
  useExtension(editable ? placeholder : null, { editor });
  useDocChange(() => session.changed(editable), { editor });

  useEffect(() => session.receive(state), [session, state]);
  useEffect(() => () => session.flush(), [session]);

  return (
    <ProseKit editor={editor}>
      <div className="relative">
        <div
          ref={mount}
          data-testid="notes-editor"
          data-selectable
          className="notes-doc"
          onClick={(event) => {
            const target = event.target instanceof Element ? event.target.closest<HTMLElement>("[data-timestamp]") : null;
            if (!target) return;
            onJump({ seq: Number(target.dataset.seq ?? -1), atMs: Number(target.dataset.ms ?? 0) });
          }}
        />
        {editable && (
          <>
            <InlineMenu />
            <SlashMenu />
          </>
        )}
      </div>
    </ProseKit>
  );
}
