package handler

import (
	"context"
	"net/http"
	"strings"

	"flashsale/apps/gateway/user/internal/svc"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/handlerx"
)

// ProductPublicHandler 处理用户侧公开商品请求。
type ProductPublicHandler struct {
	svcCtx *svc.ServiceContext
}

// ListProducts 返回公开商品列表。
func (h *ProductPublicHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	page, pageSize := handlerx.ParsePagination(r, 1, 20, 100)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))

	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.ProductRPCCli.ListProductsPublic(rpcCtx, &productpb.ListProductsPublicReq{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}

// GetProduct 返回公开商品详情。
func (h *ProductPublicHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svcCtx == nil || h.svcCtx.ProductRPCCli == nil {
		writeFail(w, http.StatusInternalServerError, errorx.New(errorx.CodeSysInternal, "gateway not initialized"))
		return
	}
	productID, ok := handlerx.ParsePathInt64(r, "product_id")
	if !ok {
		writeFail(w, http.StatusBadRequest, errorx.New(errorx.CodeSysBadRequest, "product_id 非法"))
		return
	}
	rpcCtx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	resp, err := h.svcCtx.ProductRPCCli.GetProductPublic(rpcCtx, &productpb.GetProductPublicReq{ProductId: productID})
	if err != nil {
		writeRPCFail(w, err)
		return
	}
	writeOK(w, resp)
}
