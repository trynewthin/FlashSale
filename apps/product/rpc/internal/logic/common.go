// Package logic 实现商品 RPC 的业务流程。
package logic

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"flashsale/apps/product/rpc/internal/model"
	"flashsale/apps/product/rpc/pb"
	"flashsale/pkg/base/errorx"
)

const (
	defaultPage     int64 = 1
	defaultPageSize int64 = 20
	maxPageSize     int64 = 100
)

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	l := utf8.RuneCountInString(name)
	if l < 1 || l > 100 {
		return "", errorx.New(errorx.CodeSysBadRequest, "商品名称长度需在1到100之间")
	}
	return name, nil
}

func normalizeMainImage(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errorx.New(errorx.CodeSysBadRequest, "商品主图不能为空")
	}
	if utf8.RuneCountInString(v) > 512 {
		return "", errorx.New(errorx.CodeSysBadRequest, "商品主图长度不能超过512")
	}
	return v, nil
}

func normalizeDescription(v string) (string, error) {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > 2000 {
		return "", errorx.New(errorx.CodeSysBadRequest, "商品描述长度不能超过2000")
	}
	return v, nil
}

func normalizePriceCent(v int64) (int64, error) {
	if v <= 0 {
		return 0, errorx.New(errorx.CodeSysBadRequest, "商品价格必须大于0")
	}
	return v, nil
}

func normalizeStock(v int64) (int64, error) {
	if v < 0 {
		return 0, errorx.New(errorx.CodeSysBadRequest, "商品库存不能为负数")
	}
	return v, nil
}

func normalizeStatus(v int32) (int8, error) {
	if v != int32(model.StatusOffShelf) && v != int32(model.StatusOnShelf) {
		return 0, errorx.New(errorx.CodeProductInvalidStatus, "商品状态非法")
	}
	return int8(v), nil
}

func normalizePagination(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func generateSKU(productID int64) string {
	return fmt.Sprintf("SPU%d", productID)
}

func toAdminProduct(m *model.Product) *pb.AdminProduct {
	if m == nil {
		return nil
	}
	return &pb.AdminProduct{
		ProductId:     m.ID,
		SkuCode:       m.SkuCode,
		Name:          m.Name,
		MainImage:     m.MainImage,
		Description:   m.Description,
		PriceCent:     m.PriceCent,
		Stock:         m.Stock,
		Status:        int32(m.Status),
		CreatedAtUnix: m.CreatedAt.Unix(),
		UpdatedAtUnix: m.UpdatedAt.Unix(),
	}
}

func toPublicProduct(m *model.Product) *pb.PublicProduct {
	if m == nil {
		return nil
	}
	return &pb.PublicProduct{
		ProductId:   m.ID,
		SkuCode:     m.SkuCode,
		Name:        m.Name,
		MainImage:   m.MainImage,
		Description: m.Description,
		PriceCent:   m.PriceCent,
		InStock:     m.Stock > 0,
	}
}
