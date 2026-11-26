import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";

export type EmptyKind = "meetings" | "search" | "transcript" | "notes" | "terms";

const star = "M6 0C6.6 3.6 8.4 5.4 12 6C8.4 6.6 6.6 8.4 6 12C5.4 8.4 3.6 6.6 0 6C3.6 5.4 5.4 3.6 6 0Z";

export function Spark({ className, style }: { className?: string; style?: CSSProperties }) {
  return (
    <svg viewBox="0 0 12 12" aria-hidden className={cn("animate-twinkle text-spark", className)} style={style}>
      <path d={star} fill="currentColor" />
    </svg>
  );
}

function Twinkle({ x, y, size }: { x: number; y: number; size: number }) {
  return (
    <svg x={x - size / 2} y={y - size / 2} width={size} height={size} viewBox="0 0 12 12" overflow="visible">
      <path
        d={star}
        fill="currentColor"
        className="origin-center animate-twinkle [transform-box:fill-box]"
        style={{ animationDuration: "2.6s" }}
      />
    </svg>
  );
}

const sheet = { fill: "#ffffff", stroke: "currentColor", strokeOpacity: 0.45, strokeWidth: 2 };
const ink = { fill: "none", stroke: "currentColor", strokeWidth: 4, strokeLinecap: "round" } as const;

const art: Record<EmptyKind, ReactNode> = {
  meetings: (
    <>
      <ellipse cx="60" cy="87" rx="40" ry="5" fill="#f0f0f0" />
      <g className="animate-float">
        <g transform="rotate(-9 62 32)">
          <rect x="42" y="12" width="40" height="36" rx="7" {...sheet} />
          <path d="M51 30v1M57 25v11M63 21v19M69 26v9M75 29v3" {...ink} strokeWidth={3} />
        </g>
      </g>
      <rect x="30" y="44" width="60" height="18" rx="7" fill="#dedede" />
      <path d="M21 56h78l-6.2 24.6a6 6 0 0 1-5.8 4.4H33a6 6 0 0 1-5.8-4.4z" fill="#ececec" />
      <rect x="50" y="67" width="20" height="4" rx="2" fill="#8f8f8f" />
      <Twinkle x={100} y={20} size={14} />
    </>
  ),
  search: (
    <>
      <ellipse cx="60" cy="87" rx="36" ry="5" fill="#f0f0f0" />
      <rect x="28" y="14" width="52" height="68" rx="8" fill="#ececec" />
      <rect x="38" y="26" width="32" height="5" rx="2.5" fill="#d4d4d4" />
      <rect x="38" y="38" width="24" height="5" rx="2.5" fill="#d4d4d4" />
      <rect x="38" y="50" width="14" height="5" rx="2.5" fill="#d4d4d4" />
      <g className="animate-float">
        <circle cx="74" cy="58" r="14" fill="#ffffff" stroke="currentColor" strokeWidth="3" />
        <path d="M84.5 68.5 95 79" {...ink} strokeWidth={4.5} />
        <path d="M68 58h12" fill="none" stroke="#c4c4c4" strokeWidth="3" strokeLinecap="round" />
      </g>
      <Twinkle x={102} y={22} size={12} />
    </>
  ),
  transcript: (
    <>
      <ellipse cx="60" cy="87" rx="38" ry="5" fill="#f0f0f0" />
      <rect x="52" y="12" width="52" height="38" rx="9" fill="#dedede" />
      <path d="M88 49h10v10z" fill="#dedede" />
      <rect x="62" y="24" width="30" height="5" rx="2.5" fill="#c9c9c9" />
      <g className="animate-float">
        <path
          d="M24 34h46a8 8 0 0 1 8 8v24a8 8 0 0 1-8 8H46l-12 9v-9H24a8 8 0 0 1-8-8V42a8 8 0 0 1 8-8z"
          strokeLinejoin="round"
          {...sheet}
        />
        <path d="M28 49h38M28 60h22" {...ink} />
      </g>
      <Twinkle x={26} y={18} size={12} />
    </>
  ),
  notes: (
    <>
      <ellipse cx="60" cy="87" rx="36" ry="5" fill="#f0f0f0" />
      <rect x="28" y="18" width="64" height="68" rx="9" fill="#e6e6e6" />
      <rect x="48" y="12" width="24" height="11" rx="5.5" fill="#cfcfcf" />
      <g className="animate-float">
        <rect x="37" y="30" width="46" height="48" rx="6" {...sheet} />
        <path d="M46 43h28M46 54h28M46 65h15" {...ink} />
      </g>
      <Twinkle x={102} y={20} size={14} />
    </>
  ),
  terms: (
    <>
      <ellipse cx="60" cy="87" rx="38" ry="5" fill="#f0f0f0" />
      <rect x="18" y="24" width="50" height="22" rx="11" fill="#e6e6e6" />
      <path d="M30 35h26" fill="none" stroke="#b4b4b4" strokeWidth="4" strokeLinecap="round" />
      <path d="M24 35h38" fill="none" stroke="#8f8f8f" strokeWidth="2" strokeLinecap="round" />
      <path d="M44 50v8a4 4 0 0 0 4 4h4" fill="none" stroke="#c4c4c4" strokeWidth="3" strokeLinecap="round" />
      <g className="animate-float">
        <rect x="54" y="52" width="50" height="22" rx="11" {...sheet} />
        <path d="M66 63h26" {...ink} />
      </g>
      <Twinkle x={92} y={26} size={14} />
    </>
  ),
};

export function Empty({
  kind,
  text,
  size = "md",
  className,
}: {
  kind: EmptyKind;
  text: string;
  size?: "sm" | "md";
  className?: string;
}) {
  const width = size === "sm" ? 84 : 120;
  return (
    <div
      data-testid={`empty-${kind}`}
      className={cn(
        "flex min-h-0 flex-1 animate-rise flex-col items-center justify-center text-spark",
        size === "sm" ? "gap-2.5" : "gap-3.5",
        className,
      )}
    >
      <svg width={width} height={width * 0.8} viewBox="0 0 120 96" aria-hidden className="block shrink-0 overflow-visible">
        {art[kind]}
      </svg>
      <span className="text-ui text-muted-foreground">{text}</span>
    </div>
  );
}
