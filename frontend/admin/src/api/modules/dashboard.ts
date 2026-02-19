import { adminApiClient } from "@/api/core/http"

export interface DashboardStats {
    users: { total: number }
    products: { total: number; on_sale: number }
    orders: {
        total: number
        pending_payment: number
        pending_review: number
        pending_ship: number
        shipped: number
        closed: number
    }
    seckill: { total: number; active: number; upcoming: number }
}

export const dashboardApi = {
    getStats(): Promise<DashboardStats> {
        return adminApiClient.get<DashboardStats>("/dashboard/stats")
    },
}
