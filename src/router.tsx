import { createRootRoute, createRoute, createRouter, lazyRouteComponent, Navigate, Outlet } from "@tanstack/react-router";
import { Home } from "@/routes/home";
import { MainLayout } from "@/routes/main-layout";
import { MenuPage } from "@/routes/menu";
import { MiniPage } from "@/routes/mini";
import { PanelPage } from "@/routes/panel";
import { SearchPage } from "@/routes/search";
import { DetectPage } from "@/routes/detect";
import { PeoplePage } from "@/routes/people";
import { SettingsPage } from "@/routes/settings";
import { ToastPage } from "@/routes/toast";

const rootRoute = createRootRoute({ component: Outlet });

const mainRoute = createRoute({ getParentRoute: () => rootRoute, id: "main", component: MainLayout });

const homeRoute = createRoute({
  getParentRoute: () => mainRoute,
  path: "/",
  validateSearch: (search: Record<string, unknown>): { new?: 1 } =>
    search.new === 1 || search.new === "1" ? { new: 1 } : {},
  component: Home,
});

const meetingRoute = createRoute({
  getParentRoute: () => mainRoute,
  path: "/meetings/$id",
  validateSearch: (search: Record<string, unknown>): { t?: number; tab?: "board" } => {
    const t = Number(search.t);
    return {
      ...(search.t !== undefined && Number.isFinite(t) && t >= 0 && { t }),
      ...(search.tab === "board" && { tab: "board" as const }),
    };
  },
  component: lazyRouteComponent(() => import("@/routes/meeting"), "MeetingPage"),
});

const searchRoute = createRoute({
  getParentRoute: () => mainRoute,
  path: "/search",
  validateSearch: (search: Record<string, unknown>): { q: string } => ({
    q: typeof search.q === "string" ? search.q : typeof search.q === "number" ? String(search.q) : "",
  }),
  component: SearchPage,
});

const settingsRoute = createRoute({ getParentRoute: () => mainRoute, path: "/settings", component: SettingsPage });
const peopleRoute = createRoute({ getParentRoute: () => mainRoute, path: "/people", component: PeoplePage });

const panelRoute = createRoute({ getParentRoute: () => rootRoute, path: "/panel", component: PanelPage });
const miniRoute = createRoute({ getParentRoute: () => rootRoute, path: "/mini", component: MiniPage });
const menuRoute = createRoute({ getParentRoute: () => rootRoute, path: "/menu", component: MenuPage });
const toastRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/toast",
  validateSearch: (search: Record<string, unknown>): { meeting: string } => ({
    meeting: typeof search.meeting === "string" ? search.meeting : "",
  }),
  component: ToastPage,
});
const detectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/detect",
  validateSearch: (search: Record<string, unknown>): { app: string; name: string } => ({
    app: typeof search.app === "string" ? search.app : "",
    name: typeof search.name === "string" ? search.name : "",
  }),
  component: DetectPage,
});

const routeTree = rootRoute.addChildren([
  mainRoute.addChildren([homeRoute, meetingRoute, searchRoute, settingsRoute, peopleRoute]),
  panelRoute,
  miniRoute,
  menuRoute,
  toastRoute,
  detectRoute,
]);

export const router = createRouter({
  routeTree,
  defaultNotFoundComponent: () => <Navigate to="/" replace />,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
