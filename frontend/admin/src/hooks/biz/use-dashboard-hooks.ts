import { useQuery } from "@tanstack/react-query"

import { dashboardApi } from "@/api/modules/dashboard"

export function useDashboardStatsQuery() {
    return useQuery({
        queryKey: ["admin", "dashboard", "stats"],
        queryFn: () => dashboardApi.getStats(),
        staleTime: 30_000,
        retry: 1,
    })
}
