import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { adminApiClient } from "@/api/core/http"

export interface AdminProduct {
  product_id: Int64
  sku_code: string
  name: string
  main_image: string
  description: string
  price_cent: number
  stock: number
  status: number
  created_at_unix: number
  updated_at_unix: number
}

export interface ProductUpsertReq {
  name: string
  main_image: string
  description: string
  price_cent: number
  stock: number
  status: number
}

export interface ListAdminProductsQuery extends PaginationParams {
  keyword?: string
  include_deleted?: boolean
}

export const adminProductApi = {
  createProduct(req: ProductUpsertReq): Promise<{ product: AdminProduct }> {
    return adminApiClient.post<{ product: AdminProduct }, ProductUpsertReq>("/products", req)
  },

  updateProduct(productId: Int64Like, req: ProductUpsertReq): Promise<{ product: AdminProduct }> {
    return adminApiClient.patch<{ product: AdminProduct }, ProductUpsertReq>(`/products/${productId}`, req)
  },

  deleteProduct(productId: Int64Like): Promise<{ product_id: Int64 }> {
    return adminApiClient.delete<{ product_id: Int64 }>(`/products/${productId}`)
  },

  getProduct(productId: Int64Like): Promise<{ product: AdminProduct }> {
    return adminApiClient.get<{ product: AdminProduct }>(`/products/${productId}`)
  },

  listProducts(query?: ListAdminProductsQuery): Promise<PaginatedList<AdminProduct>> {
    return adminApiClient.get<PaginatedList<AdminProduct>>("/products", {
      params: query,
    })
  },
}
