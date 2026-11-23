import { useQuery } from "@tanstack/react-query";
import { Link, Outlet, useRouter, useRouterState } from "@tanstack/react-router";
import {
  ArrowLeft,
  AudioLines,
  LoaderCircle,
  PanelLeft,
  PanelLeftClose,
  Plus,
  Settings,
  Upload,
  Users,
} from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useRef, type ReactNode } from "react";
import { FileDrop } from "@/components/file-drop";
import { useHeader } from "@/components/header-slot";
import { RecordingLabel } from "@/components/meeting-card";
import { Notice } from "@/components/notice";
import { RecDot } from "@/components/rec-dot";
import { SearchDialog, SearchTrigger } from "@/components/search-dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { events } from "@/mygo";
import { useImport, useImporter } from "@/lib/importer";
import { settingsQuery } from "@/lib/queries";
import { isActive, useRecorder } from "@/lib/recorder";
import { sections } from "@/lib/settings-sections";
import { mac, useSheet, withCommand } from "@/lib/ui";
import { cn } from "@/lib/utils";
import { useWorkspace } from "@/lib/workspace";
import { NewSheet } from "@/routes/new-sheet";

export function MainLayout() {
  const router = useRouter();
  useQuery(settingsQuery);

  useEffect(() => {
    if (!isMyGo()) return;
    return events.navigate.on((route) => {
      router.history.push(route || "/");
    });
  }, [router]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "," || !withCommand(event)) return;
      event.preventDefault();
      router.history.push("/settings");
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [router]);

  const setTitle = useHeader((s) => s.setTitle);
  const setActions = useHeader((s) => s.setActions);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const bare = pathname.startsWith("/meetings");
  const sheetOpen = useSheet((s) => s.open);
  const setSheetOpen = useSheet((s) => s.setOpen);
  const importer = useImporter();
  const { pick } = useImport();
  const sidebar = useWorkspace((s) => s.sidebar);
  const toggleSidebar = useWorkspace((s) => s.toggleSidebar);

  return (
    <div className="flex h-full flex-col bg-canvas">
      <header
        className={cn(
          "drag flex h-[52px] shrink-0 items-center gap-3",
          mac ? "pr-3 pl-[82px]" : "pr-[calc(var(--mygo-titlebar-inset-right,0px)+8px)] pl-3",
        )}
      >
        <HeaderButton label={sidebar ? "收起侧边栏" : "展开侧边栏"} testId="sidebar-toggle" onClick={toggleSidebar}>
          {sidebar ? <PanelLeftClose className="size-4" /> : <PanelLeft className="size-4" />}
        </HeaderButton>
        <div ref={setTitle} className="flex min-w-0 flex-1 basis-0 flex-col items-start justify-center" />
        <SearchTrigger />
        <div className="flex min-w-0 flex-1 basis-0 items-center justify-end gap-1">
          <div ref={setActions} className="flex min-w-0 items-center gap-2 pr-1.5" />
          {pathname !== "/" && (
            <>
              <HeaderButton label="导入录音或视频" testId="import" busy={importer.importing} onClick={() => void pick()}>
                {importer.importing ? <LoaderCircle className="size-4 animate-spin" /> : <Upload className="size-4" />}
              </HeaderButton>
              <HeaderButton label="新录制" testId="new-recording" pressed={sheetOpen} onClick={() => setSheetOpen(true)}>
                <Plus className="size-[18px]" />
              </HeaderButton>
            </>
          )}
        </div>
      </header>
      <Notice
        testId="import-error"
        message={importer.error}
        onDismiss={importer.clear}
        className="mx-2 mb-2"
      />
      <div className="flex min-h-0 flex-1 gap-2 px-2 pb-2">
        <Sidebar open={sidebar} />
        <main className={cn("relative flex min-w-0 flex-1 flex-col gap-2", !bare && "card overflow-hidden")}>
          <Outlet />
        </main>
      </div>
      <NewSheet />
      <SearchDialog />
      <FileDrop />
    </div>
  );
}

function HeaderButton({
  label,
  testId,
  busy = false,
  pressed = false,
  onClick,
  children,
}: {
  label: string;
  testId: string;
  busy?: boolean;
  pressed?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-busy={busy}
          data-testid={testId}
          disabled={busy}
          onClick={onClick}
          className={cn(
            "inline-flex size-8 shrink-0 items-center justify-center rounded-[9px] text-secondary-foreground transition-colors outline-none hover:bg-black/[0.07] focus-visible:shadow-[0_0_0_3px_rgba(0,0,0,0.1)]",
            pressed && "bg-black/[0.07]",
          )}
        >
          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

const pages = [
  { to: "/", label: "记录", icon: AudioLines, testId: "nav-records", matches: (path: string) => path === "/" || path.startsWith("/meetings/") },
  { to: "/people", label: "人物", icon: Users, testId: "nav-people", matches: (path: string) => path.startsWith("/people") },
] as const;

const item =
  "flex h-9 shrink-0 items-center gap-2.5 rounded-[10px] px-2.5 text-ui text-secondary-foreground transition-colors outline-none hover:bg-black/[0.05] focus-visible:shadow-[inset_0_0_0_2px_rgba(0,0,0,0.12)]";
const current = { className: "bg-background font-medium text-foreground hover:bg-background" };

function Sidebar({ open }: { open: boolean }) {
  const router = useRouter();
  const status = useRecorder((s) => s.status);
  const href = useRouterState({ select: (s) => s.location.href });
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const settings = href.startsWith("/settings");
  const back = useRef("/");
  useEffect(() => {
    if (!settings) back.current = href;
  }, [settings, href]);
  const text = cn("whitespace-nowrap transition-opacity", !open && "opacity-0");

  return (
    <nav
      aria-label={settings ? "设置" : "导航"}
      data-open={open}
      className={cn(
        "flex shrink-0 flex-col gap-0.5 overflow-hidden pl-1 transition-[width] duration-200 ease-out",
        open ? "w-[176px]" : "w-10",
      )}
    >
      {settings ? (
        <>
          {sections.map((section) => (
            <Link
              key={section.id}
              to="/settings/$section"
              params={{ section: section.id }}
              data-testid={`settings-nav-${section.id}`}
              aria-label={section.label}
              title={open ? undefined : section.label}
              className={item}
              activeProps={current}
            >
              <section.icon className="size-4 shrink-0" />
              <span className={text}>{section.label}</span>
            </Link>
          ))}
          <button
            type="button"
            data-testid="settings-back"
            aria-label="返回"
            title={open ? undefined : "返回"}
            onClick={() => router.history.push(back.current)}
            className={cn(item, "mt-auto")}
          >
            <ArrowLeft className="size-4 shrink-0" />
            <span className={text}>返回</span>
          </button>
        </>
      ) : (
        <>
          {pages.map((page) => (
            <Link
              key={page.to}
              to={page.to}
              data-testid={page.testId}
              aria-label={page.label}
              title={open ? undefined : page.label}
              className={cn(item, page.matches(pathname) && current.className)}
            >
              <page.icon className="size-4 shrink-0" />
              <span className={text}>{page.label}</span>
            </Link>
          ))}
          <div className="mt-auto flex flex-col gap-0.5">
            {isActive(status) && status.meeting && (
              <Link to="/meetings/$id" params={{ id: status.meeting.id }} data-testid="nav-live" className={item}>
                {open ? (
                  <RecordingLabel status={status} className="text-ui whitespace-nowrap" />
                ) : (
                  <RecDot paused={status.phase === "paused"} className="mx-[5px] size-1.5" />
                )}
              </Link>
            )}
            <Link
              to="/settings"
              data-testid="nav-settings"
              aria-label="设置"
              title={open ? undefined : "设置"}
              className={item}
            >
              <Settings className="size-4 shrink-0" />
              <span className={text}>设置</span>
            </Link>
          </div>
        </>
      )}
    </nav>
  );
}
