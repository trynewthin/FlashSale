import { type Int64 } from "@/api/core/types"
import { type ListAdminsQuery, type ListAuditLogsQuery, type ListRolesQuery } from "@/api/modules/admin"
import { type ListAdminOrdersQuery } from "@/api/modules/order"
import { type ListAdminProductsQuery } from "@/api/modules/product"
import {
  type GetActivityTrafficQuery,
  type ListActivitiesQuery,
  type ListActivityOrdersQuery,
} from "@/api/modules/seckill"

export const adminQueryKeys = {
  session: () => ["admin", "session"] as const,
  profile: (adminId?: Int64) => ["admin", adminId ?? "anonymous", "profile"] as const,
  admins: (operatorAdminId?: Int64, params?: ListAdminsQuery) =>
    ["admin", operatorAdminId ?? "anonymous", "admins", params ?? {}] as const,
  adminDetail: (operatorAdminId: Int64 | undefined, targetAdminId: string | number) =>
    ["admin", operatorAdminId ?? "anonymous", "detail", String(targetAdminId)] as const,
  roles: (operatorAdminId?: Int64, params?: ListRolesQuery) =>
    ["admin", operatorAdminId ?? "anonymous", "roles", params ?? {}] as const,
  roleDetail: (operatorAdminId: Int64 | undefined, roleId: string | number) =>
    ["admin", operatorAdminId ?? "anonymous", "role", String(roleId)] as const,
  auditLogs: (operatorAdminId?: Int64, params?: ListAuditLogsQuery) =>
    ["admin", operatorAdminId ?? "anonymous", "audit", params ?? {}] as const,
  managedUserProfile: (operatorAdminId: Int64 | undefined, userId: string | number) =>
    ["mgmt", operatorAdminId ?? "anonymous", "users", String(userId)] as const,
  productsList: (operatorAdminId?: Int64, params?: ListAdminProductsQuery) =>
    ["mgmt", operatorAdminId ?? "anonymous", "products", "list", params ?? {}] as const,
  productDetail: (operatorAdminId: Int64 | undefined, productId: string | number) =>
    ["mgmt", operatorAdminId ?? "anonymous", "products", "detail", String(productId)] as const,
  ordersList: (operatorAdminId?: Int64, params?: ListAdminOrdersQuery) =>
    ["mgmt", operatorAdminId ?? "anonymous", "orders", "list", params ?? {}] as const,
  orderDetail: (operatorAdminId: Int64 | undefined, orderId: string | number) =>
    ["mgmt", operatorAdminId ?? "anonymous", "orders", "detail", String(orderId)] as const,
  seckillActivities: (operatorAdminId?: Int64, params?: ListActivitiesQuery) =>
    ["mgmt", operatorAdminId ?? "anonymous", "seckill", "activities", params ?? {}] as const,
  seckillActivityDetail: (operatorAdminId: Int64 | undefined, activityId: string | number) =>
    ["mgmt", operatorAdminId ?? "anonymous", "seckill", "activity", String(activityId)] as const,
  seckillTraffic: (
    operatorAdminId: Int64 | undefined,
    activityId: string | number,
    params?: GetActivityTrafficQuery
  ) => ["mgmt", operatorAdminId ?? "anonymous", "seckill", "traffic", String(activityId), params ?? {}] as const,
  seckillOrders: (
    operatorAdminId: Int64 | undefined,
    activityId: string | number,
    params?: ListActivityOrdersQuery
  ) => ["mgmt", operatorAdminId ?? "anonymous", "seckill", "orders", String(activityId), params ?? {}] as const,
  opsPing: (operatorAdminId?: Int64) => ["ops", operatorAdminId ?? "anonymous", "ping"] as const,
}

