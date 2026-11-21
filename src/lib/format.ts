import { format, isSameWeek, isSameYear, isToday, isYesterday } from "date-fns";
import { zhCN } from "date-fns/locale";
import type { AppRef, Track } from "@/mygo";

export function clock(ms: number) {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;
}

export function minutes(ms: number) {
  return `${Math.max(1, Math.round(ms / 60_000))} 分钟`;
}

export type DayGroup = "今天" | "昨天" | "本周" | "更早";

export function dayGroup(at: number, now = new Date()): DayGroup {
  if (isToday(at)) return "今天";
  if (isYesterday(at)) return "昨天";
  if (isSameWeek(at, now, { weekStartsOn: 1 })) return "本周";
  return "更早";
}

export function dayLabel(at: number, now = new Date()) {
  if (isToday(at)) return "今天";
  if (isYesterday(at)) return "昨天";
  if (isSameWeek(at, now, { weekStartsOn: 1 })) return format(at, "EEE", { locale: zhCN });
  if (isSameYear(at, now)) return format(at, "M月d日", { locale: zhCN });
  return format(at, "yyyy年M月d日", { locale: zhCN });
}

export function listDate(at: number, now = new Date()) {
  const group = dayGroup(at, now);
  return group === "今天" || group === "昨天" ? format(at, "HH:mm") : dayLabel(at, now);
}

export function meetingDate(at: number) {
  return `${dayLabel(at)} ${format(at, "HH:mm")}`;
}

export const speaker: Record<Track, string> = { mic: "我", system: "对方" };

export function systemLabel(apps: AppRef[]) {
  return apps.length > 0 ? apps.map((app) => app.name).join("、") : "全部系统声音";
}
