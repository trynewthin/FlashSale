// 商品浏览 Hooks：承接公开商品列表与详情查询。
import { useQuery } from "@tanstack/react-query"

import { userQueryKeys } from "@/hooks/query-keys"
import { userProductApi, type ListProductsQuery } from "@/api/modules/product"
import { type Int64Like } from "@/api/core/types"

export function useProductsQuery(params?: ListProductsQuery) {
  return useQuery({
    queryKey: userQueryKeys.productsList(params),
    queryFn: () => userProductApi.listProducts(params),
    retry: 1,
  })
}

export function useProductDetailQuery(productId: Int64Like | undefined) {
  return useQuery({
    queryKey: userQueryKeys.productDetail(productId ?? "0"),
    queryFn: () => userProductApi.getProduct(productId ?? "0"),
    enabled: productId !== undefined && productId !== null,
    retry: 1,
  })
}

