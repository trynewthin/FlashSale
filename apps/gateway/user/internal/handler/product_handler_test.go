package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"flashsale/apps/gateway/user/internal/svc"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/apps/product/rpc/productrpc"
	"google.golang.org/grpc"
)

type fakeProductRPC struct{}

func (f *fakeProductRPC) CreateProduct(context.Context, *productrpc.CreateProductReq, ...grpc.CallOption) (*productrpc.CreateProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) UpdateProduct(context.Context, *productrpc.UpdateProductReq, ...grpc.CallOption) (*productrpc.UpdateProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) DeleteProduct(context.Context, *productrpc.DeleteProductReq, ...grpc.CallOption) (*productrpc.DeleteProductResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) GetProductAdmin(context.Context, *productrpc.GetProductAdminReq, ...grpc.CallOption) (*productrpc.GetProductAdminResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) ListProductsAdmin(context.Context, *productrpc.ListProductsAdminReq, ...grpc.CallOption) (*productrpc.ListProductsAdminResp, error) {
	return nil, nil
}
func (f *fakeProductRPC) GetProductPublic(context.Context, *productrpc.GetProductPublicReq, ...grpc.CallOption) (*productrpc.GetProductPublicResp, error) {
	return &productpb.GetProductPublicResp{Product: &productpb.PublicProduct{ProductId: 1001, Name: "可乐", PriceCent: 399, InStock: true}}, nil
}
func (f *fakeProductRPC) ListProductsPublic(context.Context, *productrpc.ListProductsPublicReq, ...grpc.CallOption) (*productrpc.ListProductsPublicResp, error) {
	return &productpb.ListProductsPublicResp{List: []*productpb.PublicProduct{{ProductId: 1001, Name: "可乐", PriceCent: 399, InStock: true}}, Total: 1, Page: 1, PageSize: 20}, nil
}

func TestPublicProductRoutesAnonymousAccess(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, &svc.ServiceContext{ProductRPCCli: &fakeProductRPC{}})

	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/products?page=1&page_size=20", nil)
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("list status mismatch: got %d want %d", recList.Code, http.StatusOK)
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/products/1001", nil)
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get status mismatch: got %d want %d", recGet.Code, http.StatusOK)
	}
}
