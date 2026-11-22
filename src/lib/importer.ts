import { useNavigate } from "@tanstack/react-router";
import { create } from "zustand";
import { Library } from "@/mygo";
import { errorMessage } from "@/lib/queries";

export const useImporter = create<{ importing: boolean; error: string | null; clear: () => void }>()((set) => ({
  importing: false,
  error: null,
  clear: () => set({ error: null }),
}));

export function useImport() {
  const navigate = useNavigate();
  return async () => {
    if (useImporter.getState().importing) return;
    useImporter.setState({ importing: true, error: null });
    try {
      const meeting = await Library.import();
      if (meeting) void navigate({ to: "/meetings/$id", params: { id: meeting.id } });
    } catch (err) {
      useImporter.setState({ error: errorMessage(err) });
    } finally {
      useImporter.setState({ importing: false });
    }
  };
}
