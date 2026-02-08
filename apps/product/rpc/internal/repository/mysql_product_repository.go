// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
	reserveStockSQL  = "UPDATE products SET stock = stock - ? WHERE id = ? AND status = 1 AND deleted_at IS NULL AND stock >= ?"
	releaseStockSQL  = "UPDATE products SET stock = stock + ? WHERE id = ?"

	insertIdempotencySQL       = "INSERT INTO idempotency_records (idempotency_key, scene, status, expired_at) VALUES (?, ?, 0, ?)"
	queryIdempotencySQL        = "SELECT status, response_payload FROM idempotency_records WHERE idempotency_key = ? AND scene = ? LIMIT 1 FOR UPDATE"
	updateIdempotencyDoneSQL   = "UPDATE idempotency_records SET status = 1, response_payload = ?, expired_at = ? WHERE idempotency_key = ? AND scene = ?"
	deleteIdempotencyRecordSQL = "DELETE FROM idempotency_records WHERE idempotency_key = ? AND scene = ?"

	findByIDAdminSQL  = "SELECT id, sku_code, name, main_image, description, price_cent, stock, status, deleted_at, created_at, updated_at FROM products WHERE id = ? AND deleted_at IS NULL LIMIT 1"
	findByIDPublicSQL = "SELECT id, sku_code, name, main_image, description, price_cent, stock, status, deleted_at, created_at, updated_at FROM products WHERE id = ? AND status = 1 AND deleted_at IS NULL LIMIT 1"

	idempotencySceneReserveStock = "product_stock_reserve"
	idempotencySceneReleaseStock = "product_stock_release"
	idempotencyRecordTTL         = 24 * time.Hour
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

// ReserveStock 预扣库存（带幂等保障）。
func (r *MySQLProductRepository) ReserveStock(ctx context.Context, productID int64, quantity int64, _ string, idempotencyKey string) (_ int64, err error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("repository db is nil")
	}
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be positive")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return 0, fmt.Errorf("idempotency key is empty")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	inserted, replayRemain, err := lockIdempotencyRecordTx(ctx, tx, idempotencySceneReserveStock, idempotencyKey)
	if err != nil {
		return 0, err
	}
	if replayRemain != nil {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return *replayRemain, nil
	}

	remain, err := r.reserveStockTx(ctx, tx, productID, quantity)
	if err != nil {
		if inserted {
			_ = cleanupIdempotencyRecordTx(ctx, tx, idempotencySceneReserveStock, idempotencyKey)
		}
		return 0, err
	}
	if err := markIdempotencyDoneTx(ctx, tx, idempotencySceneReserveStock, idempotencyKey, remain); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return remain, nil
}

// ReleaseStock 回补库存（带幂等保障）。
func (r *MySQLProductRepository) ReleaseStock(ctx context.Context, productID int64, quantity int64, _ string, idempotencyKey string) (_ int64, err error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("repository db is nil")
	}
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be positive")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return 0, fmt.Errorf("idempotency key is empty")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	inserted, replayRemain, err := lockIdempotencyRecordTx(ctx, tx, idempotencySceneReleaseStock, idempotencyKey)
	if err != nil {
		return 0, err
	}
	if replayRemain != nil {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return *replayRemain, nil
	}

	remain, err := r.releaseStockTx(ctx, tx, productID, quantity)
	if err != nil {
		if inserted {
			_ = cleanupIdempotencyRecordTx(ctx, tx, idempotencySceneReleaseStock, idempotencyKey)
		}
		return 0, err
	}
	if err := markIdempotencyDoneTx(ctx, tx, idempotencySceneReleaseStock, idempotencyKey, remain); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return remain, nil
}

func (r *MySQLProductRepository) reserveStockTx(ctx context.Context, tx *sql.Tx, productID int64, quantity int64) (int64, error) {
	ret, err := tx.ExecContext(ctx, reserveStockSQL, quantity, productID, quantity)
	if err != nil {
		return 0, err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		stock, status, deleted, findErr := loadProductStateTx(ctx, tx, productID)
		if findErr != nil {
			return 0, findErr
		}
		if deleted || status != model.StatusOnShelf {
			return 0, ErrProductNotFound
		}
		if stock < quantity {
			return 0, ErrStockNotEnough
		}
		return 0, ErrProductNotFound
	}
	var remain int64
	if err := tx.QueryRowContext(ctx, "SELECT stock FROM products WHERE id = ? LIMIT 1", productID).Scan(&remain); err != nil {
		return 0, err
	}
	return remain, nil
}

func (r *MySQLProductRepository) releaseStockTx(ctx context.Context, tx *sql.Tx, productID int64, quantity int64) (int64, error) {
	ret, err := tx.ExecContext(ctx, releaseStockSQL, quantity, productID)
	if err != nil {
		return 0, err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, ErrProductNotFound
	}
	var remain int64
	if err := tx.QueryRowContext(ctx, "SELECT stock FROM products WHERE id = ? LIMIT 1", productID).Scan(&remain); err != nil {
		return 0, err
	}
	return remain, nil
}

func lockIdempotencyRecordTx(ctx context.Context, tx *sql.Tx, scene, key string) (inserted bool, replayRemain *int64, err error) {
	expiredAt := time.Now().Add(idempotencyRecordTTL)
	_, err = tx.ExecContext(ctx, insertIdempotencySQL, key, scene, expiredAt)
	if err == nil {
		return true, nil, nil
	}
	if !IsDuplicateEntry(err) {
		return false, nil, err
	}
	status, remain, readErr := readIdempotencyReplayTx(ctx, tx, scene, key)
	if readErr != nil {
		return false, nil, readErr
	}
	if status == 1 && remain != nil {
		return false, remain, nil
	}
	return false, nil, ErrIdempotencyInProgress
}

func readIdempotencyReplayTx(ctx context.Context, tx *sql.Tx, scene, key string) (int64, *int64, error) {
	var (
		status  int64
		payload sql.NullString
	)
	if err := tx.QueryRowContext(ctx, queryIdempotencySQL, key, scene).Scan(&status, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil, fmt.Errorf("idempotency record not found")
		}
		return 0, nil, err
	}
	if !payload.Valid || strings.TrimSpace(payload.String) == "" {
		return status, nil, nil
	}
	var decoded struct {
		RemainStock int64 `json:"remain_stock"`
	}
	if err := json.Unmarshal([]byte(payload.String), &decoded); err != nil {
		return 0, nil, err
	}
	return status, &decoded.RemainStock, nil
}

func markIdempotencyDoneTx(ctx context.Context, tx *sql.Tx, scene, key string, remain int64) error {
	payload, err := json.Marshal(map[string]any{"remain_stock": remain})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, updateIdempotencyDoneSQL, string(payload), time.Now().Add(idempotencyRecordTTL), key, scene)
	return err
}

func cleanupIdempotencyRecordTx(ctx context.Context, tx *sql.Tx, scene, key string) error {
	_, err := tx.ExecContext(ctx, deleteIdempotencyRecordSQL, key, scene)
	return err
}

func loadProductStateTx(ctx context.Context, tx *sql.Tx, productID int64) (stock int64, status int8, deleted bool, err error) {
	var deletedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT stock, status, deleted_at FROM products WHERE id = ? LIMIT 1", productID).Scan(&stock, &status, &deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, false, ErrProductNotFound
		}
		return 0, 0, false, err
	}
	return stock, status, deletedAt.Valid, nil
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
