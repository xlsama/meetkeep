import { QueryClient, queryOptions } from "@tanstack/react-query";
import { Config, events, Library, People, Terms, Sources, Updates, type LLMSettings } from "@/mygo";
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

export const updateQuery = queryOptions({
  queryKey: ["update"],
  queryFn: () => Updates.ready(),
});

export const cliQuery = queryOptions({
  queryKey: ["cli"],
  queryFn: () => Config.cli(),
});

export const modelsQuery = (llm: LLMSettings) => {
  const custom = llm.services.find((s) => s.id === llm.service);
  return queryOptions({
    queryKey: ["models", ...(llm.provider === "custom" ? [custom?.baseUrl, custom?.key] : [llm.provider, llm[llm.provider].key])],
    queryFn: () => Config.models(llm),
    staleTime: 5 * 60_000,
  });
};

export const meetingsQuery = queryOptions({
  queryKey: ["meetings"],
  queryFn: () => Library.list(),
});

export const cardsQuery = queryOptions({
  queryKey: ["meetings", "cards"],
  queryFn: () => Library.cards(),
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

export const peopleQuery = queryOptions({
  queryKey: ["people"],
  queryFn: () => People.list(),
});

export const termsQuery = queryOptions({
  queryKey: ["terms"],
  queryFn: () => Terms.list(),
});

export const notesQuery = (id: string) =>
  queryOptions({
    queryKey: ["notes", id],
    queryFn: () => Library.notes(id),
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
    void queryClient.invalidateQueries({ queryKey: ["people"] });
    void queryClient.invalidateQueries({ queryKey: ["terms"] });
  });
  events.processing.on(applyProgress);
  events.notes.on((state) => {
    queryClient.setQueryData(notesQuery(state.meetingId).queryKey, state);
    if (!state.working) void queryClient.invalidateQueries({ queryKey: cardsQuery.queryKey });
  });
  events.settings.on((settings) => {
    queryClient.setQueryData(settingsQuery.queryKey, settings);
  });
  events.update.on((version) => {
    queryClient.setQueryData(updateQuery.queryKey, version);
  });
}

export function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  return String(error);
}
