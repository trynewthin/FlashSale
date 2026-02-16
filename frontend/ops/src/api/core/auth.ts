const OPS_KEY_STORAGE = "flashsale_ops_key"

// getOpsAccessKey 读取本地保存的 ops 访问密钥。
export function getOpsAccessKey(): string {
  return (localStorage.getItem(OPS_KEY_STORAGE) || "").trim()
}

// setOpsAccessKey 保存 ops 访问密钥。
export function setOpsAccessKey(value: string): void {
  const trimmed = (value || "").trim()
  if (!trimmed) {
    localStorage.removeItem(OPS_KEY_STORAGE)
    return
  }
  localStorage.setItem(OPS_KEY_STORAGE, trimmed)
}

// clearOpsAccessKey 清空 ops 访问密钥。
export function clearOpsAccessKey(): void {
  localStorage.removeItem(OPS_KEY_STORAGE)
}
