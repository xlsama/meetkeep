import { Download, Ellipsis, FolderOpen, LoaderCircle, Trash2 } from "lucide-react";
import { useState } from "react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Library, type Meeting } from "@/mygo";
import { errorMessage } from "@/lib/queries";
import { mac } from "@/lib/ui";
import { cn } from "@/lib/utils";

export type OnError = (error: string | null) => void;

export async function run(onError: OnError, action: () => Promise<unknown>) {
  onError(null);
  try {
    await action();
  } catch (err) {
    onError(errorMessage(err));
  }
}

export function MeetingMenu({
  meeting,
  onError,
  className,
}: {
  meeting: Meeting;
  onError: OnError;
  className?: string;
}) {
  const [confirming, setConfirming] = useState(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label="更多"
            data-testid={`meeting-menu-${meeting.id}`}
            className={cn(
              "inline-flex size-7 items-center justify-center rounded-[8px] text-muted-foreground opacity-0 transition-[opacity,background-color] outline-none group-hover/meeting:opacity-100 hover:bg-black/[0.06] focus-visible:opacity-100 focus-visible:shadow-[0_0_0_2px_rgba(0,0,0,0.12)] data-[state=open]:bg-black/[0.06] data-[state=open]:opacity-100",
              className,
            )}
          >
            <Ellipsis className="size-4" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            data-testid="menu-save-recording"
            disabled={meeting.audio === "" || meeting.status === "processing"}
            onSelect={() => void run(onError, () => Library.saveRecording(meeting.id))}
          >
            <Download />
            保存录音
          </DropdownMenuItem>
          <DropdownMenuItem data-testid="menu-reveal" onSelect={() => void run(onError, () => Library.reveal(meeting.id))}>
            <FolderOpen />
            {mac ? "在访达中显示" : "在资源管理器中显示"}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" data-testid="menu-delete" onSelect={() => setConfirming(true)}>
            <Trash2 />
            删除
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <DeleteDialog meeting={meeting} open={confirming} onOpenChange={setConfirming} />
    </>
  );
}

export function DeleteDialog({
  meeting,
  open,
  onOpenChange,
  onDeleted,
}: {
  meeting: Meeting;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const remove = async () => {
    setDeleting(true);
    setError(null);
    try {
      await Library.delete(meeting.id);
      onOpenChange(false);
      onDeleted?.();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (deleting) return;
        setError(null);
        onOpenChange(next);
      }}
    >
      <AlertDialogContent data-testid="delete-dialog">
        <AlertDialogTitle>删除「{meeting.title}」？</AlertDialogTitle>
        <AlertDialogDescription>录音会移到废纸篓。</AlertDialogDescription>
        {error && <p className="m-0 mt-1 text-label text-destructive">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button variant="outline" size="md" data-testid="meeting-delete-cancel" disabled={deleting}>
              取消
            </Button>
          </AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button
              variant="destructive"
              size="md"
              data-testid="meeting-delete-confirm"
              disabled={deleting}
              onClick={(event) => {
                event.preventDefault();
                void remove();
              }}
            >
              {deleting && <LoaderCircle className="size-[13px] animate-spin" />}
              删除
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
