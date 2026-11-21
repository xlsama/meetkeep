import { motion } from "motion/react";
import { cn } from "@/lib/utils";

const profile = [5, 9, 13, 9, 5];

export function LevelMeter({
  level,
  on,
  off,
  className,
  heights = profile,
}: {
  level: number;
  on: string;
  off: string;
  className?: string;
  heights?: number[];
}) {
  const lit = level <= 0.02 ? 0 : Math.min(heights.length, Math.ceil(Math.sqrt(level) * heights.length));
  return (
    <span aria-hidden className={cn("inline-flex h-4 items-center gap-0.5", className)}>
      {heights.map((height, index) => (
        <motion.span
          key={index}
          className="w-[3px] rounded-[2px]"
          style={{ height }}
          initial={false}
          animate={{ backgroundColor: index < lit ? on : off }}
          transition={{ duration: 0.12 }}
        />
      ))}
    </span>
  );
}

export function PlayingMeter({ className }: { className?: string }) {
  return (
    <span aria-hidden className={cn("inline-flex h-4 items-center gap-0.5", className)}>
      {profile.map((height, index) => (
        <motion.span
          key={index}
          className="w-[3px] origin-center rounded-[2px] bg-secondary-foreground"
          style={{ height }}
          animate={{ scaleY: [0.45, 1, 0.6, 0.9, 0.45] }}
          transition={{ duration: 1.4, repeat: Infinity, ease: "easeInOut", delay: index * 0.12 }}
        />
      ))}
    </span>
  );
}
