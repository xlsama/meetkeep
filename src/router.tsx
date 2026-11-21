import { createRootRoute, createRoute, createRouter, Navigate, Outlet } from "@tanstack/react-router";
import { Home } from "@/routes/home";
import { MainLayout } from "@/routes/main-layout";
import { MeetingPage } from "@/routes/meeting";
import { MenuPage } from "@/routes/menu";
import { MiniPage } from "@/routes/mini";
import { PanelPage } from "@/routes/panel";
import { SearchPage } from "@/routes/search";
import { SettingsPage } from "@/routes/settings";

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
  validateSearch: (search: Record<string, unknown>): { t?: number } => {
    const t = Number(search.t);
    return search.t !== undefined && Number.isFinite(t) && t >= 0 ? { t } : {};
  },
  component: MeetingPage,
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

const panelRoute = createRoute({ getParentRoute: () => rootRoute, path: "/panel", component: PanelPage });
const miniRoute = createRoute({ getParentRoute: () => rootRoute, path: "/mini", component: MiniPage });
const menuRoute = createRoute({ getParentRoute: () => rootRoute, path: "/menu", component: MenuPage });

const routeTree = rootRoute.addChildren([
  mainRoute.addChildren([homeRoute, meetingRoute, searchRoute, settingsRoute]),
  panelRoute,
  miniRoute,
  menuRoute,
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
