import { useQuery } from "@tanstack/react-query";
import { defineReactNodeView, type ReactNodeViewProps } from "prosekit/react";
import { Popover } from "radix-ui";
import { createContext, useContext, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { SpeakerPopover } from "@/components/speaker-popover";
import { meetingQuery } from "@/lib/queries";
import { speakerVoice } from "@/lib/speakers";

export const NotesMeeting = createContext("");

const glue = "\u2060";

function AssigneeView({ node }: ReactNodeViewProps) {
  const meetingId = useContext(NotesMeeting);
  const speakers = useQuery({ ...meetingQuery(meetingId), enabled: meetingId !== "" }).data?.speakers ?? [];
  const owner = String(node.attrs.owner ?? "").trim();
  const due = String(node.attrs.due ?? "").trim();
  const speaker = owner === "" ? undefined : speakers.find((s) => s.name === owner);
  const [open, setOpen] = useState(false);
  const anchor = useRef<HTMLButtonElement>(null);

  if (!speaker) {
    return (
      <>
        <span className="notes-chip" data-assignee="">
          {[owner, due].filter((part) => part !== "").join(" · ")}
        </span>
        {glue}
      </>
    );
  }
  const face = (
    <>
      <Avatar name={speaker.name} tone={speakerVoice(speaker, speakers).avatar} className="size-[14px] text-[9px]" />
      {owner}
    </>
  );
  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <span className="notes-chip" data-assignee="">
        {speaker.self ? (
          <span className="notes-owner">{face}</span>
        ) : (
          <Popover.Trigger asChild>
            <button
              ref={anchor}
              type="button"
              aria-label={`说话人：${owner}`}
              data-testid="notes-assignee"
              onMouseDown={(event) => event.preventDefault()}
              className="notes-owner"
            >
              {face}
            </button>
          </Popover.Trigger>
        )}
        {due !== "" && ` · ${due}`}
      </span>
      {glue}
      {open && !speaker.self && (
        <SpeakerPopover meetingId={meetingId} speakerId={speaker.id} avatar={anchor} onDone={() => setOpen(false)} />
      )}
    </Popover.Root>
  );
}

export function defineAssigneeView() {
  return defineReactNodeView({
    name: "assignee",
    as: () => {
      const element = document.createElement("span");
      element.className = "notes-glue";
      return element;
    },
    component: AssigneeView,
    stopEvent: (event) => event.target instanceof Element && event.target.closest("button.notes-owner") !== null,
  });
}
