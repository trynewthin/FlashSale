import axios, { AxiosHeaders } from "axios"

import { adminTokenStore } from "@/api/core/token-store"

// Media API 通过 Nginx 网关代理访问（/api/v1/admin/media/ → media-store）。
// JWT 鉴权由 Nginx auth_request 验证，media-store secret 由 Nginx server-side 注入。
// 前端不持有任何 media-store secret。
const ADMIN_API_BASE_URL = import.meta.env.VITE_ADMIN_API_BASE_URL ?? "/api/v1/admin"

const mediaHttp = axios.create({
    baseURL: ADMIN_API_BASE_URL,
    timeout: 30000,
})

// 复用 admin JWT 鉴权
mediaHttp.interceptors.request.use((config) => {
    const token = adminTokenStore.getAccessToken()
    if (token) {
        config.headers = AxiosHeaders.from(config.headers)
        config.headers.set("Authorization", `Bearer ${token}`)
    }
    return config
})

export interface MediaFileInfo {
    filename: string
    category: string
    url: string
    size: number
    created_at: string
}

export interface MediaCategoryInfo {
    name: string
    count: number
}

export const mediaApi = {
    async upload(file: File, category = "products"): Promise<MediaFileInfo> {
        const formData = new FormData()
        formData.append("file", file)
        const resp = await mediaHttp.post<MediaFileInfo>(
            `/media/files/upload?category=${encodeURIComponent(category)}`,
            formData,
            { headers: { "Content-Type": "multipart/form-data" } }
        )
        return resp.data
    },

    async list(category?: string): Promise<{ files: MediaFileInfo[]; total: number }> {
        const params = category ? { category } : {}
        const resp = await mediaHttp.get<{ files: MediaFileInfo[]; total: number }>("/media/files", { params })
        return resp.data
    },

    async deleteFile(category: string, filename: string): Promise<void> {
        await mediaHttp.delete(`/media/files/${encodeURIComponent(category)}/${encodeURIComponent(filename)}`)
    },

    async categories(): Promise<{ categories: MediaCategoryInfo[] }> {
        const resp = await mediaHttp.get<{ categories: MediaCategoryInfo[] }>("/media/categories")
        return resp.data
    },
}
