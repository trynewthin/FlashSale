// 商品管理 Hooks：封装管理端商品列表、详情与写操作。
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { type Int64Like } from "@/api/core/types"
import { adminProductApi, type ListAdminProductsQuery, type ProductUpsertReq } from "@/api/modules/product"
import { adminQueryKeys } from "@/hooks/query-keys"
import { useAdminAuthStore } from "@/stores/auth-store"

export function useAdminProductListQuery(params?: ListAdminProductsQuery) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.productsList(operatorAdminId, params),
    queryFn: () => adminProductApi.listProducts(params),
    retry: 1,
  })
}

export function useAdminProductDetailQuery(productId: Int64Like | undefined) {
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useQuery({
    queryKey: adminQueryKeys.productDetail(operatorAdminId, productId ?? "0"),
    queryFn: () => adminProductApi.getProduct(productId ?? "0"),
    enabled: productId !== undefined && productId !== null,
    retry: 1,
  })
}

export function useCreateProductMutation() {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ProductUpsertReq) => adminProductApi.createProduct(req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.productsList(operatorAdminId),
      })
    },
  })
}

export function useUpdateProductMutation(productId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: (req: ProductUpsertReq) => adminProductApi.updateProduct(productId, req),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.productDetail(operatorAdminId, productId),
      })
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.productsList(operatorAdminId),
      })
    },
  })
}

export function useDeleteProductMutation(productId: Int64Like) {
  const queryClient = useQueryClient()
  const operatorAdminId = useAdminAuthStore((state) => state.profile?.admin_id)
  return useMutation({
    mutationFn: () => adminProductApi.deleteProduct(productId),
    retry: 0,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminQueryKeys.productsList(operatorAdminId),
      })
      queryClient.removeQueries({
        queryKey: adminQueryKeys.productDetail(operatorAdminId, productId),
      })
    },
  })
}

