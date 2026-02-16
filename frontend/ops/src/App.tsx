import { RouterProvider } from "react-router-dom"

import { router } from "@/app/router"

// App 挂载 Ops 路由树。
export default function App() {
  return <RouterProvider router={router} />
}
