import { adminApiClient } from "@/api/core/http"
import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"

export interface ManagedUserView {
  user_id: Int64
  phone: string
  nickname: string
  status: number
  last_login_at_unix: number
  last_login_ip: string
  created_at_unix: number
  updated_at_unix: number
}

export interface ListUsersQuery extends PaginationParams {
  keyword?: string
  status?: number
}

export interface UpdateManagedUserNicknameReq {
  nickname: string
}

export const adminUserApi = {
  getUserProfile(userId: Int64Like): Promise<{ user: ManagedUserView }> {
    return adminApiClient.get<{ user: ManagedUserView }>(`/users/${userId}`)
  },

  updateUserNickname(userId: Int64Like, req: UpdateManagedUserNicknameReq): Promise<{ user: ManagedUserView }> {
    return adminApiClient.patch<{ user: ManagedUserView }, UpdateManagedUserNicknameReq>(
      `/users/${userId}/nickname`,
      req
    )
  },

  deleteUser(userId: Int64Like): Promise<{ user_id: Int64 }> {
    return adminApiClient.delete<{ user_id: Int64 }>(`/users/${userId}`)
  },

  listUsers(query?: ListUsersQuery): Promise<PaginatedList<ManagedUserView>> {
    return adminApiClient.get<PaginatedList<ManagedUserView>>("/users", {
      params: query,
    })
  },
}
