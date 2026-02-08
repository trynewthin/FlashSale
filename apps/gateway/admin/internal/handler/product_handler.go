// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
	"flashsale/pkg/base/rpcmeta"
)

// ProductAdminHandler 处理管理员商品请求。
type ProductAdminHandler struct {
	svcCtx *svc.ServiceContext
}

func (h *ProductAdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	var req productUpsertReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	if err := validateProductUpsertReq(&req); err != nil {
		writeFail(w, errorx.FromError(err).Code.HTTPStatus(), err)
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.ProductRPCCli.CreateProduct(rpcCtx, &productpb.CreateProductReq{
		Name:        req.Name,
		MainImage:   req.MainImage,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Stock:       req.Stock,
		Status:      req.Status,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *ProductAdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	productID, ok := handlerx.ParsePathInt64(r, "product_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
		return
	}
	var req productUpsertReq
	if err := decodeJSON(r, &req); err != nil {
		writeFail(w, http.StatusBadRequest, errorx.Wrap(errorx.CodeSysBadRequest, "请求体非法", err))
		return
	}
	if err := validateProductUpsertReq(&req); err != nil {
		writeFail(w, errorx.FromError(err).Code.HTTPStatus(), err)
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.ProductRPCCli.UpdateProduct(rpcCtx, &productpb.UpdateProductReq{
		ProductId:   productID,
		Name:        req.Name,
		MainImage:   req.MainImage,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Stock:       req.Stock,
		Status:      req.Status,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func validateProductUpsertReq(req *productUpsertReq) error {
	if req == nil {
		return errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}

	name := strings.TrimSpace(req.Name)
	if l := utf8.RuneCountInString(name); l < 1 || l > 100 {
		return errorx.New(errorx.CodeSysBadRequest, "商品名称长度需在1到100之间")
	}
	mainImage := strings.TrimSpace(req.MainImage)
	if mainImage == "" {
		return errorx.New(errorx.CodeSysBadRequest, "商品主图不能为空")
	}
	if utf8.RuneCountInString(mainImage) > 512 {
		return errorx.New(errorx.CodeSysBadRequest, "商品主图长度不能超过512")
	}
	description := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(description) > 2000 {
		return errorx.New(errorx.CodeSysBadRequest, "商品描述长度不能超过2000")
	}
	if req.PriceCent <= 0 {
		return errorx.New(errorx.CodeSysBadRequest, "商品价格必须大于0")
	}
	if req.Stock < 0 {
		return errorx.New(errorx.CodeSysBadRequest, "商品库存不能为负数")
	}
	if req.Status != 0 && req.Status != 1 {
		return errorx.New(errorx.CodeProductInvalidStatus, "商品状态非法")
	}

	req.Name = name
	req.MainImage = mainImage
	req.Description = description
	return nil
}

func (h *ProductAdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	productID, ok := handlerx.ParsePathInt64(r, "product_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.ProductRPCCli.DeleteProduct(rpcCtx, &productpb.DeleteProductReq{ProductId: productID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *ProductAdminHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	productID, ok := handlerx.ParsePathInt64(r, "product_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
		return
	}
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.ProductRPCCli.GetProductAdmin(rpcCtx, &productpb.GetProductAdminReq{ProductId: productID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

func (h *ProductAdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	includeDeleted := handlerx.ParseBoolQuery(r, "include_deleted")
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	rpcCtx = rpcmeta.WithAccessToken(rpcCtx, token)
	resp, err := h.svcCtx.ProductRPCCli.ListProductsAdmin(rpcCtx, &productpb.ListProductsAdminReq{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        keyword,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
