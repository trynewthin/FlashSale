package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

// UpsertActivityItem 创建或更新活动商品配置。
func (r *MySQLSeckillRepository) UpsertActivityItem(ctx context.Context, item *model.ActivityItem, isCreate bool) (*model.ActivityItem, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	if item == nil {
		return nil, fmt.Errorf("item is nil")
	}
	if item.ID <= 0 {
		return nil, fmt.Errorf("item id is invalid")
	}
	if isCreate {
		_, err := r.db.ExecContext(ctx, insertItemSQL,
			item.ID,
			item.ActivityID,
			item.ProductID,
			item.SKUCode,
			item.SnapshotName,
			item.SnapshotMainImage,
			item.OriginPriceCent,
			item.SeckillPriceCent,
			item.ReservedStockTotal,
			item.ReservedStockTotal,
			0,
			item.UserLimitMode,
			item.UserLimitWindowSec,
			item.UserLimitQty,
			item.MaxQtyPerOrder,
			item.Status,
		)
		if err != nil {
			return nil, err
		}
		return r.findActivityItemByID(ctx, item.ActivityID, item.ID)
	}
	ret, err := r.db.ExecContext(ctx, updateItemSQL,
		item.ProductID,
		item.SKUCode,
		item.SnapshotName,
		item.SnapshotMainImage,
		item.OriginPriceCent,
		item.SeckillPriceCent,
		item.ReservedStockTotal,
		item.UserLimitMode,
		item.UserLimitWindowSec,
		item.UserLimitQty,
		item.MaxQtyPerOrder,
		item.Status,
		item.ID,
		item.ActivityID,
	)
	if err != nil {
		return nil, err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrActivityItemNotFound
	}
	return r.findActivityItemByID(ctx, item.ActivityID, item.ID)
}

// RemoveActivityItem 删除活动商品配置。
func (r *MySQLSeckillRepository) RemoveActivityItem(ctx context.Context, activityID, itemID, adminID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	ret, err := tx.ExecContext(ctx, deleteItemSQL, itemID, activityID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityItemNotFound
	}
	if _, err := tx.ExecContext(ctx, "UPDATE seckill_activities SET updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", adminID, time.Now(), activityID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListActivityItems 查询活动下商品列表。
func (r *MySQLSeckillRepository) ListActivityItems(ctx context.Context, activityID int64, publicOnly bool) ([]*model.ActivityItem, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	query := "SELECT id, activity_id, product_id, sku_code, snapshot_name, snapshot_main_image, origin_price_cent, seckill_price_cent, reserved_stock_total, available_stock, sold_stock, user_limit_mode, user_limit_window_sec, user_limit_qty, max_qty_per_order, status, created_at, updated_at FROM seckill_activity_items WHERE activity_id = ?"
	args := []any{activityID}
	if publicOnly {
		query += " AND status = ?"
		args = append(args, model.ItemStatusEnabled)
	}
	query += " ORDER BY id ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*model.ActivityItem, 0, 8)
	for rows.Next() {
		item, err := scanActivityItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindActivityItem 查询指定活动商品。
func (r *MySQLSeckillRepository) FindActivityItem(ctx context.Context, activityID, itemID int64) (*model.ActivityItem, error) {
	return r.findActivityItemByID(ctx, activityID, itemID)
}

// findActivityItemByID 查询活动商品（内部复用）。
func (r *MySQLSeckillRepository) findActivityItemByID(ctx context.Context, activityID, itemID int64) (*model.ActivityItem, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	row := r.db.QueryRowContext(ctx, "SELECT id, activity_id, product_id, sku_code, snapshot_name, snapshot_main_image, origin_price_cent, seckill_price_cent, reserved_stock_total, available_stock, sold_stock, user_limit_mode, user_limit_window_sec, user_limit_qty, max_qty_per_order, status, created_at, updated_at FROM seckill_activity_items WHERE activity_id = ? AND id = ? LIMIT 1", activityID, itemID)
	return scanActivityItem(row)
}

// findActivityItemForUpdate 在事务中加锁读取活动商品。
func (r *MySQLSeckillRepository) findActivityItemForUpdate(ctx context.Context, tx *sql.Tx, activityID, itemID int64) (*model.ActivityItem, error) {
	row := tx.QueryRowContext(ctx, "SELECT id, activity_id, product_id, sku_code, snapshot_name, snapshot_main_image, origin_price_cent, seckill_price_cent, reserved_stock_total, available_stock, sold_stock, user_limit_mode, user_limit_window_sec, user_limit_qty, max_qty_per_order, status, created_at, updated_at FROM seckill_activity_items WHERE activity_id = ? AND id = ? LIMIT 1 FOR UPDATE", activityID, itemID)
	item, err := scanActivityItem(row)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// ResetAvailableStockByReserved 把可售库存重置为预占库存。
func (r *MySQLSeckillRepository) ResetAvailableStockByReserved(ctx context.Context, activityID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, "UPDATE seckill_activity_items SET available_stock = reserved_stock_total, sold_stock = 0 WHERE activity_id = ?", activityID)
	return err
}
