// 幂等键工具：支持“同一业务动作复用同一 key”，避免重复提交导致的副作用。
import { useCallback, useRef } from "react"
import { v4 as uuidv4 } from "uuid"

export function useIdempotencyKey() {
  const keyRef = useRef<string>("")

  const renew = useCallback((): string => {
    const next = uuidv4()
    keyRef.current = next
    return next
  }, [])

  const get = useCallback((): string => {
    if (keyRef.current) {
      return keyRef.current
    }
    return renew()
  }, [renew])

  const clear = useCallback((): void => {
    keyRef.current = ""
  }, [])

  return {
    getIdempotencyKey: get,
    renewIdempotencyKey: renew,
    clearIdempotencyKey: clear,
  }
}

