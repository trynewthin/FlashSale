import { type Int64, type Int64Like, type PaginatedList, type PaginationParams } from "@/api/core/types"
import { userApiClient } from "@/api/core/http"

export interface PublicProduct {
  product_id: Int64
  sku_code: string
  name: string
  main_image: string
  description: string
  price_cent: number
  in_stock: boolean
}

export interface ListProductsQuery extends PaginationParams {
  keyword?: string
}

export type ListProductsResp = PaginatedList<PublicProduct>

export interface GetProductResp {
  product: PublicProduct
}

export const userProductApi = {
  listProducts(query?: ListProductsQuery): Promise<ListProductsResp> {
    return userApiClient.get<ListProductsResp>("/products", {
      params: query,
    })
  },

  getProduct(productId: Int64Like): Promise<GetProductResp> {
    return userApiClient.get<GetProductResp>(`/products/${productId}`)
  },
}
