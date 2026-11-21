import { QueryClient, queryOptions } from "@tanstack/react-query";
import { Config, events, Library, Sources } from "@/mygo";
import { applyProgress } from "@/lib/progress";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false, refetchOnWindowFocus: false, staleTime: Infinity },
  },
});

export const settingsQuery = queryOptions({
  queryKey: ["settings"],
  queryFn: () => Config.get(),
});

export const settingsPathQuery = queryOptions({
  queryKey: ["settings-path"],
  queryFn: () => Config.path(),
});

export const meetingsQuery = queryOptions({
  queryKey: ["meetings"],
  queryFn: () => Library.list(),
});

export const meetingQuery = (id: string) =>
  queryOptions({
    queryKey: ["meeting", id],
    queryFn: () => Library.get(id),
  });

export const searchQuery = (query: string) =>
  queryOptions({
    queryKey: ["search", query],
    queryFn: () => Library.search(query),
  });

export const microphoneQuery = queryOptions({
  queryKey: ["microphone"],
  queryFn: () => Sources.microphone(),
  staleTime: 0,
});

export const appsQuery = queryOptions({
  queryKey: ["apps"],
  queryFn: () => Sources.apps(),
  staleTime: 30_000,
});

let synced = false;

export function syncQueries() {
  if (synced) return;
  synced = true;
  events.library.on(() => {
    void queryClient.invalidateQueries({ queryKey: ["meetings"] });
    void queryClient.invalidateQueries({ queryKey: ["meeting"] });
    void queryClient.invalidateQueries({ queryKey: ["search"] });
  });
  events.processing.on(applyProgress);
  events.settings.on((settings) => {
    queryClient.setQueryData(settingsQuery.queryKey, settings);
  });
}

export function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  return String(error);
}
