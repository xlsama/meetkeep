import { TriangleAlert, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import type { ReactNode } from "react";
import type { Track } from "@/mygo";
import { useRecorder } from "@/lib/recorder";
import { cn } from "@/lib/utils";

const trackLabel: Record<Track, string> = { mic: "麦克风", system: "系统声音", file: "导入的文件" };

export function Notice({
  message,
  onDismiss,
  action,
  className,
  testId,
}: {
  message: string | null;
  onDismiss?: () => void;
  action?: ReactNode;
  className?: string;
  testId: string;
}) {
  return (
    <AnimatePresence initial={false}>
      {message && (
        <motion.div
          key="notice"
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.15 }}
          className="shrink-0 overflow-hidden"
        >
          <div
            role="alert"
            data-testid={testId}
            className={cn(
              "flex items-start gap-2 rounded-md border border-warning-border bg-warning-bg py-2 pr-1.5 pl-3 text-label leading-normal text-warning",
              className,
            )}
          >
            <TriangleAlert className="mt-[3px] size-[13px] shrink-0" />
            <span data-selectable className="min-w-0 flex-1 break-words">
              {message}
            </span>
            {action}
            {onDismiss && (
              <button
                type="button"
                aria-label="关闭"
                data-testid={`${testId}-dismiss`}
                onClick={onDismiss}
                className="no-drag -my-0.5 inline-flex size-[22px] shrink-0 items-center justify-center rounded-xs text-warning transition-colors hover:bg-warning-border/50"
              >
                <X className="size-[13px]" />
              </button>
            )}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

export function FaultNotice({ className }: { className?: string }) {
  const fault = useRecorder((s) => s.fault);
  const dismiss = useRecorder((s) => s.dismissFault);
  return (
    <Notice
      testId="fault"
      className={className}
      message={fault ? `${trackLabel[fault.track]}：${fault.message}` : null}
      onDismiss={dismiss}
    />
  );
}
