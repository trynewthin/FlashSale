import type { JobDetail } from "@/api/types"

// parseTaskArgs 将输入参数按空白分割为参数数组。
export function parseTaskArgs(raw: string): string[] {
  const trimmed = raw.trim()
  if (!trimmed) {
    return []
  }
  return trimmed.split(/\s+/)
}

// formatTaskArgs 将参数数组转换为可编辑文本。
export function formatTaskArgs(args: string[] | undefined): string {
  if (!args || args.length === 0) {
    return ""
  }
  return args.join(" ")
}

function isFlagToken(token: string): boolean {
  return token.startsWith("-") || token.startsWith("--")
}

function normalizeFlag(flag: string): string {
  return flag.replace(/^-+/, "")
}

// getFlagArg 从参数数组中读取指定 flag 的值。
export function getFlagArg(args: string[] | undefined, flag: string): string {
  if (!args || args.length === 0) {
    return ""
  }
  const target = normalizeFlag(flag)
  for (let index = 0; index < args.length; index += 1) {
    const token = args[index]
    if (!isFlagToken(token)) {
      continue
    }
    const normalized = normalizeFlag(token)
    if (normalized === target) {
      const next = args[index + 1]
      if (next && !isFlagToken(next)) {
        return next
      }
      return ""
    }
    if (normalized.startsWith(`${target}=`)) {
      return normalized.slice(target.length + 1)
    }
  }
  return ""
}

// removeFlagArg 从参数数组中移除指定 flag（含值）。
export function removeFlagArg(args: string[], flag: string): string[] {
  const target = normalizeFlag(flag)
  const nextArgs: string[] = []
  for (let index = 0; index < args.length; index += 1) {
    const token = args[index]
    if (!isFlagToken(token)) {
      nextArgs.push(token)
      continue
    }
    const normalized = normalizeFlag(token)
    if (normalized === target) {
      const maybeValue = args[index + 1]
      if (maybeValue && !isFlagToken(maybeValue)) {
        index += 1
      }
      continue
    }
    if (normalized.startsWith(`${target}=`)) {
      continue
    }
    nextArgs.push(token)
  }
  return nextArgs
}

// setFlagArg 设置指定 flag 的值，不存在时自动追加。
export function setFlagArg(args: string[], flag: string, value: string): string[] {
  const target = normalizeFlag(flag)
  if (!value.trim()) {
    return removeFlagArg(args, flag)
  }
  const nextArgs = [...args]
  for (let index = 0; index < nextArgs.length; index += 1) {
    const token = nextArgs[index]
    if (!isFlagToken(token)) {
      continue
    }
    const normalized = normalizeFlag(token)
    if (normalized === target) {
      if (index + 1 < nextArgs.length && !isFlagToken(nextArgs[index + 1])) {
        nextArgs[index + 1] = value
      } else {
        nextArgs.splice(index + 1, 0, value)
      }
      return nextArgs
    }
    if (normalized.startsWith(`${target}=`)) {
      const prefix = token.startsWith("--") ? "--" : "-"
      nextArgs[index] = `${prefix}${target}=${value}`
      return nextArgs
    }
  }
  nextArgs.push(`-${target}`, value)
  return nextArgs
}

// jobDurationSeconds 计算任务已运行秒数。
export function jobDurationSeconds(job: JobDetail | null): number {
  if (!job) {
    return 0
  }
  const start = Date.parse(job.started_at || job.created_at || "")
  if (Number.isNaN(start)) {
    return 0
  }
  const end = job.finished_at ? Date.parse(job.finished_at) : Date.now()
  if (Number.isNaN(end) || end < start) {
    return 0
  }
  return Math.floor((end - start) / 1000)
}
