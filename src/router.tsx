import { createRootRoute, createRoute, createRouter, Navigate, Outlet } from "@tanstack/react-router";
import { Home } from "@/routes/home";
import { MainLayout } from "@/routes/main-layout";
import { MeetingPage } from "@/routes/meeting";
import { MenuPage } from "@/routes/menu";
import { MiniPage } from "@/routes/mini";
import { PanelPage } from "@/routes/panel";
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
  component: MeetingPage,
});

const settingsRoute = createRoute({ getParentRoute: () => mainRoute, path: "/settings", component: SettingsPage });

const panelRoute = createRoute({ getParentRoute: () => rootRoute, path: "/panel", component: PanelPage });
const miniRoute = createRoute({ getParentRoute: () => rootRoute, path: "/mini", component: MiniPage });
const menuRoute = createRoute({ getParentRoute: () => rootRoute, path: "/menu", component: MenuPage });

const routeTree = rootRoute.addChildren([
  mainRoute.addChildren([homeRoute, meetingRoute, settingsRoute]),
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
