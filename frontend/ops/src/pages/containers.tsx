import { ContainersPageFeature } from "@/features/containers/containers-page"

// ContainersPage 作为路由层包装，实际实现位于 features/containers。
export function ContainersPage() {
  return <ContainersPageFeature />
}
