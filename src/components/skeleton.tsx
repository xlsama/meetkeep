import { cn } from "@/lib/utils";

function Bone({ className }: { className: string }) {
  return <span aria-hidden className={cn("skeleton", className)} />;
}

const said = [["w-[86px]", ["w-[46%]"]], ["w-[78px]", ["w-[24%]"]], ["w-[86px]", ["w-[72%]"]], ["w-[78px]", ["w-[88%]", "w-[38%]"]], ["w-[86px]", ["w-[56%]"]]] as const;

export function TranscriptSkeleton() {
  return (
    <div className="flex min-h-0 flex-1 animate-rise flex-col gap-6 overflow-hidden px-8 pt-3" data-testid="transcript-skeleton">
      {said.map(([name, lines], index) => (
        <div key={index} className="flex gap-3">
          <Bone className="size-7 rounded-full" />
          <div className="flex min-w-0 flex-1 flex-col gap-[9px] pt-1">
            <Bone className={cn("h-2.5", name)} />
            {lines.map((width) => (
              <Bone key={width} className={cn("h-[13px]", width)} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function NotesSkeleton() {
  return (
    <div className="flex animate-rise flex-col gap-[22px]" data-testid="notes-skeleton">
      <div className="flex flex-col gap-2.5">
        <Bone className="h-5 w-[74%] rounded-[5px]" />
        <Bone className="h-[13px] w-[52%]" />
      </div>
      <div className="flex flex-col gap-3">
        <Bone className="h-2.5 w-9" />
        {["w-[58%]", "w-[44%]"].map((width) => (
          <div key={width} className="flex items-center gap-2.5">
            <Bone className="size-[15px] rounded-[5px]" />
            <Bone className={cn("h-[13px]", width)} />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-3">
        <Bone className="h-2.5 w-12" />
        <Bone className="h-[13px] w-[82%]" />
        <Bone className="h-[13px] w-[66%]" />
        <Bone className="h-[13px] w-[74%]" />
      </div>
    </div>
  );
}
