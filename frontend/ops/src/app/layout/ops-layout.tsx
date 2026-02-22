import { useEffect, useState } from "react"
import { Outlet } from "react-router-dom"

import { clearOpsAccessKey, getOpsAccessKey, setOpsAccessKey } from "@/api/core/auth"
import { verifyOpsAccessKey } from "@/api/core/http"
import { StatusToast } from "@/components/common/status-toast"
import { OpsSidebar } from "@/components/layout/ops-sidebar"
import { useOpsUIStore } from "@/store/ops-ui-store"

// OpsLayout 负责侧边栏与页面主内容布局。
// 各页面通过 PageShell 自行管理标题、操作栏与内边距。
export function OpsLayout() {
  const collapsed = useOpsUIStore((state) => state.sidebarCollapsed)
  const setCollapsed = useOpsUIStore((state) => state.setSidebarCollapsed)
  const showNotice = useOpsUIStore((state) => state.showNotice)
  const initialAccessKey = getOpsAccessKey().trim()
  const [accessKey, setAccessKey] = useState(() => initialAccessKey)
  const [keySaved, setKeySaved] = useState(false)
  const [editingKey, setEditingKey] = useState(() => initialAccessKey.length === 0)
  const [savingKey, setSavingKey] = useState(false)

  useEffect(() => {
    let cancelled = false
    const bootstrap = async () => {
      if (!initialAccessKey) {
        return
      }
      const ok = await verifyOpsAccessKey(initialAccessKey)
      if (cancelled) {
        return
      }
      if (!ok) {
        setAccessKey("")
        setKeySaved(false)
        setEditingKey(true)
        clearOpsAccessKey()
        showNotice("error", "已保存密钥无效，请重新输入")
        return
      }
      setKeySaved(true)
      setEditingKey(false)
    }
    void bootstrap()
    return () => {
      cancelled = true
    }
  }, [initialAccessKey, showNotice])

  const saveKey = async () => {
    const key = accessKey.trim()
    if (!key) {
      setAccessKey("")
      setKeySaved(false)
      setEditingKey(true)
      clearOpsAccessKey()
      showNotice("error", "密钥为空，已清空输入")
      return
    }
    setSavingKey(true)
    try {
      const valid = await verifyOpsAccessKey(key)
      if (!valid) {
        setAccessKey("")
        setKeySaved(false)
        setEditingKey(true)
        clearOpsAccessKey()
        showNotice("error", "密钥无效或服务不可达，已清空输入")
        return
      }
      setOpsAccessKey(key)
      setAccessKey(key)
      setKeySaved(true)
      setEditingKey(false)
      showNotice("success", "访问密钥已保存")
    } catch {
      setAccessKey("")
      setKeySaved(false)
      setEditingKey(true)
      clearOpsAccessKey()
      showNotice("error", "密钥保存失败，已清空输入")
    } finally {
      setSavingKey(false)
    }
  }

  return (
    <div className="flex h-screen bg-background">
      <OpsSidebar
        collapsed={collapsed}
        onToggle={() => setCollapsed(!collapsed)}
        accessKey={accessKey}
        onAccessKeyChange={setAccessKey}
        keySaved={keySaved}
        editingKey={editingKey}
        savingKey={savingKey}
        onBeginEdit={() => setEditingKey(true)}
        onSaveAccessKey={() => void saveKey()}
      />
      {/* 主内容区：全高无内边距，各页面通过 PageShell 自行决定布局 */}
      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Outlet />
      </main>
      <StatusToast />
    </div>
  )
}
