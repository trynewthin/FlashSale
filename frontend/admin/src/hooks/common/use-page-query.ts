// 分页参数工具：统一 page/page_size 默认值与更新方式，减少列表页重复状态代码。
import { useMemo, useState } from "react"

export interface PageQuery {
  page: number
  page_size: number
}

export function usePageQuery(initial?: Partial<PageQuery>) {
  const [page, setPage] = useState(initial?.page ?? 1)
  const [pageSize, setPageSize] = useState(initial?.page_size ?? 20)

  const query = useMemo<PageQuery>(() => {
    return {
      page,
      page_size: pageSize,
    }
  }, [page, pageSize])

  const resetPage = () => setPage(1)

  return {
    page,
    pageSize,
    query,
    setPage,
    setPageSize,
    resetPage,
  }
}

