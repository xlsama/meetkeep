import { useQuery } from "@tanstack/react-query";
import { defineReactNodeView, type ReactNodeViewProps } from "prosekit/react";
import { Popover } from "radix-ui";
import { createContext, useContext, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { SpeakerPopover } from "@/components/speaker-popover";
import { meetingQuery } from "@/lib/queries";
import { speakerNamed, speakerVoice } from "@/lib/speakers";

export const NotesMeeting = createContext("");

const glue = "\u2060";

function AssigneeView({ node }: ReactNodeViewProps) {
  const meetingId = useContext(NotesMeeting);
  const speakers = useQuery({ ...meetingQuery(meetingId), enabled: meetingId !== "" }).data?.speakers ?? [];
  const owner = String(node.attrs.owner ?? "").trim();
  const due = String(node.attrs.due ?? "").trim();
  const speaker = speakerNamed(owner, speakers);
  const [open, setOpen] = useState(false);
  const anchor = useRef<HTMLButtonElement>(null);

  const face = (
    <>
      <Avatar
        name={speaker?.name ?? owner}
        tone={speaker ? speakerVoice(speaker, speakers).avatar : "bg-placeholder"}
        className="size-4 self-center text-[9px]"
      />
      {owner}
    </>
  );
  const named = speaker !== undefined && !speaker.self;
  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <span className="notes-assignee" data-assignee="">
        {owner !== "" &&
          (named ? (
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
          ) : (
            <span className="notes-owner">{face}</span>
          ))}
        {due !== "" && <span className="notes-chip">{due}</span>}
      </span>
      {glue}
      {open && named && (
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
