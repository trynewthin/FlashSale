import { userApiClient } from "@/api/core/http"
import { type Int64 } from "@/api/core/types"

export interface RegisterReq {
  phone: string
  password: string
  nickname?: string
}

export interface LoginReq {
  phone: string
  password: string
}

export interface UpdateNicknameReq {
  nickname: string
}

export interface AuthResp {
  user_id: Int64
  phone: string
  nickname: string
  access_token: string
  expires_in_sec: number
}

export interface ProfileResp {
  user_id: Int64
  phone: string
  nickname: string
}

export interface DeleteUserResp {
  user_id: Int64
}

export const userAuthApi = {
  register(req: RegisterReq): Promise<AuthResp> {
    return userApiClient.post<AuthResp, RegisterReq>("/user/register", req)
  },

  login(req: LoginReq): Promise<AuthResp> {
    return userApiClient.post<AuthResp, LoginReq>("/user/login", req)
  },

  getProfile(): Promise<ProfileResp> {
    return userApiClient.get<ProfileResp>("/user/profile")
  },

  updateNickname(req: UpdateNicknameReq): Promise<ProfileResp> {
    return userApiClient.patch<ProfileResp, UpdateNicknameReq>("/user/nickname", req)
  },

  deleteUser(): Promise<DeleteUserResp> {
    return userApiClient.delete<DeleteUserResp>("/user")
  },
}
