import type { TaskDef } from "@/api/types"

export type RealtimeTestFieldType = "number" | "duration" | "text" | "select"

export interface RealtimeTestFieldOption {
  value: string
  label: string
}

export interface RealtimeTestFieldDef {
  flag: string
  label: string
  type: RealtimeTestFieldType
  helper?: string
  placeholder?: string
  min?: number
  max?: number
  step?: number
  required?: boolean
  options?: RealtimeTestFieldOption[]
}

export interface RealtimeTestPreset {
  taskID: string
  title: string
  subtitle: string
  description: string
  recommended?: boolean
  fields: RealtimeTestFieldDef[]
}

const realtimeTestPresets: Record<string, RealtimeTestPreset> = {
  "perf.purchase_open": {
    taskID: "perf.purchase_open",
    title: "购买链路开环压测",
    subtitle: "固定到达率，观察削峰能力",
    description: "默认临时数据模式：自动创建临时商品/活动/用户，结束自动清理。",
    recommended: true,
    fields: [
      { flag: "rate", label: "每秒请求数 (RPS)", type: "number", min: 1, max: 5000, step: 10, helper: "固定到达率。" },
      { flag: "open-duration", label: "持续时间", type: "duration", placeholder: "30s", helper: "例如 30s / 2m。" },
      { flag: "concurrency", label: "并发上限", type: "number", min: 1, max: 5000, step: 10 },
      {
        flag: "temp-users",
        label: "测试用户数（可选）",
        type: "number",
        min: 1,
        max: 500,
        step: 1,
        required: false,
        placeholder: "留空则按并发自动推导",
        helper: "仅购买类场景生效。",
      },
      { flag: "timeout", label: "请求超时", type: "duration", placeholder: "7s", helper: "建议 3s~10s。" },
      {
        flag: "output",
        label: "结果格式",
        type: "select",
        options: [
          { value: "json", label: "JSON（推荐）" },
          { value: "text", label: "TEXT" },
        ],
      },
    ],
  },
  "perf.purchase_stress": {
    taskID: "perf.purchase_stress",
    title: "购买链路闭环压测",
    subtitle: "固定总请求，观察全程成功率",
    description: "默认临时数据模式：自动创建临时商品/活动/用户，结束自动清理。",
    fields: [
      { flag: "concurrency", label: "并发数", type: "number", min: 1, max: 5000, step: 10 },
      {
        flag: "temp-users",
        label: "测试用户数（可选）",
        type: "number",
        min: 1,
        max: 500,
        step: 1,
        required: false,
        placeholder: "留空则按并发自动推导",
        helper: "仅购买类场景生效。",
      },
      { flag: "requests", label: "总请求数", type: "number", min: 1, max: 1000000, step: 100 },
      { flag: "timeout", label: "请求超时", type: "duration", placeholder: "7s" },
      {
        flag: "output",
        label: "结果格式",
        type: "select",
        options: [
          { value: "json", label: "JSON（推荐）" },
          { value: "text", label: "TEXT" },
        ],
      },
    ],
  },
  "perf.track_open": {
    taskID: "perf.track_open",
    title: "埋点链路开环压测",
    subtitle: "固定到达率，观察事件上报吞吐",
    description: "推荐用于验证点击流采集和队列处理。",
    fields: [
      { flag: "rate", label: "每秒请求数 (RPS)", type: "number", min: 1, max: 20000, step: 50 },
      { flag: "open-duration", label: "持续时间", type: "duration", placeholder: "30s" },
      { flag: "concurrency", label: "并发上限", type: "number", min: 1, max: 8000, step: 10 },
      { flag: "timeout", label: "请求超时", type: "duration", placeholder: "4s" },
      {
        flag: "event-type",
        label: "埋点事件类型",
        type: "select",
        options: [
          { value: "pv", label: "PV" },
          { value: "click", label: "Click" },
          { value: "purchase_attempt", label: "Purchase Attempt" },
        ],
      },
      {
        flag: "output",
        label: "结果格式",
        type: "select",
        options: [
          { value: "json", label: "JSON（推荐）" },
          { value: "text", label: "TEXT" },
        ],
      },
    ],
  },
  "perf.track_stress": {
    taskID: "perf.track_stress",
    title: "埋点链路闭环压测",
    subtitle: "固定总请求，验证事件上报稳定性",
    description: "适合回归阶段快速验证埋点成功率。",
    fields: [
      { flag: "concurrency", label: "并发数", type: "number", min: 1, max: 8000, step: 10 },
      { flag: "requests", label: "总请求数", type: "number", min: 1, max: 1000000, step: 100 },
      { flag: "timeout", label: "请求超时", type: "duration", placeholder: "4s" },
      {
        flag: "event-type",
        label: "埋点事件类型",
        type: "select",
        options: [
          { value: "pv", label: "PV" },
          { value: "click", label: "Click" },
          { value: "purchase_attempt", label: "Purchase Attempt" },
        ],
      },
      {
        flag: "output",
        label: "结果格式",
        type: "select",
        options: [
          { value: "json", label: "JSON（推荐）" },
          { value: "text", label: "TEXT" },
        ],
      },
    ],
  },
  "perf.idempotency": {
    taskID: "perf.idempotency",
    title: "幂等一致性压测",
    subtitle: "重复请求同一键，验证唯一成功约束",
    description: "用于验证幂等保护和重复扣减防护。",
    fields: [
      { flag: "concurrency", label: "并发数", type: "number", min: 1, max: 2000, step: 10 },
      { flag: "requests", label: "总请求数", type: "number", min: 1, max: 100000, step: 100 },
      { flag: "expect-max-success", label: "成功上限断言", type: "number", min: 1, max: 10, step: 1 },
      {
        flag: "output",
        label: "结果格式",
        type: "select",
        options: [
          { value: "json", label: "JSON（推荐）" },
          { value: "text", label: "TEXT" },
        ],
      },
    ],
  },
}

// getRealtimeTestPreset 返回任务对应的前端测试模板。
export function getRealtimeTestPreset(taskID: string): RealtimeTestPreset | null {
  return realtimeTestPresets[taskID] ?? null
}

// getRealtimeTaskDisplayName 返回任务展示标题。
export function getRealtimeTaskDisplayName(task: TaskDef): string {
  const presetTitle = getRealtimeTestPreset(task.id)?.title
  if (presetTitle) {
    return presetTitle
  }
  if (task.name) {
    return task.name
  }
  return task.id
}

// sortRealtimeTestTasks 将“推荐场景”排在前面。
export function sortRealtimeTestTasks(tasks: TaskDef[]): TaskDef[] {
  const copy = [...tasks]
  copy.sort((left, right) => {
    const leftPreset = getRealtimeTestPreset(left.id)
    const rightPreset = getRealtimeTestPreset(right.id)
    if (leftPreset?.recommended && !rightPreset?.recommended) {
      return -1
    }
    if (!leftPreset?.recommended && rightPreset?.recommended) {
      return 1
    }
    return (leftPreset?.title ?? left.name ?? left.id).localeCompare(rightPreset?.title ?? right.name ?? right.id, "zh-CN")
  })
  return copy
}
