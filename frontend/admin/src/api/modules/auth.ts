import { adminApiClient } from "@/api/core/http"
import { adminTokenStore } from "@/api/core/token-store"
import { type Int64 } from "@/api/core/types"

export interface AdminView {
  admin_id: Int64
  username: string
  display_name: string
  status: number
  data_scope: string
  is_super_admin: boolean
  last_login_at_unix: number
  last_login_ip: string
  role_ids: Int64[]
  domains: string[]
  created_at_unix: number
  updated_at_unix: number
}

export interface AdminAuthResp {
  admin: AdminView
  access_token: string
  access_expires_in_sec: number
  refresh_token: string
  refresh_expires_in_sec: number
}

export interface AdminLoginReq {
  username: string
  password: string
}

export interface AdminRefreshReq {
  refresh_token: string
}

export interface AdminLogoutReq {
  refresh_token: string
}

export interface ChangeMyPasswordReq {
  old_password: string
  new_password: string
}

export interface ChangeMyPasswordResp {
  success: boolean
}

export const adminAuthApi = {
  login(req: AdminLoginReq): Promise<AdminAuthResp> {
    return adminApiClient.post<AdminAuthResp, AdminLoginReq>("/auth/login", req)
  },

  refresh(req: AdminRefreshReq): Promise<AdminAuthResp> {
    return adminApiClient.post<AdminAuthResp, AdminRefreshReq>("/auth/refresh", req)
  },

  logout(req: AdminLogoutReq): Promise<{ success: boolean }> {
    return adminApiClient.post<{ success: boolean }, AdminLogoutReq>("/auth/logout", req)
  },

  getMyProfile(): Promise<{ admin: AdminView }> {
    return adminApiClient.get<{ admin: AdminView }>("/me")
  },

  changeMyPassword(req: ChangeMyPasswordReq): Promise<ChangeMyPasswordResp> {
    return adminApiClient.post<ChangeMyPasswordResp, ChangeMyPasswordReq>("/me/password", req)
  },
}

export function persistAdminTokens(auth: AdminAuthResp): void {
  adminTokenStore.setAccessToken(auth.access_token)
  adminTokenStore.setRefreshToken(auth.refresh_token)
}
