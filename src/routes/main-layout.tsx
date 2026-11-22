import { useQuery } from "@tanstack/react-query";
import { Link, Outlet, useNavigate, useRouter, useRouterState, useSearch } from "@tanstack/react-router";
import { AudioLines, House, Plus, Search, Settings, Users } from "lucide-react";
import { isMyGo } from "mygo-runtime";
import { useEffect, useRef, useState } from "react";
import { useHeader } from "@/components/header-slot";
import { RecordingLabel } from "@/components/meeting-card";
import { events } from "@/mygo";
import { settingsQuery } from "@/lib/queries";
import { isActive, useRecorder } from "@/lib/recorder";
import { useSheet } from "@/lib/ui";
import { cn } from "@/lib/utils";
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

  const setSlot = useHeader((s) => s.setSlot);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const bare = pathname === "/" || pathname.startsWith("/meetings");

  return (
    <div className="flex h-full flex-col bg-canvas">
      <header className="drag flex h-[52px] shrink-0 items-center pr-4 pl-[248px]">
        <div ref={setSlot} className="flex min-w-0 flex-1 items-center gap-2.5" />
      </header>
      <div className="flex min-h-0 flex-1 gap-2 px-2 pb-2">
        <Sidebar />
        <main className={cn("relative flex min-w-0 flex-1 flex-col gap-2", !bare && "card overflow-hidden")}>
          <Outlet />
        </main>
      </div>
      <NewSheet />
    </div>
  );
}

const items = [
  { to: "/", label: "首页", icon: House, testId: "nav-home", exact: true },
  { to: "/meetings", label: "记录", icon: AudioLines, testId: "nav-records", exact: false },
  { to: "/people", label: "人物", icon: Users, testId: "nav-people", exact: false },
  { to: "/settings", label: "设置", icon: Settings, testId: "nav-settings", exact: false },
] as const;

function Sidebar() {
  const status = useRecorder((s) => s.status);
  const sheetOpen = useSheet((s) => s.open);
  const setSheetOpen = useSheet((s) => s.setOpen);

  return (
    <nav aria-label="功能" className="card flex w-[200px] shrink-0 flex-col gap-px overflow-hidden p-2">
      <div className="flex items-center gap-1.5 pb-2.5">
        <SearchBox />
        <button
          type="button"
          aria-label="新录制"
          data-testid="new-recording"
          onClick={() => setSheetOpen(true)}
          className={cn(
            "inline-flex size-8 shrink-0 items-center justify-center rounded-[9px] text-secondary-foreground transition-colors hover:bg-selected",
            sheetOpen && "bg-selected",
          )}
        >
          <Plus className="size-[17px]" />
        </button>
      </div>
      {items.map((item) => (
        <Link
          key={item.to}
          to={item.to}
          data-testid={item.testId}
          activeOptions={{ exact: item.exact }}
          className="flex h-9 items-center gap-2.5 rounded-sm px-2.5 text-ui text-secondary-foreground transition-colors hover:bg-accent"
          activeProps={{ className: "bg-selected font-medium text-foreground hover:bg-selected" }}
        >
          <item.icon className="size-4 shrink-0" />
          <span>{item.label}</span>
        </Link>
      ))}
      {isActive(status) && status.meeting && (
        <Link
          to="/meetings/$id"
          params={{ id: status.meeting.id }}
          data-testid="nav-live"
          className="mt-auto flex h-9 items-center rounded-sm px-2.5 transition-colors hover:bg-accent"
        >
          <RecordingLabel status={status} className="text-ui" />
        </Link>
      )}
    </nav>
  );
}

function SearchBox() {
  const router = useRouter();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const href = useRouterState({ select: (s) => s.location.href });
  const { q } = useSearch({ strict: false });
  const searching = pathname === "/search";
  const [value, setValue] = useState(() => (searching ? (q ?? "") : ""));
  const back = useRef("/");
  const timer = useRef(0);

  useEffect(() => () => window.clearTimeout(timer.current), []);

  const go = (text: string) => {
    window.clearTimeout(timer.current);
    const query = text.trim();
    if (query === "") {
      if (searching) router.history.push(back.current);
      return;
    }
    if (!searching) back.current = href;
    void navigate({ to: "/search", search: { q: query }, replace: searching });
  };

  return (
    <label
      className={cn(
        "flex h-8 min-w-0 flex-1 items-center gap-2 rounded-[9px] bg-line px-2.5 text-muted-foreground transition-[background-color,box-shadow]",
        "focus-within:bg-background focus-within:shadow-[inset_0_0_0_1px_var(--color-border-strong)]",
        value !== "" && "bg-background shadow-[inset_0_0_0_1px_var(--color-border-strong)]",
      )}
    >
      <Search className="size-3.5 shrink-0" />
      <input
        type="search"
        placeholder="搜索"
        aria-label="搜索会议"
        data-testid="search"
        spellCheck={false}
        autoComplete="off"
        value={value}
        onChange={(event) => {
          const next = event.target.value;
          setValue(next);
          window.clearTimeout(timer.current);
          if (next.trim() === "") go(next);
          else timer.current = window.setTimeout(() => go(next), 300);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") go(value);
          if (event.key === "Escape" && value !== "") {
            setValue("");
            go("");
          }
        }}
        className="min-w-0 flex-1 border-0 bg-transparent text-ui text-foreground outline-none"
      />
    </label>
  );
}
