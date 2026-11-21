import { useForm, useStore } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronDown, LoaderCircle, X } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import { Switch } from "@/components/ui/switch";
import { Config, type Settings } from "@/mygo";
import { errorMessage, queryClient, settingsPathQuery, settingsQuery } from "@/lib/queries";
import { useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";

const address = z
  .string()
  .trim()
  .refine((value) => value === "" || /^https?:\/\/\S+$/.test(value), "地址要以 http:// 或 https:// 开头");

const schema = z.object({
  asr: z.object({ endpoint: address, key: z.string().trim(), hotwords: z.array(z.string()) }),
  llm: z.object({ baseUrl: address, key: z.string().trim(), model: z.string().trim() }),
  recording: z.object({
    directory: z.string(),
    detectMeetings: z.boolean(),
    showPanel: z.boolean(),
    echoCancellation: z.enum(["auto", "on", "off"]),
  }),
});

type Values = z.input<typeof schema>;

const pick = (s: Settings): Values => ({ asr: s.asr, llm: s.llm, recording: s.recording });

export function SettingsPage() {
  useWindowTitle("设置");
  const settings = useQuery(settingsQuery);
  return (
    <ScrollArea className="min-h-0 flex-1">
      <div className="flex w-full max-w-[880px] flex-col gap-6 px-10 pt-8 pb-10">
        <h1 className="m-0 text-display font-semibold">设置</h1>
        {settings.data && <SettingsForm settings={settings.data} />}
        {settings.error && <p className="m-0 text-label text-destructive">{errorMessage(settings.error)}</p>}
      </div>
    </ScrollArea>
  );
}

function SettingsForm({ settings }: { settings: Settings }) {
  const path = useQuery(settingsPathQuery);
  const [saveError, setSaveError] = useState<string | null>(null);
  const editing = useRef<string | null>(null);
  const form = useForm({ defaultValues: pick(settings) });

  useEffect(() => {
    const next = pick(settings);
    const sync = <K extends keyof Values, F extends keyof Values[K] & string>(group: K, field: F) => {
      const name = `${group}.${field}`;
      const value = next[group][field];
      const current = form.getFieldValue(name as never) as unknown;
      if (editing.current === name || JSON.stringify(current) === JSON.stringify(value)) return;
      form.setFieldValue(name as never, value as never, { dontUpdateMeta: true });
    };
    sync("asr", "endpoint");
    sync("asr", "key");
    sync("asr", "hotwords");
    sync("llm", "baseUrl");
    sync("llm", "key");
    sync("llm", "model");
    sync("recording", "directory");
    sync("recording", "detectMeetings");
    sync("recording", "showPanel");
    sync("recording", "echoCancellation");
  }, [settings, form]);

  const save = async () => {
    const parsed = schema.safeParse(form.state.values);
    const current = queryClient.getQueryData(settingsQuery.queryKey);
    if (!parsed.success || !current) return;
    const next: Settings = { ...current, ...parsed.data };
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    try {
      await Config.save(next);
      setSaveError(null);
    } catch (err) {
      setSaveError(errorMessage(err));
    }
  };

  const textProps = (name: string, blur: () => void) => ({
    onFocus: () => {
      editing.current = name;
    },
    onBlur: () => {
      editing.current = null;
      blur();
      void save();
    },
  });

  const asr = useStore(form.store, (s) => s.values.asr);
  const llm = useStore(form.store, (s) => s.values.llm);

  return (
    <>
      {saveError && (
        <p role="alert" data-testid="settings-error" className="m-0 -mt-3 text-label text-destructive">
          {saveError}
        </p>
      )}
      <Section title="转写">
        <Row label="服务">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                data-testid="settings-asr-service"
                className="flex h-[34px] flex-1 items-center justify-between rounded-md border border-input bg-background px-3 text-ui text-foreground outline-none transition-colors hover:bg-muted focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5"
              >
                MAI-Transcribe-2（Azure）
                <ChevronDown className="size-3.5 text-muted-foreground" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-(--radix-dropdown-menu-trigger-width)">
              <DropdownMenuItem>
                <span className="flex-1">MAI-Transcribe-2（Azure）</span>
                <Check />
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </Row>
        <form.Field name="asr.endpoint" validators={{ onBlur: address }}>
          {(field) => (
            <Row label="服务地址" error={field.state.meta.errors[0]?.message}>
              <Input
                aria-label="转写服务地址"
                data-testid="settings-asr-endpoint"
                placeholder="https://[你的资源名].cognitiveservices.azure.com"
                aria-invalid={field.state.meta.errors.length > 0}
                className="flex-1 font-mono text-label"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                {...textProps(field.name, field.handleBlur)}
              />
            </Row>
          )}
        </form.Field>
        <form.Field name="asr.key">
          {(field) => (
            <TestRow
              label="密钥"
              testId="settings-asr-test"
              signature={JSON.stringify([asr.endpoint, asr.key])}
              run={() => Config.testTranscription({ ...asr, hotwords: asr.hotwords })}
            >
              <Input
                type="password"
                aria-label="转写密钥"
                data-testid="settings-asr-key"
                className="flex-1"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                {...textProps(field.name, field.handleBlur)}
              />
            </TestRow>
          )}
        </form.Field>
        <form.Field name="asr.hotwords">
          {(field) => (
            <Row label="热词">
              <Hotwords
                value={field.state.value}
                onChange={(words) => {
                  field.handleChange(words);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
      </Section>

      <Section title="纪要与白板">
        <form.Field name="llm.baseUrl" validators={{ onBlur: address }}>
          {(field) => (
            <Row label="服务地址" error={field.state.meta.errors[0]?.message}>
              <Input
                aria-label="模型服务地址"
                data-testid="settings-llm-base-url"
                placeholder="https://api.deepseek.com"
                aria-invalid={field.state.meta.errors.length > 0}
                className="flex-1 font-mono text-label"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                {...textProps(field.name, field.handleBlur)}
              />
            </Row>
          )}
        </form.Field>
        <form.Field name="llm.key">
          {(field) => (
            <TestRow
              label="密钥"
              testId="settings-llm-test"
              signature={JSON.stringify([llm.baseUrl, llm.key, llm.model])}
              run={() => Config.testModel(llm)}
            >
              <Input
                type="password"
                aria-label="模型密钥"
                data-testid="settings-llm-key"
                className="flex-1"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                {...textProps(field.name, field.handleBlur)}
              />
            </TestRow>
          )}
        </form.Field>
        <form.Field name="llm.model">
          {(field) => (
            <Row label="模型名">
              <Input
                aria-label="模型名"
                data-testid="settings-llm-model"
                placeholder="deepseek-chat"
                className="flex-1 font-mono text-label"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                {...textProps(field.name, field.handleBlur)}
              />
            </Row>
          )}
        </form.Field>
      </Section>

      <Section title="录制">
        <form.Field name="recording.detectMeetings">
          {(field) => (
            <Row label="检测到会议时提醒">
              <Switch
                aria-label="检测到会议时提醒"
                data-testid="settings-detect-meetings"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
        <form.Field name="recording.showPanel">
          {(field) => (
            <Row label="开始后显示侧边面板">
              <Switch
                aria-label="开始后显示侧边面板"
                data-testid="settings-show-panel"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
        <Row label="侧边面板快捷键">
          <kbd className="inline-flex h-6 items-center rounded-[7px] border border-input bg-muted px-2 font-sans text-label text-secondary-foreground">
            ⌥⌘M
          </kbd>
        </Row>
        <form.Field name="recording.echoCancellation">
          {(field) => (
            <Row label="回声消除">
              <Segmented
                aria-label="回声消除"
                value={field.state.value}
                onValueChange={(value) => {
                  field.handleChange(value === "on" || value === "off" ? value : "auto");
                  void save();
                }}
              >
                <SegmentedItem value="auto" data-testid="settings-echo-auto">
                  自动
                </SegmentedItem>
                <SegmentedItem value="on" data-testid="settings-echo-on">
                  开
                </SegmentedItem>
                <SegmentedItem value="off" data-testid="settings-echo-off">
                  关
                </SegmentedItem>
              </Segmented>
            </Row>
          )}
        </form.Field>
        <form.Field name="recording.directory">
          {(field) => (
            <Row label="录音保存到">
              <span data-selectable className="min-w-0 flex-1 truncate font-mono text-label text-secondary-foreground">
                {field.state.value}
              </span>
              <Button
                variant="outline"
                size="md"
                data-testid="settings-directory-change"
                onClick={async () => {
                  try {
                    const directory = await Config.chooseDirectory();
                    if (!directory) return;
                    field.handleChange(directory);
                    await save();
                  } catch (err) {
                    setSaveError(errorMessage(err));
                  }
                }}
              >
                更改
              </Button>
            </Row>
          )}
        </form.Field>
        <Row label="配置文件">
          <span data-selectable className="min-w-0 flex-1 truncate font-mono text-label text-secondary-foreground">
            {path.data}
          </span>
          <Button
            variant="outline"
            size="md"
            data-testid="settings-open-file"
            onClick={() => {
              Config.openFile().catch((err: unknown) => setSaveError(errorMessage(err)));
            }}
          >
            打开
          </Button>
        </Row>
      </Section>
    </>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col rounded-2xl border border-line">
      <h2 className="m-0 px-4 pt-3.5 pb-3 text-body font-semibold">{title}</h2>
      {children}
    </section>
  );
}

function Row({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <div className="grid min-h-[52px] grid-cols-[140px_minmax(0,1fr)] items-center gap-x-4 border-t border-border-subtle px-4 py-2">
      <span className="text-ui text-secondary-foreground">{label}</span>
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex min-w-0 items-center gap-2">{children}</div>
        {error && <span className="text-meta text-invalid">{error}</span>}
      </div>
    </div>
  );
}

type TestState = { kind: "idle" } | { kind: "testing" } | { kind: "ok" } | { kind: "error"; message: string };

function TestRow({
  label,
  testId,
  signature,
  run,
  children,
}: {
  label: string;
  testId: string;
  signature: string;
  run: () => Promise<void>;
  children: ReactNode;
}) {
  const [state, setState] = useState<TestState & { signature: string }>({ kind: "idle", signature });
  const shown: TestState = state.signature === signature || state.kind === "testing" ? state : { kind: "idle" };

  const test = async () => {
    setState({ kind: "testing", signature });
    try {
      await run();
      setState({ kind: "ok", signature });
    } catch (err) {
      setState({ kind: "error", message: errorMessage(err), signature });
    }
  };

  return (
    <Row label={label}>
      {children}
      <Button
        variant="outline"
        size="md"
        data-testid={testId}
        disabled={shown.kind === "testing"}
        aria-busy={shown.kind === "testing"}
        onClick={() => void test()}
      >
        {shown.kind === "testing" && <LoaderCircle className="size-[13px] animate-spin" />}
        测试
      </Button>
      {shown.kind === "ok" && (
        <span data-testid={`${testId}-ok`} className="inline-flex shrink-0 items-center gap-[5px] text-label text-success-foreground">
          <span aria-hidden className="size-1.5 rounded-full bg-success" />
          可用
        </span>
      )}
      {shown.kind === "error" && (
        <span
          data-testid={`${testId}-error`}
          data-selectable
          title={shown.message}
          className="line-clamp-2 max-w-[42%] shrink-0 text-label leading-snug break-words text-destructive"
        >
          {shown.message}
        </span>
      )}
    </Row>
  );
}

function Hotwords({ value, onChange }: { value: string[]; onChange: (words: string[]) => void }) {
  const [draft, setDraft] = useState("");

  const add = () => {
    const words = draft
      .split(/[,，、\n]/)
      .map((word) => word.trim())
      .filter((word) => word !== "" && !value.includes(word));
    setDraft("");
    if (words.length > 0) onChange([...value, ...new Set(words)]);
  };

  return (
    <span className="flex flex-wrap items-center gap-1.5">
      {value.map((word) => (
        <span
          key={word}
          className="inline-flex h-[26px] items-center gap-1 rounded-full bg-secondary pr-1.5 pl-2.5 text-label text-secondary-foreground"
        >
          {word}
          <button
            type="button"
            aria-label={`删除${word}`}
            data-testid={`settings-hotword-remove-${word}`}
            onClick={() => onChange(value.filter((item) => item !== word))}
            className={cn("inline-flex size-[18px] items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground")}
          >
            <X className="size-3" />
          </button>
        </span>
      ))}
      <input
        aria-label="添加热词"
        data-testid="settings-asr-hotwords-input"
        placeholder="添加"
        spellCheck={false}
        autoComplete="off"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={add}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.nativeEvent.isComposing) {
            event.preventDefault();
            add();
          }
          if (event.key === "Backspace" && draft === "" && value.length > 0) onChange(value.slice(0, -1));
        }}
        className="h-[26px] w-20 border-0 bg-transparent text-label outline-none"
      />
    </span>
  );
}
