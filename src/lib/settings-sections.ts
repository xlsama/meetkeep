import { Bot, Captions, Disc, FileText, HardDrive, Settings2, type LucideIcon } from "lucide-react";

export type SectionId = "general" | "transcription" | "notes" | "recording" | "storage" | "agent";

export const sections: { id: SectionId; label: string; icon: LucideIcon }[] = [
  { id: "general", label: "通用", icon: Settings2 },
  { id: "transcription", label: "转写", icon: Captions },
  { id: "notes", label: "纪要与白板", icon: FileText },
  { id: "recording", label: "录制", icon: Disc },
  { id: "storage", label: "存储", icon: HardDrive },
  { id: "agent", label: "Agent", icon: Bot },
];
