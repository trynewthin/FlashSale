import { useCallback, useEffect, useState } from "react"

import { opsApi } from "@/api/modules/ops"
import type { ContainerRuntimeSnapshot } from "@/api/types"
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

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const data = await opsApi.getContainersStatus()
      setSnapshot(data)
      if (focusService && !data.services.some((item) => item.name === focusService)) {
        setFocusService("")
      }
    } catch (error) {
      showApiError(error, "加载容器状态失败")
    } finally {
      setLoading(false)
    }
  }, [focusService, showApiError])

  useEffect(() => {
    void refresh()
    const timer = window.setInterval(() => {
      void refresh()
    }, 3000)
    return () => window.clearInterval(timer)
  }, [refresh])

  const runContainerAction = async (containerName: string, action: "start" | "stop" | "restart") => {
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
  }

  const runScale = async (service: string, replicas: number) => {
    try {
      const key = `scale:${service}`
      setActioningKey(key)
      await opsApi.scaleService(service, replicas)
      showNotice("success", `服务 ${service} 已扩缩容到 ${replicas}`)
      await refresh()
    } catch (error) {
      showApiError(error, `服务扩缩容失败：${service}`)
    } finally {
      setActioningKey("")
    }
  }

  return (
    <div className="relative h-full min-h-[720px] overflow-hidden bg-slate-50">
      <ServiceTopologyCanvas
        snapshot={snapshot}
        focusService={focusService}
        actioningKey={actioningKey}
        onActionContainer={(containerName, action) => void runContainerAction(containerName, action)}
        onScaleUpService={(serviceName, targetReplicas) => void runScale(serviceName, targetReplicas)}
        onScaleDownService={(serviceName, targetReplicas) => void runScale(serviceName, targetReplicas)}
        onOpenReplicaLogs={(serviceName, containerName) =>
          setLogSheet({
            open: true,
            serviceName,
            containerName,
          })
        }
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
