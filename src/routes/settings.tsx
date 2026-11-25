import { useForm, useStore } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { ArrowRight, Check, ChevronDown, Copy, LoaderCircle, Plus, Trash2, X } from "lucide-react";
import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { z } from "zod";
import { PageTitle } from "@/components/header-slot";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Segmented, SegmentedItem } from "@/components/ui/segmented";
import { Switch } from "@/components/ui/switch";
import { Config, Terms, type ASRProvider, type LLMProvider, type Service, type Settings, type Term } from "@/mygo";
import { copyText } from "@/lib/clipboard";
import { cliQuery, errorMessage, modelsQuery, queryClient, settingsPathQuery, settingsQuery, termsQuery } from "@/lib/queries";
import { sections, type SectionId } from "@/lib/settings-sections";
import { mac, shortcuts, useWindowTitle } from "@/lib/ui";
import { cn } from "@/lib/utils";

const address = z
  .string()
  .trim()
  .refine((value) => value === "" || /^https?:\/\/\S+$/.test(value), "地址需以 http:// 或 https:// 开头");

const service = z.object({ key: z.string().trim(), model: z.string().trim() });

const providers: { id: LLMProvider; label: string }[] = [
  { id: "anthropic", label: "Claude" },
  { id: "deepseek", label: "DeepSeek" },
  { id: "openrouter", label: "OpenRouter" },
];

const custom = z.object({
  id: z.string(),
  name: z.string().trim(),
  baseUrl: address,
  key: z.string().trim(),
  model: z.string().trim(),
});

const transcribers: { id: ASRProvider; label: string }[] = [
  { id: "mai", label: "MAI-Transcribe-2（Azure）" },
  { id: "deepseek", label: "DeepSeek" },
];

const schema = z.object({
  general: z.object({ openAtLogin: z.boolean(), menuBar: z.boolean() }),
  asr: z.object({
    provider: z.enum(["mai", "deepseek", "custom"]),
    endpoint: address,
    key: z.string().trim(),
    deepseek: z.object({ key: z.string().trim() }),
    service: z.string(),
    services: z.array(custom),
    hotwords: z.array(z.string()),
    polish: z.boolean(),
  }),
  llm: z.object({
    provider: z.enum(["anthropic", "deepseek", "openrouter", "custom"]),
    anthropic: service,
    deepseek: service,
    openrouter: service,
    service: z.string(),
    services: z.array(custom),
  }),
  recording: z.object({
    directory: z.string(),
    detectMeetings: z.boolean(),
    showPanel: z.boolean(),
    echoCancellation: z.enum(["auto", "on", "off"]),
    sounds: z.boolean(),
  }),
});

type Values = z.input<typeof schema>;

const skillCommand = "npx skills add xlsama/meetkeep --skill meetkeep -g";

const pick = (s: Settings): Values => ({ general: s.general, asr: s.asr, llm: s.llm, recording: s.recording });

export function SettingsPage() {
  useWindowTitle("设置");
  const { section: id } = useParams({ strict: false });
  const section = sections.find((s) => s.id === id) ?? sections[0];
  const settings = useQuery(settingsQuery);
  if (!section) return null;
  return (
    <ScrollArea className="min-h-0 flex-1">
      <div className="mx-auto flex w-full max-w-[640px] flex-col gap-5 px-6 pt-9 pb-12" data-testid={`settings-${section.id}`}>
        <PageTitle title="设置" />
        <h1 className="m-0 text-[18px] font-semibold">{section.label}</h1>
        {settings.data && <SettingsForm settings={settings.data} section={section.id} />}
        {settings.error && <p className="m-0 text-label text-destructive">{errorMessage(settings.error)}</p>}
      </div>
    </ScrollArea>
  );
}

function SettingsForm({ settings, section }: { settings: Settings; section: SectionId }) {
  const path = useQuery(settingsPathQuery);
  const cli = useQuery(cliQuery);
  const [copied, setCopied] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const editing = useRef<string | null>(null);
  const form = useForm({ defaultValues: pick(settings) });

  useEffect(() => {
    const next = pick(settings);
    const sync = <K extends keyof Values, F extends keyof Values[K] & string>(group: K, field: F) => {
      const name = `${group}.${field}`;
      const value = next[group][field];
      const current = form.getFieldValue(name as never) as unknown;
      if (editing.current === name || editing.current?.startsWith(`${name}.`)) return;
      if (JSON.stringify(current) === JSON.stringify(value)) return;
      form.setFieldValue(name as never, value as never, { dontUpdateMeta: true });
    };
    sync("general", "openAtLogin");
    sync("general", "menuBar");
    sync("asr", "provider");
    sync("asr", "endpoint");
    sync("asr", "key");
    sync("asr", "deepseek");
    sync("asr", "service");
    sync("asr", "services");
    sync("asr", "hotwords");
    sync("asr", "polish");
    sync("llm", "provider");
    sync("llm", "anthropic");
    sync("llm", "deepseek");
    sync("llm", "openrouter");
    sync("llm", "service");
    sync("llm", "services");
    sync("recording", "directory");
    sync("recording", "detectMeetings");
    sync("recording", "showPanel");
    sync("recording", "echoCancellation");
    sync("recording", "sounds");
  }, [settings, form]);

  const save = async () => {
    const parsed = schema.safeParse(form.state.values);
    const current = queryClient.getQueryData(settingsQuery.queryKey);
    if (!parsed.success || !current) return;
    const next: Settings = {
      ...current,
      ...parsed.data,
      recording: { ...current.recording, ...parsed.data.recording },
    };
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
  const asrCustom = asr.provider === "custom" ? asr.services.find((s) => s.id === asr.service) : undefined;
  const llmCustom = llm.provider === "custom" ? llm.services.find((s) => s.id === llm.service) : undefined;
  const builtin = llm.provider === "custom" ? undefined : llm.provider;
  const models = useQuery({
    ...modelsQuery({ ...settings.llm, provider: llm.provider, service: llm.service }),
    enabled: section === "notes" && (llm.provider !== "custom" || Boolean(llmCustom?.baseUrl)),
  });
  const transcriptionModels = useQuery({
    ...modelsQuery({ ...settings.llm, provider: "custom", service: asr.service, services: settings.asr.services }),
    enabled: section === "transcription" && Boolean(asrCustom?.baseUrl),
  });

  const choose = (group: "asr" | "llm", provider: ASRProvider | LLMProvider, service: string) => {
    form.setFieldValue(`${group}.provider`, provider as never);
    if (service) form.setFieldValue(`${group}.service`, service);
    void save();
  };
  const add = (group: "asr" | "llm") => {
    const services = form.getFieldValue(`${group}.services`);
    let name = "自定义服务";
    for (let n = 2; services.some((s) => s.name === name); n++) name = `自定义服务 ${n}`;
    const id = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`;
    form.setFieldValue(`${group}.services`, [...services, { id, name, baseUrl: "", key: "", model: "" }]);
    choose(group, "custom", id);
  };
  const put = (group: "asr" | "llm", next: Service) => {
    const services = form.getFieldValue(`${group}.services`);
    form.setFieldValue(`${group}.services`, services.map((s) => (s.id === next.id ? next : s)));
    void save();
  };
  const remove = (group: "asr" | "llm", id: string) => {
    const services = form.getFieldValue(`${group}.services`);
    form.setFieldValue(`${group}.services`, services.filter((s) => s.id !== id));
    form.setFieldValue(`${group}.service`, "");
    choose(group, group === "asr" ? "mai" : "deepseek", "");
  };

  return (
    <>
      {saveError && (
        <p role="alert" data-testid="settings-error" className="m-0 -mt-2 text-label text-destructive">
          {saveError}
        </p>
      )}
      {section === "general" && (
      <Group>
        <form.Field name="general.openAtLogin">
          {(field) => (
            <Row label="开机自启">
              <Switch
                aria-label="开机自启"
                data-testid="settings-open-at-login"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
        <form.Field name="general.menuBar">
          {(field) => (
            <Row label={mac ? "在菜单栏显示" : "在系统托盘显示"}>
              <Switch
                aria-label={mac ? "在菜单栏显示" : "在系统托盘显示"}
                data-testid="settings-menu-bar"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
      </Group>
      )}
      {section === "transcription" && (
      <>
      <Group>
        <Row label="服务">
          <ServiceMenu
            label="转写服务"
            testId="settings-asr-service"
            builtin={transcribers}
            provider={asr.provider}
            service={asr.service}
            services={asr.services}
            onChoose={(provider, service) => choose("asr", provider, service)}
            onAdd={() => add("asr")}
          />
        </Row>
        {asrCustom ? (
          <CustomService
            key={asrCustom.id}
            label="转写"
            testId="settings-asr"
            value={asrCustom}
            models={transcriptionModels.data ?? []}
            loading={transcriptionModels.isFetching}
            onChange={(next) => put("asr", next)}
            onRemove={() => remove("asr", asrCustom.id)}
            test={(tried) => Config.testTranscription({ ...asr, service: tried.id, services: [tried] })}
          />
        ) : asr.provider === "deepseek" ? (
          <form.Field name="asr.deepseek.key">
            {(field) => (
              <TestRow
                label="密钥"
                testId="settings-asr-test"
                signature={JSON.stringify(["deepseek", asr.deepseek.key])}
                run={() => Config.testTranscription(asr)}
              >
                <Input
                  type="password"
                  aria-label="转写密钥"
                  data-testid="settings-asr-key"
                  className="w-[252px]"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  {...textProps(field.name, field.handleBlur)}
                />
              </TestRow>
            )}
          </form.Field>
        ) : (
          <>
            <form.Field name="asr.endpoint" validators={{ onBlur: address }}>
              {(field) => (
                <Row label="服务地址" error={field.state.meta.errors[0]?.message}>
                  <Input
                    aria-label="转写服务地址"
                    data-testid="settings-asr-endpoint"
                    placeholder="https://[你的资源名].cognitiveservices.azure.com"
                    aria-invalid={field.state.meta.errors.length > 0}
                    className="w-[360px] font-mono text-label"
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
                  run={() => Config.testTranscription(asr)}
                >
                  <Input
                    type="password"
                    aria-label="转写密钥"
                    data-testid="settings-asr-key"
                    className="w-[252px]"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    {...textProps(field.name, field.handleBlur)}
                  />
                </TestRow>
              )}
            </form.Field>
          </>
        )}
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
        <form.Field name="asr.polish">
          {(field) => (
            <Row label="去掉语气词，纠正听错的术语">
              <Switch
                aria-label="去掉语气词，纠正听错的术语"
                data-testid="settings-asr-polish"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
      </Group>
      {asr.polish && <TermList />}
      </>
      )}
      {section === "notes" && (
      <Group>
        <Row label="服务">
          <ServiceMenu
            label="模型服务"
            testId="settings-llm-provider"
            builtin={providers}
            provider={llm.provider}
            service={llm.service}
            services={llm.services}
            onChoose={(provider, service) => choose("llm", provider, service)}
            onAdd={() => add("llm")}
          />
        </Row>
        {llmCustom && (
          <CustomService
            key={llmCustom.id}
            label="模型"
            testId="settings-llm"
            value={llmCustom}
            models={models.data ?? []}
            loading={models.isFetching}
            onChange={(next) => put("llm", next)}
            onRemove={() => remove("llm", llmCustom.id)}
            test={(tried) => Config.testModel({ ...llm, service: tried.id, services: [tried] })}
          />
        )}
        {builtin && (
          <>
            <form.Field key={`${builtin}.key`} name={`llm.${builtin}.key`}>
              {(field) => (
                <TestRow
                  label="密钥"
                  testId="settings-llm-test"
                  signature={JSON.stringify([builtin, llm[builtin]])}
                  run={() => Config.testModel(llm)}
                >
                  <Input
                    type="password"
                    aria-label="模型密钥"
                    data-testid="settings-llm-key"
                    className="w-[252px]"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    {...textProps(field.name, field.handleBlur)}
                  />
                </TestRow>
              )}
            </form.Field>
            <form.Field key={`${builtin}.model`} name={`llm.${builtin}.model`}>
              {(field) => (
                <Row label="模型">
                  <Combobox
                    aria-label="模型"
                    testId="settings-llm-model"
                    className="w-[360px]"
                    value={field.state.value}
                    options={models.data ?? []}
                    loading={models.isFetching}
                    onValueChange={field.handleChange}
                    onPick={(model) => {
                      field.handleChange(model);
                      void save();
                    }}
                    {...textProps(field.name, field.handleBlur)}
                  />
                </Row>
              )}
            </form.Field>
          </>
        )}
      </Group>
      )}
      {section === "recording" && (
      <Group>
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
            {shortcuts.panel}
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
        <form.Field name="recording.sounds">
          {(field) => (
            <Row label="提示音">
              <Switch
                aria-label="提示音"
                data-testid="settings-sounds"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(checked);
                  void save();
                }}
              />
            </Row>
          )}
        </form.Field>
      </Group>
      )}
      {section === "storage" && (
      <Group>
        <form.Field name="recording.directory">
          {(field) => (
            <Row label="录音保存到">
              <span data-selectable className="min-w-0 truncate font-mono text-label text-secondary-foreground">
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
          <span data-selectable className="min-w-0 truncate font-mono text-label text-secondary-foreground">
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
      </Group>
      )}

      {section === "agent" && (
      <Group>
        <Row label="命令行工具">
          {cli.data ? (
            <span data-selectable data-testid="settings-cli-path" className="min-w-0 truncate font-mono text-label text-secondary-foreground">
              {cli.data}
            </span>
          ) : (
            <Button
              variant="outline"
              size="md"
              data-testid="settings-cli-install"
              onClick={async () => {
                try {
                  queryClient.setQueryData(cliQuery.queryKey, await Config.installCLI());
                  setSaveError(null);
                } catch (err) {
                  setSaveError(errorMessage(err));
                }
              }}
            >
              安装
            </Button>
          )}
        </Row>
        <Row label="Skill">
          <span data-selectable data-testid="settings-skill-command" className="min-w-0 truncate font-mono text-label text-secondary-foreground">
            {skillCommand}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="复制"
            title="复制"
            data-testid="settings-skill-copy"
            onClick={async () => {
              try {
                await copyText(skillCommand);
                setSaveError(null);
                setCopied(true);
                window.setTimeout(() => setCopied(false), 1600);
              } catch (err) {
                setSaveError(errorMessage(err));
              }
            }}
          >
            {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
          </Button>
        </Row>
      </Group>
      )}
    </>
  );
}

function ServiceMenu<P extends string>({
  label,
  testId,
  builtin,
  provider,
  service,
  services,
  onChoose,
  onAdd,
}: {
  label: string;
  testId: string;
  builtin: { id: P; label: string }[];
  provider: P | "custom";
  service: string;
  services: Service[];
  onChoose: (provider: P | "custom", service: string) => void;
  onAdd: () => void;
}) {
  const chosen = provider === "custom" ? services.find((s) => s.id === service) : undefined;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" aria-label={label} data-testid={testId} className="flex h-8 items-center gap-2 rounded-md border border-input bg-background pr-2.5 pl-3 text-ui text-foreground outline-none transition-colors hover:bg-muted focus-visible:border-border-strong focus-visible:ring-[3px] focus-visible:ring-black/5">
          {chosen ? chosen.name || "未命名" : builtin.find((p) => p.id === provider)?.label}
          <ChevronDown className="size-3.5 text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {builtin.map((p) => (
          <DropdownMenuItem key={p.id} data-testid={`${testId}-${p.id}`} onSelect={() => onChoose(p.id, "")}>
            <span className="flex-1">{p.label}</span>
            {p.id === provider && <Check />}
          </DropdownMenuItem>
        ))}
        {services.map((s, i) => (
          <DropdownMenuItem key={s.id} data-testid={`${testId}-custom-${i}`} onSelect={() => onChoose("custom", s.id)}>
            <span className="flex-1">{s.name || "未命名"}</span>
            {s.id === chosen?.id && <Check />}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem data-testid={`${testId}-add`} onSelect={onAdd}>
          <Plus />
          添加自定义服务
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CustomService({
  label,
  testId,
  value,
  models,
  loading,
  onChange,
  onRemove,
  test,
}: {
  label: string;
  testId: string;
  value: Service;
  models: string[];
  loading: boolean;
  onChange: (next: Service) => void;
  onRemove: () => void;
  test: (tried: Service) => Promise<void>;
}) {
  const [draft, setDraft] = useState(value);
  const [invalid, setInvalid] = useState<string>();
  const commit = (next: Service) => {
    const parsed = address.safeParse(next.baseUrl);
    setInvalid(parsed.success ? undefined : parsed.error.issues[0]?.message);
    if (parsed.success && JSON.stringify(next) !== JSON.stringify(value)) onChange(next);
  };
  return (
    <>
      <Row label="名称">
        <Input
          aria-label={`${label}服务名称`}
          data-testid={`${testId}-name`}
          className="w-[316px]"
          value={draft.name}
          onChange={(event) => setDraft({ ...draft, name: event.target.value })}
          onBlur={() => commit(draft)}
        />
        <Button
          variant="ghost"
          size="icon"
          aria-label="删除这个服务"
          data-testid={`${testId}-remove`}
          onClick={onRemove}
        >
          <Trash2 className="size-4" />
        </Button>
      </Row>
      <Row label="服务地址" error={invalid}>
        <Input
          aria-label={`${label}服务地址`}
          data-testid={`${testId}-base-url`}
          placeholder="https://api.openai.com/v1"
          aria-invalid={invalid !== undefined}
          className="w-[360px] font-mono text-label"
          value={draft.baseUrl}
          onChange={(event) => setDraft({ ...draft, baseUrl: event.target.value })}
          onBlur={() => commit(draft)}
        />
      </Row>
      <TestRow
        label="密钥"
        testId={`${testId}-test`}
        signature={JSON.stringify([draft.baseUrl, draft.key, draft.model])}
        run={() => test(draft)}
      >
        <Input
          type="password"
          aria-label={`${label}密钥`}
          data-testid={`${testId}-key`}
          className="w-[252px]"
          value={draft.key}
          onChange={(event) => setDraft({ ...draft, key: event.target.value })}
          onBlur={() => commit(draft)}
        />
      </TestRow>
      <Row label="模型">
        <Combobox
          aria-label={`${label}服务的模型`}
          testId={`${testId}-model`}
          className="w-[360px]"
          value={draft.model}
          options={models}
          loading={loading}
          onValueChange={(model) => setDraft({ ...draft, model })}
          onPick={(model) => {
            setDraft({ ...draft, model });
            commit({ ...draft, model });
          }}
          onBlur={() => commit(draft)}
        />
      </Row>
    </>
  );
}

function Group({ children }: { children: ReactNode }) {
  return <section className="flex flex-col rounded-xl border border-line">{children}</section>;
}

function Row({
  label,
  error,
  note,
  children,
}: {
  label: string;
  error?: string;
  note?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-[52px] items-center gap-4 border-t border-border-subtle px-4 py-2.5 first:border-t-0">
      <span className="shrink-0 text-ui text-foreground">{label}</span>
      <div className="flex min-w-0 flex-1 flex-col items-end gap-1">
        <div className="flex max-w-full min-w-0 items-center justify-end gap-2">{children}</div>
        {error && <span className="text-meta text-invalid">{error}</span>}
        {note}
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
    <Row
      label={label}
      note={
        shown.kind === "error" ? (
          <span
            data-testid={`${testId}-error`}
            data-selectable
            title={shown.message}
            className="line-clamp-2 max-w-[380px] text-right text-label leading-snug break-words text-destructive"
          >
            {shown.message}
          </span>
        ) : undefined
      }
    >
      {shown.kind === "ok" && (
        <span
          data-testid={`${testId}-ok`}
          className="inline-flex shrink-0 items-center gap-[5px] text-label text-success-foreground"
        >
          <span aria-hidden className="size-1.5 rounded-full bg-success" />
          可用
        </span>
      )}
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
    </Row>
  );
}

function TermList() {
  const { data: terms = [] } = useQuery(termsQuery);
  const [heard, setHeard] = useState("");
  const [meant, setMeant] = useState("");
  const [error, setError] = useState<string>();

  const act = async (change: () => Promise<void>) => {
    try {
      await change();
      setError(undefined);
      await queryClient.invalidateQueries({ queryKey: termsQuery.queryKey });
    } catch (err) {
      setError(errorMessage(err));
    }
  };
  const add = () =>
    act(async () => {
      await Terms.confirm(heard, meant);
      setHeard("");
      setMeant("");
    });
  const enter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !event.nativeEvent.isComposing) void add();
  };

  return (
    <Group>
      <Row label="术语" error={error}>
        <Input
          aria-label="听成的写法"
          data-testid="settings-term-heard"
          placeholder="听成"
          spellCheck={false}
          autoComplete="off"
          className="w-[120px]"
          value={heard}
          onChange={(event) => setHeard(event.target.value)}
          onKeyDown={enter}
        />
        <ArrowRight className="size-3.5 shrink-0 text-muted-foreground" />
        <Input
          aria-label="正确的写法"
          data-testid="settings-term-meant"
          placeholder="应为"
          spellCheck={false}
          autoComplete="off"
          className="w-[120px]"
          value={meant}
          onChange={(event) => setMeant(event.target.value)}
          onKeyDown={enter}
        />
        <Button
          variant="ghost"
          size="icon"
          aria-label="添加术语"
          title="添加"
          data-testid="settings-term-add"
          disabled={heard.trim() === "" || meant.trim() === ""}
          onClick={() => void add()}
        >
          <Plus className="size-4" />
        </Button>
      </Row>
      {terms.map((term) => (
        <TermRow key={`${term.heard}\u0000${term.meant}`} term={term} act={act} />
      ))}
    </Group>
  );
}

function TermRow({ term, act }: { term: Term; act: (change: () => Promise<void>) => Promise<void> }) {
  const id = `${term.heard}-${term.meant}`;
  return (
    <div
      data-testid={`settings-term-${id}`}
      data-state={term.state || "learned"}
      className="flex min-h-[44px] items-center gap-2 border-t border-border-subtle py-1.5 pr-3 pl-4"
    >
      <span
        className={cn(
          "flex min-w-0 flex-1 items-center gap-2 text-ui",
          term.state === "dropped" ? "text-muted-foreground line-through" : "text-foreground",
        )}
      >
        <span className="truncate">{term.heard}</span>
        <ArrowRight className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{term.meant}</span>
      </span>
      {term.meetings > 0 && <span className="shrink-0 text-meta text-muted-foreground tabular-nums">{term.meetings} 场</span>}
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={`确认${term.heard}应为${term.meant}`}
        aria-pressed={term.state === "confirmed"}
        title="是听错的"
        data-testid={`settings-term-confirm-${id}`}
        className={cn(term.state === "confirmed" && "bg-secondary text-foreground")}
        onClick={() => void act(() => Terms.confirm(term.heard, term.meant))}
      >
        <Check className="size-4" />
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={`${term.heard}不是听错的`}
        aria-pressed={term.state === "dropped"}
        title="不是听错的"
        data-testid={`settings-term-drop-${id}`}
        className={cn(term.state === "dropped" && "bg-secondary text-foreground")}
        onClick={() => void act(() => Terms.drop(term.heard, term.meant))}
      >
        <X className="size-4" />
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={`删除${term.heard}`}
        title="删除"
        data-testid={`settings-term-forget-${id}`}
        onClick={() => void act(() => Terms.forget(term.heard, term.meant))}
      >
        <Trash2 className="size-4" />
      </Button>
    </div>
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
    <span className="flex flex-wrap items-center justify-end gap-1.5">
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
