import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { MotionConfig } from "motion/react";
import { isMyGo } from "mygo-runtime";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { TooltipProvider } from "@/components/ui/tooltip";
import { queryClient, syncQueries } from "@/lib/queries";
import { startRecorder } from "@/lib/recorder";
import { router } from "@/router";
import "@/style.css";

const surfaces = ["panel", "mini", "menu"] as const;
const surface = surfaces.find((name) => location.pathname.startsWith(`/${name}`)) ?? "main";
document.documentElement.dataset.window = surface;

if (isMyGo()) {
  syncQueries();
  startRecorder();
}

const root = document.getElementById("root");
if (root) {
  createRoot(root).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <MotionConfig reducedMotion="user">
          <TooltipProvider>
            <RouterProvider router={router} />
          </TooltipProvider>
        </MotionConfig>
      </QueryClientProvider>
    </StrictMode>,
  );
}
