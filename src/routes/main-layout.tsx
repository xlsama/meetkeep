import { useQuery } from "@tanstack/react-query";
import { Link, Outlet, useRouter, useRouterState } from "@tanstack/react-router";
import { AudioLines, House, LoaderCircle, PanelLeft, PanelLeftClose, Plus, Settings, Upload, Users } from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, type ReactNode } from "react";
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
import { useSheet } from "@/lib/ui";
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

  const setTitle = useHeader((s) => s.setTitle);
  const setActions = useHeader((s) => s.setActions);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const bare = pathname === "/" || pathname.startsWith("/meetings");
  const sheetOpen = useSheet((s) => s.open);
  const setSheetOpen = useSheet((s) => s.setOpen);
  const importer = useImporter();
  const importFile = useImport();
  const sidebar = useWorkspace((s) => s.sidebar);
  const toggleSidebar = useWorkspace((s) => s.toggleSidebar);

  return (
    <div className="flex h-full flex-col bg-canvas">
      <header className="drag flex h-[52px] shrink-0 items-center gap-3 pr-3 pl-[82px]">
        <HeaderButton label={sidebar ? "收起侧边栏" : "展开侧边栏"} testId="sidebar-toggle" onClick={toggleSidebar}>
          {sidebar ? <PanelLeftClose className="size-4" /> : <PanelLeft className="size-4" />}
        </HeaderButton>
        <div ref={setTitle} className="flex min-w-0 flex-1 basis-0 flex-col items-start justify-center" />
        <SearchTrigger />
        <div className="flex min-w-0 flex-1 basis-0 items-center justify-end gap-1">
          <div ref={setActions} className="flex min-w-0 items-center gap-2 pr-1.5" />
          <HeaderButton label="导入录音或视频" testId="import" busy={importer.importing} onClick={() => void importFile()}>
            {importer.importing ? <LoaderCircle className="size-4 animate-spin" /> : <Upload className="size-4" />}
          </HeaderButton>
          <HeaderButton label="新录制" testId="new-recording" pressed={sheetOpen} onClick={() => setSheetOpen(true)}>
            <Plus className="size-[18px]" />
          </HeaderButton>
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

const items = [
  { to: "/", label: "首页", icon: House, testId: "nav-home", exact: true },
  { to: "/meetings", label: "记录", icon: AudioLines, testId: "nav-records", exact: false },
  { to: "/people", label: "人物", icon: Users, testId: "nav-people", exact: false },
  { to: "/settings", label: "设置", icon: Settings, testId: "nav-settings", exact: false },
] as const;

function Sidebar({ open }: { open: boolean }) {
  const status = useRecorder((s) => s.status);

  return (
    <nav
      aria-label="功能"
      data-open={open}
      className={cn(
        "flex shrink-0 flex-col gap-0.5 overflow-hidden pl-1 transition-[width] duration-200 ease-out",
        open ? "w-[176px]" : "w-10",
      )}
    >
      {items.map((item) => (
        <Link
          key={item.to}
          to={item.to}
          data-testid={item.testId}
          aria-label={item.label}
          title={open ? undefined : item.label}
          activeOptions={{ exact: item.exact }}
          className="flex h-9 items-center gap-2.5 rounded-[10px] px-2.5 text-ui text-secondary-foreground transition-colors hover:bg-black/[0.05]"
          activeProps={{ className: "bg-background font-medium text-foreground hover:bg-background" }}
        >
          <item.icon className="size-4 shrink-0" />
          <span className={cn("whitespace-nowrap transition-opacity", !open && "opacity-0")}>{item.label}</span>
        </Link>
      ))}
      {isActive(status) && status.meeting && (
        <Link
          to="/meetings/$id"
          params={{ id: status.meeting.id }}
          data-testid="nav-live"
          className="mt-auto flex h-9 items-center rounded-[10px] px-2.5 transition-colors hover:bg-black/[0.05]"
        >
          {open ? (
            <RecordingLabel status={status} className="text-ui whitespace-nowrap" />
          ) : (
            <RecDot paused={status.phase === "paused"} className="mx-[5px] size-1.5" />
          )}
        </Link>
      )}
    </nav>
  );
}
