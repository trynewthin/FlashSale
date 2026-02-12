import { adminApiClient } from "@/api/core/http"
import { type Int64 } from "@/api/core/types"

export interface OpsPingResp {
  admin_id: Int64
  domains: string[]
  data_scope: string
  message: string
}

export const adminOpsApi = {
  ping(): Promise<OpsPingResp> {
    return adminApiClient.get<OpsPingResp>("/ping")
  },
}
