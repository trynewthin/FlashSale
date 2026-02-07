package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"flashsale/apps/gateway/admin/internal/authz"
	"flashsale/apps/gateway/admin/internal/svc"
	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/apps/product/rpc/productrpc"
	baseauth "flashsale/pkg/base/authx"
	"google.golang.org/grpc"
)

func TestProductRouteForbiddenWithoutProductDomain(t *testing.T) {
	initAdminJWTForProductTest(t)
	fakeRPC := &fakeAdminProductRPC{}

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "5001", time.Hour, []string{"operations"}, "all")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer(), ProductRPCCli: fakeRPC}
	mux := http.NewServeMux()
	RegisterRoutes(mux, ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusForbidden)
	}
	if fakeRPC.listAdminCalls != 0 {
		t.Fatalf("rpc should not be called when forbidden")
	}
}

func TestProductRoutePassDomainMiddleware(t *testing.T) {
	initAdminJWTForProductTest(t)
	fakeRPC := &fakeAdminProductRPC{}

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "5001", time.Hour, []string{"product_management"}, "all")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer(), ProductRPCCli: fakeRPC}
	mux := http.NewServeMux()
	RegisterRoutes(mux, ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusOK)
	}
	if fakeRPC.listAdminCalls != 1 {
		t.Fatalf("expected one list rpc call, got %d", fakeRPC.listAdminCalls)
	}
}

func TestCreateProductLightValidationBlocksRPC(t *testing.T) {
	initAdminJWTForProductTest(t)
	fakeRPC := &fakeAdminProductRPC{}

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "5001", time.Hour, []string{"product_management"}, "all")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}
	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer(), ProductRPCCli: fakeRPC}
	mux := http.NewServeMux()
	RegisterRoutes(mux, ctx)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", strings.NewReader(`{"name":"   ","main_image":"https://img/a.png","description":"ok","price_cent":100,"stock":1,"status":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusBadRequest)
	}
	if fakeRPC.createCalls != 0 {
		t.Fatalf("create rpc should not be called on validation fail")
	}
}

func TestUpdateProductLightValidationBlocksRPC(t *testing.T) {
	initAdminJWTForProductTest(t)
	fakeRPC := &fakeAdminProductRPC{}

	token, err := baseauth.IssueWithClaims(baseauth.TokenTypeAdmin, "5001", time.Hour, []string{"product_management"}, "all")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}
	ctx := &svc.ServiceContext{Authorizer: authz.NewStaticAuthorizer(), ProductRPCCli: fakeRPC}
	mux := http.NewServeMux()
	RegisterRoutes(mux, ctx)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/products/1001", strings.NewReader(`{"name":"商品A","main_image":"https://img/a.png","description":"ok","price_cent":100,"stock":1,"status":2}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d want %d", rec.Code, http.StatusBadRequest)
	}
	if fakeRPC.updateCalls != 0 {
		t.Fatalf("update rpc should not be called on validation fail")
	}
}

func initAdminJWTForProductTest(t *testing.T) {
	t.Helper()
	err := baseauth.Init(baseauth.JWTConfig{
		User: baseauth.JWTDomainConfig{
			Secret:   "admin-handler-product-user-secret",
			Issuer:   "admin-handler-product-user-issuer",
			Audience: "admin-handler-product-user-aud",
			TTL:      time.Hour,
		},
		Admin: baseauth.JWTDomainConfig{
			Secret:   "admin-handler-product-admin-secret",
			Issuer:   "admin-handler-product-admin-issuer",
			Audience: "admin-handler-product-admin-aud",
			TTL:      time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("init auth failed: %v", err)
	}
}

type fakeAdminProductRPC struct {
	createCalls    int
	updateCalls    int
	listAdminCalls int
}

func (f *fakeAdminProductRPC) CreateProduct(context.Context, *productrpc.CreateProductReq, ...grpc.CallOption) (*productrpc.CreateProductResp, error) {
	f.createCalls++
	return &productpb.CreateProductResp{}, nil
}

func (f *fakeAdminProductRPC) UpdateProduct(context.Context, *productrpc.UpdateProductReq, ...grpc.CallOption) (*productrpc.UpdateProductResp, error) {
	f.updateCalls++
	return &productpb.UpdateProductResp{}, nil
}

func (f *fakeAdminProductRPC) DeleteProduct(context.Context, *productrpc.DeleteProductReq, ...grpc.CallOption) (*productrpc.DeleteProductResp, error) {
	return &productpb.DeleteProductResp{}, nil
}

func (f *fakeAdminProductRPC) GetProductAdmin(context.Context, *productrpc.GetProductAdminReq, ...grpc.CallOption) (*productrpc.GetProductAdminResp, error) {
	return &productpb.GetProductAdminResp{}, nil
}

func (f *fakeAdminProductRPC) ListProductsAdmin(context.Context, *productrpc.ListProductsAdminReq, ...grpc.CallOption) (*productrpc.ListProductsAdminResp, error) {
	f.listAdminCalls++
	return &productpb.ListProductsAdminResp{}, nil
}

func (f *fakeAdminProductRPC) GetProductPublic(context.Context, *productrpc.GetProductPublicReq, ...grpc.CallOption) (*productrpc.GetProductPublicResp, error) {
	return nil, nil
}

func (f *fakeAdminProductRPC) ListProductsPublic(context.Context, *productrpc.ListProductsPublicReq, ...grpc.CallOption) (*productrpc.ListProductsPublicResp, error) {
	return nil, nil
}
