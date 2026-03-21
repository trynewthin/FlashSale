import { create } from "zustand"

export type NoticeKind = "success" | "error"

export interface OpsNotice {
  type: NoticeKind
  message: string
}

interface OpsUIState {
  sidebarCollapsed: boolean
  selectedJobId: string
  notice: OpsNotice | null
  setSidebarCollapsed: (value: boolean) => void
  setSelectedJobId: (value: string) => void
  showNotice: (type: NoticeKind, message: string) => void
  clearNotice: () => void
}

// useOpsUIStore 维护 Ops 页共享 UI 状态。
export const useOpsUIStore = create<OpsUIState>((set) => ({
  sidebarCollapsed: false,
  selectedJobId: "",
  notice: null,
  setSidebarCollapsed: (value) => set({ sidebarCollapsed: value }),
  setSelectedJobId: (value) => set({ selectedJobId: value }),
  showNotice: (type, message) => set({ notice: { type, message } }),
  clearNotice: () => set({ notice: null }),
}))
