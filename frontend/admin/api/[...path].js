/**
 * Vercel API 代理：
 * - 将 /api/* 请求转发到 BACKEND_ORIGIN
 * - 避免前端直连后端产生跨域问题
 */
module.exports = async function handler(req, res) {
  const backendOrigin = String(process.env.BACKEND_ORIGIN || "").replace(/\/+$/, "")
  if (!backendOrigin) {
    res.status(500).json({ code: "ERR", message: "BACKEND_ORIGIN is required" })
    return
  }

  const rawPath = Array.isArray(req.query.path) ? req.query.path.join("/") : String(req.query.path || "")
  const targetUrl = `${backendOrigin}/api/${rawPath}${buildSearch(req)}`

  const headers = { ...req.headers }
  delete headers.host
  delete headers["content-length"]
  delete headers.connection

  const init = {
    method: req.method,
    headers,
  }
  if (!["GET", "HEAD"].includes(req.method || "")) {
    init.body = toRequestBody(req.body)
  }

  try {
    const upstream = await fetch(targetUrl, init)
    res.status(upstream.status)
    upstream.headers.forEach((value, key) => {
      if (key.toLowerCase() === "transfer-encoding") {
        return
      }
      res.setHeader(key, value)
    })
    const buf = Buffer.from(await upstream.arrayBuffer())
    res.send(buf)
  } catch (err) {
    res.status(502).json({
      code: "ERR",
      message: `proxy request failed: ${err instanceof Error ? err.message : "unknown error"}`,
    })
  }
}

function buildSearch(req) {
  const query = { ...req.query }
  delete query.path
  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((item) => params.append(key, String(item)))
      return
    }
    if (value !== undefined) {
      params.append(key, String(value))
    }
  })
  const qs = params.toString()
  return qs ? `?${qs}` : ""
}

function toRequestBody(body) {
  if (body === undefined || body === null) {
    return undefined
  }
  if (Buffer.isBuffer(body) || typeof body === "string") {
    return body
  }
  return JSON.stringify(body)
}

