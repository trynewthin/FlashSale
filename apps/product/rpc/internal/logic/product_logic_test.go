package logic

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"flashsale/apps/product/rpc/internal/model"
	"flashsale/apps/product/rpc/internal/repository"
	"flashsale/apps/product/rpc/internal/svc"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/bwmarrin/snowflake"
)

type memoryProductRepo struct {
	byID  map[int64]*model.Product
	bySKU map[string]int64
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{byID: make(map[int64]*model.Product), bySKU: make(map[string]int64)}
}

func (m *memoryProductRepo) Create(_ context.Context, product *model.Product) error {
	if _, ok := m.bySKU[product.SkuCode]; ok {
		return repository.ErrProductNotFound // tests here won't cover duplicate path
	}
	now := time.Now()
	cp := *product
	cp.CreatedAt = now
	cp.UpdatedAt = now
	m.byID[cp.ID] = &cp
	m.bySKU[cp.SkuCode] = cp.ID
	return nil
}

func (m *memoryProductRepo) Update(_ context.Context, product *model.Product) error {
	old, ok := m.byID[product.ID]
	if !ok || old.DeletedAt != nil {
		return repository.ErrProductNotFound
	}
	old.Name = product.Name
	old.MainImage = product.MainImage
	old.Description = product.Description
	old.PriceCent = product.PriceCent
	old.Stock = product.Stock
	old.Status = product.Status
	old.UpdatedAt = time.Now()
	return nil
}

func (m *memoryProductRepo) SoftDelete(_ context.Context, productID int64, at time.Time) error {
	old, ok := m.byID[productID]
	if !ok || old.DeletedAt != nil {
		return repository.ErrProductNotFound
	}
	old.DeletedAt = &at
	old.Status = model.StatusOffShelf
	old.UpdatedAt = at
	return nil
}

func (m *memoryProductRepo) FindByIDAdmin(_ context.Context, productID int64) (*model.Product, error) {
	old, ok := m.byID[productID]
	if !ok || old.DeletedAt != nil {
		return nil, repository.ErrProductNotFound
	}
	cp := *old
	return &cp, nil
}

func (m *memoryProductRepo) FindByIDPublic(_ context.Context, productID int64) (*model.Product, error) {
	old, ok := m.byID[productID]
	if !ok || old.DeletedAt != nil || old.Status != model.StatusOnShelf {
		return nil, repository.ErrProductNotFound
	}
	cp := *old
	return &cp, nil
}

func (m *memoryProductRepo) ListAdmin(_ context.Context, query repository.ListQuery) ([]*model.Product, int64, error) {
	items := make([]*model.Product, 0, len(m.byID))
	for _, v := range m.byID {
		if !query.IncludeDeleted && v.DeletedAt != nil {
			continue
		}
		if strings.TrimSpace(query.Keyword) != "" && !strings.Contains(v.Name, query.Keyword) {
			continue
		}
		cp := *v
		items = append(items, &cp)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginate(items, query.Page, query.PageSize)
}

func (m *memoryProductRepo) ListPublic(_ context.Context, query repository.ListQuery) ([]*model.Product, int64, error) {
	items := make([]*model.Product, 0, len(m.byID))
	for _, v := range m.byID {
		if v.DeletedAt != nil || v.Status != model.StatusOnShelf {
			continue
		}
		if strings.TrimSpace(query.Keyword) != "" && !strings.Contains(v.Name, query.Keyword) {
			continue
		}
		cp := *v
		items = append(items, &cp)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return paginate(items, query.Page, query.PageSize)
}

func paginate(items []*model.Product, page, pageSize int64) ([]*model.Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	total := int64(len(items))
	start := (page - 1) * pageSize
	if start >= total {
		return []*model.Product{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}

func TestProductCreateUpdateDeleteAndPublicVisibility(t *testing.T) {
	repo := newMemoryProductRepo()
	node, _ := snowflake.NewNode(10)
	svcCtx := &svc.ServiceContext{ProductRepo: repo, IDNode: node}
	ctx := context.Background()

	create := NewCreateProductLogic(ctx, svcCtx)
	cResp, err := create.CreateProduct(&pb.CreateProductReq{
		Name:        "可乐",
		MainImage:   "https://img/a.png",
		Description: "desc",
		PriceCent:   399,
		Stock:       5,
		Status:      int32(model.StatusOnShelf),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cResp.GetProduct().GetSkuCode() == "" {
		t.Fatal("sku should not be empty")
	}

	pubGet := NewGetProductPublicLogic(ctx, svcCtx)
	if _, err := pubGet.GetProductPublic(&pb.GetProductPublicReq{ProductId: cResp.Product.ProductId}); err != nil {
		t.Fatalf("public get should success: %v", err)
	}

	update := NewUpdateProductLogic(ctx, svcCtx)
	_, err = update.UpdateProduct(&pb.UpdateProductReq{
		ProductId:   cResp.Product.ProductId,
		Name:        "可乐2",
		MainImage:   "https://img/b.png",
		Description: "d2",
		PriceCent:   499,
		Stock:       0,
		Status:      int32(model.StatusOffShelf),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	_, err = pubGet.GetProductPublic(&pb.GetProductPublicReq{ProductId: cResp.Product.ProductId})
	if err == nil {
		t.Fatal("off shelf product should not be visible in public get")
	}
	if code := errorx.FromError(err).Code; code != errorx.CodeProductNotFound {
		t.Fatalf("unexpected error code: %s", code)
	}

	deleteLogic := NewDeleteProductLogic(ctx, svcCtx)
	if _, err := deleteLogic.DeleteProduct(&pb.DeleteProductReq{ProductId: cResp.Product.ProductId}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	adminGet := NewGetProductAdminLogic(ctx, svcCtx)
	_, err = adminGet.GetProductAdmin(&pb.GetProductAdminReq{ProductId: cResp.Product.ProductId})
	if err == nil {
		t.Fatal("deleted product should not be found in admin get")
	}
	if code := errorx.FromError(err).Code; code != errorx.CodeProductNotFound {
		t.Fatalf("unexpected error code: %s", code)
	}
}
