/**
 * 格式化 Unix 时间戳为本地时间字符串。
 * 所有 _unix 字段统一用此函数格式化，确保时区一致。
 */
export function formatUnix(unix: number | string | undefined | null): string {
    if (!unix) return "-"
    const ts = typeof unix === "string" ? Number(unix) : unix
    if (!Number.isFinite(ts) || ts <= 0) return "-"
    return new Date(ts * 1000).toLocaleString("zh-CN", {
        timeZone: "Asia/Shanghai",
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
    })
}

/**
 * 格式化金额（分 → 元）。
 */
export function formatCent(cent: number | string | undefined | null): string {
    if (cent === undefined || cent === null) return "¥0.00"
    const n = typeof cent === "string" ? Number(cent) : cent
    return `¥${(n / 100).toFixed(2)}`
}
