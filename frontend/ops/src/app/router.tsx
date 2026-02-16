import { Navigate, createBrowserRouter } from "react-router-dom"

import { OpsLayout } from "@/app/layout/ops-layout"
import { ContainersPage } from "@/pages/containers"
import { JobLogsPage } from "@/pages/job-logs"
import { OverviewPage } from "@/pages/overview"
import { RealtimePage } from "@/pages/realtime"
import { ServiceLogsPage } from "@/pages/service-logs"
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
        { path: "containers", element: <ContainersPage /> },
        { path: "tasks", element: <TasksPage /> },
        { path: "job-logs", element: <JobLogsPage /> },
        { path: "service-logs", element: <ServiceLogsPage /> },
      ],
    },
  ],
  { basename: resolveBasename() }
)
