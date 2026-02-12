const USER_ACCESS_TOKEN_KEY = "flashsale:user:access_token"

function getStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null
  }
  return window.localStorage
}

export const userTokenStore = {
  getAccessToken(): string {
    const storage = getStorage()
    if (!storage) {
      return ""
    }
    return storage.getItem(USER_ACCESS_TOKEN_KEY) ?? ""
  },

  setAccessToken(token: string): void {
    const storage = getStorage()
    if (!storage) {
      return
    }
    if (token.trim() === "") {
      storage.removeItem(USER_ACCESS_TOKEN_KEY)
      return
    }
    storage.setItem(USER_ACCESS_TOKEN_KEY, token)
  },

  clear(): void {
    const storage = getStorage()
    storage?.removeItem(USER_ACCESS_TOKEN_KEY)
  },
}

