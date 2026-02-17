import { useEffect, useRef } from "react"

interface EventSourceHandlers {
  onMessage?: (event: MessageEvent) => void
  onEvent?: (eventType: string, payload: unknown) => void
  onOpen?: () => void
  onError?: () => void
}

interface EventSourceState {
  enabled: boolean
  url: string
  handlers?: EventSourceHandlers
}

function parsePayload(raw: string): unknown {
  try {
    return JSON.parse(raw)
  } catch {
    return raw
  }
}

// useEventSource 管理单连接 SSE 生命周期。
export function useEventSource({ enabled, url, handlers }: EventSourceState) {
  const handlersRef = useRef<EventSourceHandlers | undefined>(handlers)

  useEffect(() => {
    handlersRef.current = handlers
  }, [handlers])

  useEffect(() => {
    if (!enabled || !url) {
      return
    }

    const source = new EventSource(url)

    source.onopen = () => {
      handlersRef.current?.onOpen?.()
    }

    source.onmessage = (event) => {
      handlersRef.current?.onMessage?.(event)
    }

    const subscribeEvent = (eventType: string) => {
      source.addEventListener(eventType, (event) => {
        const data = (event as MessageEvent).data
        handlersRef.current?.onEvent?.(eventType, parsePayload(String(data || "")))
      })
    }

    subscribeEvent("log")
    subscribeEvent("snapshot")
    subscribeEvent("done")
    subscribeEvent("ping")
    subscribeEvent("meta")
    subscribeEvent("error")

    source.onerror = () => {
      handlersRef.current?.onError?.()
    }

    return () => {
      source.close()
    }
  }, [enabled, url])
}
