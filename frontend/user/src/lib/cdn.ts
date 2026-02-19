/**
 * 将数据库中的相对路径（如 /assets/products/xxx.svg）
 * 拼接为完整的 CDN 可访问 URL。
 *
 * - 开发环境：VITE_CDN_BASE_URL=http://localhost:19000
 * - 生产环境：VITE_CDN_BASE_URL=https://cdn.example.com
 */

const CDN_BASE_URL = (import.meta.env.VITE_CDN_BASE_URL ?? "").replace(/\/+$/, "")

export function cdnUrl(path: string | undefined | null): string {
    if (!path) return ""
    return CDN_BASE_URL + path
}
