import { useNavigate } from "@tanstack/react-router";
import { create } from "zustand";
import { Library, type Meeting } from "@/mygo";
import { errorMessage } from "@/lib/queries";

export const useImporter = create<{ importing: boolean; error: string | null; clear: () => void }>()((set) => ({
  importing: false,
  error: null,
  clear: () => set({ error: null }),
}));

const media = new Set([
  "aac", "aif", "aiff", "caf", "flac", "m4a", "m4v", "mkv", "mov", "mp3", "mp4", "oga", "ogg", "opus", "wav", "webm",
]);

function extension(path: string) {
  const name = path.slice(path.lastIndexOf("/") + 1);
  const dot = name.lastIndexOf(".");
  return dot < 0 ? "" : name.slice(dot + 1).toLowerCase();
}

export function useImport() {
  const navigate = useNavigate();

  const run = async (work: () => Promise<Meeting[]>) => {
    if (useImporter.getState().importing) return;
    useImporter.setState({ importing: true, error: null });
    try {
      const [first, ...rest] = await work();
      if (first && rest.length === 0) void navigate({ to: "/meetings/$id", params: { id: first.id } });
      else if (first) void navigate({ to: "/" });
    } catch (err) {
      useImporter.setState({ error: errorMessage(err) });
    } finally {
      useImporter.setState({ importing: false });
    }
  };

  return {
    pick: () =>
      run(async () => {
        const meeting = await Library.import();
        return meeting ? [meeting] : [];
      }),
    drop: (paths: string[]) =>
      run(async () => {
        const files = paths.filter((path) => media.has(extension(path)));
        if (files.length === 0) throw new Error("只能导入录音或视频文件");
        const meetings: Meeting[] = [];
        for (const path of files) meetings.push(await Library.importFile(path));
        return meetings;
      }),
  };
}
