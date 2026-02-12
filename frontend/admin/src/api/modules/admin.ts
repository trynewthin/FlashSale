import { type Int64, type Int64Like, type PaginatedItems, type PaginationParams } from "@/api/core/types"
import { adminApiClient } from "@/api/core/http"
import { toInt64String } from "@/api/core/int64"
import { type AdminView } from "@/api/modules/auth"

export interface CreateAdminReq {
  username: string
  display_name: string
  password: string
  data_scope: "all" | "self"
  status: number
}

export interface UpdateAdminReq {
  display_name: string
  data_scope: "all" | "self"
}

export interface SetAdminStatusReq {
  status: number
}

export interface ResetAdminPasswordReq {
  new_password: string
}

export interface BindAdminRolesReq {
  role_ids: Int64Like[]
}

export interface ListAdminsQuery extends PaginationParams {
  keyword?: string
  status?: number
}

export interface RoleView {
  role_id: Int64
  role_code: string
  role_name: string
  status: number
  is_system: boolean
  domains: string[]
  created_at_unix: number
  updated_at_unix: number
}

export interface CreateRoleReq {
  role_code: string
  role_name: string
  status: number
}

export interface UpdateRoleReq {
  role_name: string
  status: number
}

export interface SetRoleDomainsReq {
  domains: string[]
}

export interface ListRolesQuery extends PaginationParams {
  keyword?: string
  status?: number
}

export interface AdminAuditLogView {
  log_id: Int64
  admin_id: Int64
  action: string
  target_type: string
  target_id: Int64
  result: string
  request_id: string
  ip: string
  user_agent: string
  detail_json: string
  created_at_unix: number
}

export interface ListAuditLogsQuery extends PaginationParams {
  admin_id?: Int64Like
  action?: string
  target_type?: string
  target_id?: Int64Like
}

export const adminManagementApi = {
  createAdmin(req: CreateAdminReq): Promise<{ admin: AdminView }> {
    return adminApiClient.post<{ admin: AdminView }, CreateAdminReq>("/admins", req)
  },

  updateAdmin(adminId: Int64Like, req: UpdateAdminReq): Promise<{ admin: AdminView }> {
    return adminApiClient.patch<{ admin: AdminView }, UpdateAdminReq>(`/admins/${adminId}`, req)
  },

  setAdminStatus(adminId: Int64Like, req: SetAdminStatusReq): Promise<{ success: boolean }> {
    return adminApiClient.post<{ success: boolean }, SetAdminStatusReq>(`/admins/${adminId}/status`, req)
  },

  resetAdminPassword(adminId: Int64Like, req: ResetAdminPasswordReq): Promise<{ success: boolean }> {
    return adminApiClient.post<{ success: boolean }, ResetAdminPasswordReq>(
      `/admins/${adminId}/reset-password`,
      req
    )
  },

  deleteAdmin(adminId: Int64Like): Promise<{ success: boolean }> {
    return adminApiClient.delete<{ success: boolean }>(`/admins/${adminId}`)
  },

  getAdmin(adminId: Int64Like): Promise<{ admin: AdminView }> {
    return adminApiClient.get<{ admin: AdminView }>(`/admins/${adminId}`)
  },

  listAdmins(query?: ListAdminsQuery): Promise<PaginatedItems<AdminView>> {
    return adminApiClient.get<PaginatedItems<AdminView>>("/admins", {
      params: query,
    })
  },

  bindAdminRoles(adminId: Int64Like, req: BindAdminRolesReq): Promise<{ success: boolean }> {
    return adminApiClient.post<{ success: boolean }, { role_ids: string[] }>(`/admins/${adminId}/roles`, {
      role_ids: req.role_ids.map((roleID) => toInt64String(roleID, "role_ids")),
    })
  },

  createRole(req: CreateRoleReq): Promise<{ role: RoleView }> {
    return adminApiClient.post<{ role: RoleView }, CreateRoleReq>("/roles", req)
  },

  updateRole(roleId: Int64Like, req: UpdateRoleReq): Promise<{ role: RoleView }> {
    return adminApiClient.patch<{ role: RoleView }, UpdateRoleReq>(`/roles/${roleId}`, req)
  },

  deleteRole(roleId: Int64Like): Promise<{ success: boolean }> {
    return adminApiClient.delete<{ success: boolean }>(`/roles/${roleId}`)
  },

  getRole(roleId: Int64Like): Promise<{ role: RoleView }> {
    return adminApiClient.get<{ role: RoleView }>(`/roles/${roleId}`)
  },

  listRoles(query?: ListRolesQuery): Promise<PaginatedItems<RoleView>> {
    return adminApiClient.get<PaginatedItems<RoleView>>("/roles", {
      params: query,
    })
  },

  setRoleDomains(roleId: Int64Like, req: SetRoleDomainsReq): Promise<{ success: boolean }> {
    return adminApiClient.post<{ success: boolean }, SetRoleDomainsReq>(`/roles/${roleId}/domains`, req)
  },

  listAuditLogs(query?: ListAuditLogsQuery): Promise<PaginatedItems<AdminAuditLogView>> {
    return adminApiClient.get<PaginatedItems<AdminAuditLogView>>("/audit-logs", {
      params: query,
    })
  },
}
