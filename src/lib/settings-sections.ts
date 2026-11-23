import { Bot, Captions, Disc, FileText, HardDrive, type LucideIcon } from "lucide-react";

export type SectionId = "transcription" | "notes" | "recording" | "storage" | "agent";

export const sections: { id: SectionId; label: string; icon: LucideIcon }[] = [
  { id: "transcription", label: "转写", icon: Captions },
  { id: "notes", label: "纪要与白板", icon: FileText },
  { id: "recording", label: "录制", icon: Disc },
  { id: "storage", label: "存储", icon: HardDrive },
  { id: "agent", label: "Agent", icon: Bot },
];
