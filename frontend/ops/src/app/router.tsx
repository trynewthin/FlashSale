import { Navigate, createBrowserRouter } from "react-router-dom"

import { OpsLayout } from "@/app/layout/ops-layout"
import { ContainersPage } from "@/pages/containers"
import { OverviewPage } from "@/pages/overview"
import { PerfTestPage } from "@/pages/perf-test"
import { RealtimePage } from "@/pages/realtime"
import { TasksPage } from "@/pages/tasks"

function resolveBasename() {
  const path = window.location.pathname
  if (path === "/ops" || path.startsWith("/ops/")) {
    return "/ops"
  }
  return "/"
}

// router 定义 Ops 侧边栏页面。
export const router = createBrowserRouter(
  [
    {
      path: "/",
      element: <OpsLayout />,
      children: [
        { index: true, element: <Navigate to="/overview" replace /> },
        { path: "overview", element: <OverviewPage /> },
        { path: "realtime", element: <RealtimePage /> },
        { path: "perf-test", element: <PerfTestPage /> },
        { path: "containers", element: <ContainersPage /> },
        { path: "tasks", element: <TasksPage /> },
        { path: "job-logs", element: <Navigate to="/tasks" replace /> },
      ],
    },
  ],
  { basename: resolveBasename() }
)

