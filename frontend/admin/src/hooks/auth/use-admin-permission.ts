// 权限判定 Hook：封装 domain 与 data_scope 的统一判断。
import { useMemo } from "react"

import { useAdminSessionState } from "@/hooks/auth/use-admin-session"

export function useAdminPermission() {
  const { domains, dataScope } = useAdminSessionState()

  const domainSet = useMemo(() => new Set(domains), [domains])

  const can = (domain: string): boolean => domainSet.has(domain)
  const canAny = (domainList: string[]): boolean => domainList.some((domain) => domainSet.has(domain))
  const isDataScopeAll = (): boolean => dataScope === "all"

  return {
    domains,
    dataScope,
    can,
    canAny,
    isDataScopeAll,
  }
}

