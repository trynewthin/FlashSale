const ADMIN_ACCESS_TOKEN_KEY = "flashsale:admin:access_token"
const ADMIN_REFRESH_TOKEN_KEY = "flashsale:admin:refresh_token"

function getStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null
  }
  return window.localStorage
}

function setOrRemove(key: string, value: string): void {
  const storage = getStorage()
  if (!storage) {
    return
  }
  if (value.trim() === "") {
    storage.removeItem(key)
    return
  }
  storage.setItem(key, value)
}

export const adminTokenStore = {
  getAccessToken(): string {
    const storage = getStorage()
    if (!storage) {
      return ""
    }
    return storage.getItem(ADMIN_ACCESS_TOKEN_KEY) ?? ""
  },

  setAccessToken(token: string): void {
    setOrRemove(ADMIN_ACCESS_TOKEN_KEY, token)
  },

  getRefreshToken(): string {
    const storage = getStorage()
    if (!storage) {
      return ""
    }
    return storage.getItem(ADMIN_REFRESH_TOKEN_KEY) ?? ""
  },

  setRefreshToken(token: string): void {
    setOrRemove(ADMIN_REFRESH_TOKEN_KEY, token)
  },

  clear(): void {
    const storage = getStorage()
    storage?.removeItem(ADMIN_ACCESS_TOKEN_KEY)
    storage?.removeItem(ADMIN_REFRESH_TOKEN_KEY)
  },
}

