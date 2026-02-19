import { createBrowserRouter } from "react-router-dom"

import { UserLayout } from "@/components/layout/user-layout"
import { AuthGuard } from "@/components/layout/auth-guard"
import { LoginPage } from "@/pages/login"
import { HomePage } from "@/pages/home"
import { ProductListPage } from "@/pages/products/list"
import { ProductDetailPage } from "@/pages/products/detail"
import { OrderListPage } from "@/pages/orders/list"
import { OrderDetailPage } from "@/pages/orders/detail"
import { SeckillListPage } from "@/pages/seckill/list"
import { SeckillDetailPage } from "@/pages/seckill/detail"
import { ProfilePage } from "@/pages/profile"

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: (
      <AuthGuard>
        <UserLayout />
      </AuthGuard>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: "products", element: <ProductListPage /> },
      { path: "products/:productId", element: <ProductDetailPage /> },
      { path: "orders", element: <OrderListPage /> },
      { path: "orders/:orderId", element: <OrderDetailPage /> },
      { path: "seckill", element: <SeckillListPage /> },
      { path: "seckill/:activityId", element: <SeckillDetailPage /> },
      { path: "profile", element: <ProfilePage /> },
    ],
  },
])
