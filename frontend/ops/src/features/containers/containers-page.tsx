import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { ContainerRuntimeSnapshot, EtcdRegistrySnapshot } from "@/api/types"
import { useOpsApiError } from "@/hooks/use-ops-api-error"
import { useOpsUIStore } from "@/store/ops-ui-store"
import { ReplicaLogSheet } from "@/features/containers/replica-log-sheet"
import { ServiceTopologyCanvas } from "@/features/containers/service-topology-canvas"
import { TopologyLegendCard } from "@/features/containers/topology-legend-card"

// ContainersPageFeature 提供容器编排总页面，协调状态与操作回调。
export function ContainersPageFeature() {
  const showApiError = useOpsApiError()
  const showNotice = useOpsUIStore((state) => state.showNotice)

  const [snapshot, setSnapshot] = useState<ContainerRuntimeSnapshot | null>(null)
  const [loading, setLoading] = useState(false)
  const [actioningKey, setActioningKey] = useState("")
  const [focusService, setFocusService] = useState("")
  const [logSheet, setLogSheet] = useState({
    open: false,
    serviceName: "",
    containerName: "",
  })

  // 用 ref 存 focusService，避免 refresh 依赖 focusService 导致 effect 重置
  const focusServiceRef = useRef(focusService)
  focusServiceRef.current = focusService

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const data = await opsApi.getContainersStatus()
      setSnapshot(data)
      const currentFocus = focusServiceRef.current
      if (currentFocus && !data.services.some((item) => item.name === currentFocus)) {
        setFocusService("")
      }
    } catch (error) {
      showApiError(error, "加载容器状态失败")
    } finally {
      setLoading(false)
    }
  }, [showApiError])

  useEffect(() => {
    void refresh()
    const timer = window.setInterval(() => {
      void refresh()
    }, 15000)
    return () => window.clearInterval(timer)
  }, [refresh])

  // etcd 注册表（独立请求，不阻塞主流程）
  const [etcdSnap, setEtcdSnap] = useState<EtcdRegistrySnapshot | null>(null)
  const refreshEtcd = useCallback(() => {
    opsApi.getEtcdServices().then(setEtcdSnap).catch(() => { })
  }, [])
  useEffect(() => {
    refreshEtcd()
    const timer = window.setInterval(refreshEtcd, 15000)
    return () => window.clearInterval(timer)
  }, [refreshEtcd])

  // 构建 serviceKey → 注册实例数 映射
  const etcdInstanceMap = useMemo(() => {
    const map = new Map<string, number>()
    if (!etcdSnap?.services) return map
    for (const svc of etcdSnap.services) {
      map.set(svc.service_key, svc.instances.length)
    }
    return map
  }, [etcdSnap])

  // 稳定回调：使用 useCallback 避免每次 render 产生新引用
  const onActionContainer = useCallback(
    async (containerName: string, action: "start" | "stop" | "restart") => {
      const key = `${containerName}:${action}`
      try {
        setActioningKey(key)
        await opsApi.actionContainer(containerName, action)
        showNotice("success", `容器 ${containerName} ${action} 已执行`)
        await refresh()
      } catch (error) {
        showApiError(error, `容器操作失败：${containerName}`)
      } finally {
        setActioningKey("")
      }
    },
    [refresh, showNotice, showApiError]
  )

  const onScale = useCallback(
    async (service: string, replicas: number) => {
      try {
        const key = `scale:${service}`
        setActioningKey(key)
        await opsApi.scaleService(service, replicas)
        showNotice("success", `服务 ${service} 已扩缩容到 ${replicas}`)
        await refresh()
        refreshEtcd()
      } catch (error) {
        showApiError(error, `服务扩缩容失败：${service}`)
      } finally {
        setActioningKey("")
      }
    },
    [refresh, refreshEtcd, showNotice, showApiError]
  )

  const onOpenReplicaLogs = useCallback(
    (serviceName: string, containerName: string) => {
      setLogSheet({ open: true, serviceName, containerName })
    },
    []
  )

  return (
    <div className="relative h-full min-h-[720px] overflow-hidden bg-slate-50">
      <ServiceTopologyCanvas
        snapshot={snapshot}
        etcdInstanceMap={etcdInstanceMap}
        focusService={focusService}
        actioningKey={actioningKey}
        onActionContainer={onActionContainer}
        onScaleUpService={onScale}
        onScaleDownService={onScale}
        onOpenReplicaLogs={onOpenReplicaLogs}
        onFocusServiceChange={setFocusService}
      />

      <div className="absolute left-4 top-4 z-20">
        <TopologyLegendCard snapshot={snapshot} loading={loading} onRefresh={() => void refresh()} />
      </div>

      <ReplicaLogSheet
        open={logSheet.open}
        serviceName={logSheet.serviceName}
        containerName={logSheet.containerName}
        onOpenChange={(nextOpen) =>
          setLogSheet((prev) => ({
            ...prev,
            open: nextOpen,
          }))
        }
      />
    </div>
  )
}
