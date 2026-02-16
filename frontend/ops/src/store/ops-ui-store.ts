import { create } from "zustand"

export type NoticeKind = "success" | "error"

export interface OpsNotice {
  type: NoticeKind
  message: string
}

interface OpsUIState {
  sidebarCollapsed: boolean
  selectedJobId: string
  selectedServiceLogFileId: string
  notice: OpsNotice | null
  setSidebarCollapsed: (value: boolean) => void
  setSelectedJobId: (value: string) => void
  setSelectedServiceLogFileId: (value: string) => void
  showNotice: (type: NoticeKind, message: string) => void
  clearNotice: () => void
}

// useOpsUIStore 维护 Ops 页共享 UI 状态。
export const useOpsUIStore = create<OpsUIState>((set) => ({
  sidebarCollapsed: false,
  selectedJobId: "",
  selectedServiceLogFileId: "",
  notice: null,
  setSidebarCollapsed: (value) => set({ sidebarCollapsed: value }),
  setSelectedJobId: (value) => set({ selectedJobId: value }),
  setSelectedServiceLogFileId: (value) => set({ selectedServiceLogFileId: value }),
  showNotice: (type, message) => set({ notice: { type, message } }),
  clearNotice: () => set({ notice: null }),
}))
