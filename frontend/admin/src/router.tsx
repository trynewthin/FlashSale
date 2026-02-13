import { createBrowserRouter } from "react-router-dom"

import { AdminLayout } from "@/components/layout/admin-layout"
import { AuthGuard } from "@/components/layout/auth-guard"
import { LoginPage } from "@/pages/login"
import { DashboardPage } from "@/pages/dashboard"
import { AdminListPage } from "@/pages/admins"
import { RoleListPage } from "@/pages/roles"
import { AuditLogPage } from "@/pages/audit-logs"
import { UserManagementPage } from "@/pages/users"
import { ProductManagementPage } from "@/pages/products"
import { OrderManagementPage } from "@/pages/orders"
import { SeckillActivityListPage } from "@/pages/seckill/list"
import { SeckillActivityDetailPage } from "@/pages/seckill/detail"

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: (
      <AuthGuard>
        <AdminLayout />
      </AuthGuard>
    ),
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "admins", element: <AdminListPage /> },
      { path: "roles", element: <RoleListPage /> },
      { path: "audit-logs", element: <AuditLogPage /> },
      { path: "users", element: <UserManagementPage /> },
      { path: "products", element: <ProductManagementPage /> },
      { path: "orders", element: <OrderManagementPage /> },
      { path: "seckill", element: <SeckillActivityListPage /> },
      { path: "seckill/:activityId", element: <SeckillActivityDetailPage /> },
    ],
  },
])
