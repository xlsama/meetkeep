import { useForm, useStore } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { LoaderCircle, Mic, Volume2 } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { Channel } from "mygo-runtime";
import { useEffect, useState, type ReactNode } from "react";
import { z } from "zod";
import { LevelMeter, PlayingMeter } from "@/components/level-meter";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DialogTitle, Sheet } from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import { Switch } from "@/components/ui/switch";
import { Recorder, Sources, type AppRef, type Settings, type SourceApp } from "@/mygo";
import { appsQuery, errorMessage, microphoneQuery, settingsQuery } from "@/lib/queries";
import { useSheet } from "@/lib/ui";
import { cn } from "@/lib/utils";

const schema = z
  .object({
    title: z.string(),
    microphone: z.boolean(),
    system: z.boolean(),
    scope: z.enum(["all", "apps"]),
    apps: z.array(z.object({ id: z.string(), name: z.string() })),
    showPanel: z.boolean(),
  })
  .refine((v) => v.microphone || v.system, { path: ["microphone"] })
  .refine((v) => !v.system || v.scope === "all" || v.apps.length > 0, { path: ["apps"] });

export function NewSheet() {
  const open = useSheet((s) => s.open);
  const setOpen = useSheet((s) => s.setOpen);
  const settings = useQuery(settingsQuery);
  const [busy, setBusy] = useState(false);

  return (
    <Sheet
      open={open && settings.data !== undefined}
      onOpenChange={(next) => {
        if (!busy) setOpen(next);
      }}
      data-testid="new-sheet"
    >
      {settings.data && <SheetForm settings={settings.data} onBusy={setBusy} onDone={() => setOpen(false)} />}
    </Sheet>
  );
}

function SheetForm({
  settings,
  onBusy,
  onDone,
}: {
  settings: Settings;
  onBusy: (busy: boolean) => void;
  onDone: () => void;
}) {
  const navigate = useNavigate();
  const microphone = useQuery({ ...microphoneQuery, refetchOnMount: "always" });
  const apps = useQuery({ ...appsQuery, refetchOnMount: "always" });
  const [error, setError] = useState<string | null>(null);
  const listed = (apps.data ?? []).toSorted((a, b) => Number(b.playing) - Number(a.playing));

  const form = useForm({
    defaultValues: {
      title: "",
      microphone: settings.sources.microphone,
      system: settings.sources.system,
      scope: settings.sources.apps.length > 0 ? ("apps" as const) : ("all" as const),
      apps: settings.sources.apps,
      showPanel: settings.recording.showPanel,
    },
    validators: { onSubmit: schema },
    onSubmit: async ({ value }) => {
      setError(null);
      onBusy(true);
      const chosen = apps.data ? value.apps.filter((ref) => apps.data.some((app) => app.id === ref.id)) : value.apps;
      try {
        const meeting = await Recorder.start({
          title: value.title.trim(),
          microphone: value.microphone,
          system: value.system,
          apps: value.system && value.scope === "apps" ? chosen : [],
          showPanel: value.showPanel,
        });
        onBusy(false);
        onDone();
        void navigate({ to: "/meetings/$id", params: { id: meeting.id } });
      } catch (err) {
        onBusy(false);
        setError(errorMessage(err));
      }
    },
  });

  const system = useStore(form.store, (s) => s.values.system);
  const scope = useStore(form.store, (s) => s.values.scope);
  const submitting = useStore(form.store, (s) => s.isSubmitting);
  const valid = useStore(form.store, (s) => schema.safeParse(s.values).success);

  return (
    <form
      className="flex min-h-0 flex-col"
      onSubmit={(event) => {
        event.preventDefault();
        if (!form.state.isSubmitting) void form.handleSubmit();
      }}
    >
      <div className="flex min-h-0 flex-col gap-[18px] overflow-y-auto px-6 pt-[22px] pb-5">
        <DialogTitle className="m-0 text-title font-semibold">新录制</DialogTitle>
        <form.Field name="title">
          {(field) => (
            <label className="flex flex-col gap-1.5">
              <span className="text-label font-medium text-muted-foreground">标题</span>
              <input
                data-testid="sheet-title"
                placeholder="未命名会议"
                autoComplete="off"
                spellCheck={false}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                className="h-[38px] rounded-lg border border-input bg-background px-3 text-body text-foreground outline-none transition-[border-color,box-shadow] focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5"
              />
            </label>
          )}
        </form.Field>
        <div className="flex flex-col gap-1.5">
          <span className="text-label font-medium text-muted-foreground">声音来源</span>
          <div className="flex flex-col rounded-xl border border-line">
            <form.Field name="microphone">
              {(field) => (
                <SourceRow
                  icon={<Mic className="size-4 text-secondary-foreground" />}
                  title="麦克风"
                  detail={microphone.data ? `${microphone.data} · 记为「我」` : "记为「我」"}
                >
                  {field.state.value && <MicMeter />}
                  <Switch
                    data-testid="sheet-mic"
                    aria-label="录制麦克风"
                    checked={field.state.value}
                    onCheckedChange={field.handleChange}
                  />
                </SourceRow>
              )}
            </form.Field>
            <div className="h-px bg-border-subtle" />
            <form.Field name="system">
              {(field) => (
                <SourceRow
                  icon={<Volume2 className="size-4 text-secondary-foreground" />}
                  title="系统声音"
                  detail="记为「对方」"
                >
                  <Switch
                    data-testid="sheet-system"
                    aria-label="录制系统声音"
                    checked={field.state.value}
                    onCheckedChange={field.handleChange}
                  />
                </SourceRow>
              )}
            </form.Field>
            <AnimatePresence initial={false}>
              {system && (
                <motion.div
                  key="scope"
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: "auto", opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  transition={{ duration: 0.18 }}
                  className="overflow-hidden"
                >
                  <div className="flex flex-col gap-2 px-3.5 pb-3.5">
                    <form.Field name="scope">
                      {(field) => (
                        <Segmented
                          aria-label="声音范围"
                          className="flex"
                          value={field.state.value}
                          onValueChange={(value) => field.handleChange(value === "apps" ? "apps" : "all")}
                        >
                          <SegmentedItem value="all" data-testid="sheet-scope-all" className="flex-1">
                            全部
                          </SegmentedItem>
                          <SegmentedItem value="apps" data-testid="sheet-scope-apps" className="flex-1">
                            指定 App
                          </SegmentedItem>
                        </Segmented>
                      )}
                    </form.Field>
                    {scope === "apps" && (
                      <form.Field name="apps">
                        {(field) => (
                          <AppList
                            apps={listed}
                            loading={apps.isPending}
                            error={apps.error ? errorMessage(apps.error) : null}
                            selected={field.state.value}
                            onChange={field.handleChange}
                          />
                        )}
                      </form.Field>
                    )}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
        <form.Field name="showPanel">
          {(field) => (
            <label className="flex items-center gap-2.5 text-row">
              <Checkbox
                data-testid="sheet-show-panel"
                checked={field.state.value}
                onCheckedChange={(checked) => field.handleChange(checked === true)}
              />
              开始后显示侧边面板
            </label>
          )}
        </form.Field>
        {error && (
          <p role="alert" data-testid="sheet-error" data-selectable className="m-0 text-label break-words text-destructive">
            {error}
          </p>
        )}
      </div>
      <div className="flex shrink-0 justify-end gap-2 border-t border-divider px-5 py-3.5">
        <Button variant="outline" size="md" data-testid="sheet-cancel" disabled={submitting} onClick={onDone}>
          取消
        </Button>
        <Button
          type="submit"
          variant="primary"
          size="md"
          data-testid="sheet-submit"
          disabled={!valid}
          aria-busy={submitting}
          className="gap-2 pr-4 pl-3.5"
        >
          {submitting ? (
            <LoaderCircle className="size-[13px] animate-spin" />
          ) : (
            <span aria-hidden className="size-2 rounded-full bg-rec" />
          )}
          开始录制
        </Button>
      </div>
    </form>
  );
}

const micHeights = [4, 8, 12, 8, 4];

function MicMeter() {
  const [level, setLevel] = useState(0);
  useEffect(() => {
    const levels = new Channel<number>(setLevel);
    Sources.meter(levels).catch(() => {});
    return () => levels.close();
  }, []);
  return <LevelMeter level={level} on="#3d3d3d" off="#d9d9d9" heights={micHeights} />;
}

function SourceRow({
  icon,
  title,
  detail,
  children,
}: {
  icon: ReactNode;
  title: string;
  detail: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center gap-3 px-3.5 py-3">
      {icon}
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="text-body font-medium">{title}</span>
        <span className="truncate text-label text-muted-foreground">{detail}</span>
      </div>
      {children}
    </div>
  );
}

function AppList({
  apps,
  loading,
  error,
  selected,
  onChange,
}: {
  apps: SourceApp[];
  loading: boolean;
  error: string | null;
  selected: AppRef[];
  onChange: (apps: AppRef[]) => void;
}) {
  if (loading) {
    return (
      <div className="flex h-[38px] items-center justify-center text-muted-foreground">
        <LoaderCircle className="size-4 animate-spin" />
      </div>
    );
  }
  if (error) return <p className="m-0 px-2.5 text-label text-destructive">{error}</p>;
  return (
    <ScrollArea className="max-h-[206px]" viewportClassName="max-h-[206px]">
      <div className="flex flex-col gap-0.5">
        {apps.map((app) => {
          const checked = selected.some((ref) => ref.id === app.id);
          return (
            <label
              key={app.id}
              data-testid={`sheet-app-${app.id}`}
              className={cn(
                "flex h-[38px] shrink-0 items-center gap-2.5 rounded-md px-2.5 transition-colors",
                checked ? "bg-hover" : "hover:bg-muted",
              )}
            >
              <Checkbox
                checked={checked}
                onCheckedChange={(next) =>
                  onChange(
                    next === true
                      ? [...selected, { id: app.id, name: app.name }]
                      : selected.filter((ref) => ref.id !== app.id),
                  )
                }
              />
              <AppIcon app={app} />
              <span className="min-w-0 flex-1 truncate text-row">{app.name}</span>
              {app.playing && <PlayingMeter />}
            </label>
          );
        })}
      </div>
    </ScrollArea>
  );
}

function AppIcon({ app }: { app: SourceApp }) {
  if (app.icon) {
    return <img alt="" src={`data:image/png;base64,${app.icon}`} className="size-[22px] shrink-0 rounded-xs" />;
  }
  return (
    <span className="inline-flex size-[22px] shrink-0 items-center justify-center rounded-xs bg-line text-tiny font-semibold text-secondary-foreground">
      {Array.from(app.name)[0] ?? "?"}
    </span>
  );
}
