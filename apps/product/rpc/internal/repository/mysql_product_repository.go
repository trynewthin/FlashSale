// Package repository 提供基于 MySQL 的商品仓储实现。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/product/rpc/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	insertProductSQL = "INSERT INTO products (id, sku_code, name, main_image, description, price_cent, stock, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	updateProductSQL = "UPDATE products SET name = ?, main_image = ?, description = ?, price_cent = ?, stock = ?, status = ? WHERE id = ? AND deleted_at IS NULL"
	softDeleteSQL    = "UPDATE products SET deleted_at = ?, status = 0 WHERE id = ? AND deleted_at IS NULL"

	findByIDAdminSQL  = "SELECT id, sku_code, name, main_image, description, price_cent, stock, status, deleted_at, created_at, updated_at FROM products WHERE id = ? AND deleted_at IS NULL LIMIT 1"
	findByIDPublicSQL = "SELECT id, sku_code, name, main_image, description, price_cent, stock, status, deleted_at, created_at, updated_at FROM products WHERE id = ? AND status = 1 AND deleted_at IS NULL LIMIT 1"
)

// MySQLProductRepository 是 ProductRepository 的 MySQL 实现。
type MySQLProductRepository struct {
	db *sql.DB
}

// NewMySQLProductRepository 创建 MySQL 商品仓储。
func NewMySQLProductRepository(db *sql.DB) *MySQLProductRepository {
	return &MySQLProductRepository{db: db}
}

// Create 写入新商品记录。
func (r *MySQLProductRepository) Create(ctx context.Context, product *model.Product) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if product == nil {
		return fmt.Errorf("product is nil")
	}
	_, err := r.db.ExecContext(ctx, insertProductSQL,
		product.ID,
		product.SkuCode,
		product.Name,
		product.MainImage,
		product.Description,
		product.PriceCent,
		product.Stock,
		product.Status,
	)
	return err
}

// Update 更新商品核心信息。
func (r *MySQLProductRepository) Update(ctx context.Context, product *model.Product) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if product == nil {
		return fmt.Errorf("product is nil")
	}
	ret, err := r.db.ExecContext(ctx, updateProductSQL,
		product.Name,
		product.MainImage,
		product.Description,
		product.PriceCent,
		product.Stock,
		product.Status,
		product.ID,
	)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProductNotFound
	}
	return nil
}

// SoftDelete 软删除商品。
func (r *MySQLProductRepository) SoftDelete(ctx context.Context, productID int64, at time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, softDeleteSQL, at, productID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProductNotFound
	}
	return nil
}

// FindByIDAdmin 按管理员视角查询商品。
func (r *MySQLProductRepository) FindByIDAdmin(ctx context.Context, productID int64) (*model.Product, error) {
	return r.findOne(ctx, findByIDAdminSQL, productID)
}

// FindByIDPublic 按用户公开视角查询商品。
func (r *MySQLProductRepository) FindByIDPublic(ctx context.Context, productID int64) (*model.Product, error) {
	return r.findOne(ctx, findByIDPublicSQL, productID)
}

// ListAdmin 按管理员视角分页查询商品。
func (r *MySQLProductRepository) ListAdmin(ctx context.Context, query ListQuery) ([]*model.Product, int64, error) {
	where, args := buildListWhere(query.Keyword, query.IncludeDeleted, false)
	return r.list(ctx, where, args, query)
}

// ListPublic 按公开视角分页查询商品。
func (r *MySQLProductRepository) ListPublic(ctx context.Context, query ListQuery) ([]*model.Product, int64, error) {
	where, args := buildListWhere(query.Keyword, false, true)
	return r.list(ctx, where, args, query)
}

// IsDuplicateEntry 判断错误是否为 MySQL 唯一键冲突。
func IsDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

func (r *MySQLProductRepository) findOne(ctx context.Context, query string, arg any) (*model.Product, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	var (
		m         model.Product
		deletedAt sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&m.ID,
		&m.SkuCode,
		&m.Name,
		&m.MainImage,
		&m.Description,
		&m.PriceCent,
		&m.Stock,
		&m.Status,
		&deletedAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	if deletedAt.Valid {
		m.DeletedAt = &deletedAt.Time
	}
	return &m, nil
}

func (r *MySQLProductRepository) list(ctx context.Context, where string, args []any, query ListQuery) ([]*model.Product, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}

	countSQL := "SELECT COUNT(1) FROM products " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Product{}, 0, nil
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	listSQL := "SELECT id, sku_code, name, main_image, description, price_cent, stock, status, deleted_at, created_at, updated_at FROM products " +
		where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*model.Product, 0, pageSize)
	for rows.Next() {
		var (
			m         model.Product
			deletedAt sql.NullTime
		)
		if err := rows.Scan(
			&m.ID,
			&m.SkuCode,
			&m.Name,
			&m.MainImage,
			&m.Description,
			&m.PriceCent,
			&m.Stock,
			&m.Status,
			&deletedAt,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if deletedAt.Valid {
			m.DeletedAt = &deletedAt.Time
		}
		cp := m
		items = append(items, &cp)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func buildListWhere(keyword string, includeDeleted bool, publicOnly bool) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 2)

	if !includeDeleted {
		clauses = append(clauses, "deleted_at IS NULL")
	}
	if publicOnly {
		clauses = append(clauses, "status = 1")
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		clauses = append(clauses, "name LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
