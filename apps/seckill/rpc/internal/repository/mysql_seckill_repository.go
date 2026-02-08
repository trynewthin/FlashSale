// repository 包包含相关应用代码。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	insertActivitySQL = "INSERT INTO seckill_activities (id, title, description, style_config_json, start_at, end_at, status, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	updateActivitySQL = "UPDATE seckill_activities SET title = ?, description = ?, style_config_json = ?, start_at = ?, end_at = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL"
	deleteActivitySQL = "UPDATE seckill_activities SET deleted_at = ?, status = ?, updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL"

	insertItemSQL = "INSERT INTO seckill_activity_items (id, activity_id, product_id, sku_code, snapshot_name, snapshot_main_image, origin_price_cent, seckill_price_cent, reserved_stock_total, available_stock, sold_stock, user_limit_mode, user_limit_window_sec, user_limit_qty, max_qty_per_order, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	updateItemSQL = "UPDATE seckill_activity_items SET product_id = ?, sku_code = ?, snapshot_name = ?, snapshot_main_image = ?, origin_price_cent = ?, seckill_price_cent = ?, reserved_stock_total = ?, user_limit_mode = ?, user_limit_window_sec = ?, user_limit_qty = ?, max_qty_per_order = ?, status = ? WHERE id = ? AND activity_id = ?"
	deleteItemSQL = "DELETE FROM seckill_activity_items WHERE id = ? AND activity_id = ?"

	insertOrderLinkSQL   = "INSERT INTO seckill_order_links (id, order_id, order_no, user_id, activity_id, activity_item_id, quantity, order_status, payment_status, close_reason, last_synced_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	upsertOrderLinkSQL   = "INSERT INTO seckill_order_links (id, order_id, order_no, user_id, activity_id, activity_item_id, quantity, order_status, payment_status, close_reason, last_synced_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE order_no = VALUES(order_no), user_id = VALUES(user_id), activity_id = VALUES(activity_id), activity_item_id = VALUES(activity_item_id), quantity = VALUES(quantity), order_status = VALUES(order_status), payment_status = VALUES(payment_status), close_reason = VALUES(close_reason), last_synced_at = VALUES(last_synced_at)"
	insertTrafficRawSQL  = "INSERT INTO seckill_traffic_raw_events (activity_id, activity_item_id, event_type, user_id, client_id, event_time, idempotency_key) VALUES (?, ?, ?, ?, ?, ?, ?)"
	insertStockLedgerSQL = "INSERT INTO seckill_stock_ledger (id, activity_id, activity_item_id, order_id, event_type, delta, remain_after, idempotency_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
)

// MySQLSeckillRepository 是 SeckillRepository 的 MySQL 实现。
type MySQLSeckillRepository struct {
	db *sql.DB
}

// NewMySQLSeckillRepository 创建 MySQL 秒杀仓储。
func NewMySQLSeckillRepository(db *sql.DB) *MySQLSeckillRepository {
	return &MySQLSeckillRepository{db: db}
}

// CreateActivity 创建活动主记录。
func (r *MySQLSeckillRepository) CreateActivity(ctx context.Context, activity *model.Activity) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activity == nil {
		return fmt.Errorf("activity is nil")
	}
	_, err := r.db.ExecContext(ctx, insertActivitySQL,
		activity.ID,
		activity.Title,
		activity.Description,
		nullableJSONString(activity.StyleConfigJSON),
		activity.StartAt,
		activity.EndAt,
		activity.Status,
		activity.CreatedBy,
		activity.UpdatedBy,
	)
	return err
}

// UpdateActivity 更新活动基础信息。
func (r *MySQLSeckillRepository) UpdateActivity(ctx context.Context, activity *model.Activity) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activity == nil {
		return fmt.Errorf("activity is nil")
	}
	ret, err := r.db.ExecContext(ctx, updateActivitySQL,
		activity.Title,
		activity.Description,
		nullableJSONString(activity.StyleConfigJSON),
		activity.StartAt,
		activity.EndAt,
		activity.UpdatedBy,
		activity.ID,
	)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// SoftDeleteActivity 软删除活动。
func (r *MySQLSeckillRepository) SoftDeleteActivity(ctx context.Context, activityID, adminID int64, at time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, deleteActivitySQL, at, model.ActivityStatusOffline, adminID, at, activityID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// FindActivityByID 按主键查询活动。
func (r *MySQLSeckillRepository) FindActivityByID(ctx context.Context, activityID int64, includeDeleted bool) (*model.Activity, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	query := "SELECT id, title, description, style_config_json, start_at, end_at, status, created_by, updated_by, created_at, updated_at, deleted_at FROM seckill_activities WHERE id = ?"
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " LIMIT 1"
	row := r.db.QueryRowContext(ctx, query, activityID)
	return scanActivity(row)
}

// ListActivities 分页查询活动列表。
func (r *MySQLSeckillRepository) ListActivities(ctx context.Context, query ActivityListQuery) ([]*model.Activity, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	whereParts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if !query.IncludeDeleted {
		whereParts = append(whereParts, "deleted_at IS NULL")
	}
	if query.Status >= 0 {
		whereParts = append(whereParts, "status = ?")
		args = append(args, query.Status)
	}
	if query.PublicOnly {
		whereParts = append(whereParts, "status = ?")
		args = append(args, model.ActivityStatusPublished)
	}
	if strings.TrimSpace(query.Keyword) != "" {
		whereParts = append(whereParts, "title LIKE ?")
		args = append(args, "%"+strings.TrimSpace(query.Keyword)+"%")
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	var total int64
	countSQL := "SELECT COUNT(1) FROM seckill_activities" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Activity{}, 0, nil
	}
	offset := (query.Page - 1) * query.PageSize
	listSQL := "SELECT id, title, description, style_config_json, start_at, end_at, status, created_by, updated_by, created_at, updated_at, deleted_at FROM seckill_activities" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, listSQL, append(args, query.PageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.Activity, 0, query.PageSize)
	for rows.Next() {
		item, err := scanActivity(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

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

// MarkActivityStatus 更新活动状态。
func (r *MySQLSeckillRepository) MarkActivityStatus(ctx context.Context, activityID int64, status int8, adminID int64, now time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE seckill_activities SET status = ?, updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", status, adminID, now, activityID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// ResetAvailableStockByReserved 把可售库存重置为预占库存。
func (r *MySQLSeckillRepository) ResetAvailableStockByReserved(ctx context.Context, activityID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, "UPDATE seckill_activity_items SET available_stock = reserved_stock_total, sold_stock = 0 WHERE activity_id = ?", activityID)
	return err
}

// ReservePurchase 在数据库侧执行抢购扣减与台账写入。
func (r *MySQLSeckillRepository) ReservePurchase(ctx context.Context, activityID, itemID, userID, quantity int64, idempotencyKey string, now time.Time) (_ *model.PurchaseReservation, err error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		status  int8
		startAt time.Time
		endAt   time.Time
	)
	if err := tx.QueryRowContext(ctx, "SELECT status, start_at, end_at FROM seckill_activities WHERE id = ? AND deleted_at IS NULL FOR UPDATE", activityID).Scan(&status, &startAt, &endAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	if status != model.ActivityStatusPublished {
		return nil, ErrActivityStateConflict
	}
	if now.Before(startAt) || now.After(endAt) {
		return nil, ErrActivityStateConflict
	}

	item, err := r.findActivityItemForUpdate(ctx, tx, activityID, itemID)
	if err != nil {
		return nil, err
	}
	if item.Status != model.ItemStatusEnabled {
		return nil, ErrActivityItemNotFound
	}
	if quantity <= 0 || quantity > item.MaxQtyPerOrder {
		return nil, ErrActivityLimitExceeded
	}
	if item.AvailableStock < quantity {
		return nil, ErrActivityOutOfStock
	}
	if item.UserLimitQty > 0 {
		var bought int64
		switch item.UserLimitMode {
		case model.UserLimitModeWindowCompleted:
			if item.UserLimitWindowSec <= 0 {
				return nil, ErrActivityLimitExceeded
			}
			windowStart := now.Add(-time.Duration(item.UserLimitWindowSec) * time.Second)
			if err := tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity), 0) FROM seckill_order_links WHERE activity_id = ? AND user_id = ? AND order_status = ? AND close_reason IN (?, ?) AND last_synced_at >= ?",
				activityID,
				userID,
				90,
				"completed",
				"auto_completed",
				windowStart,
			).Scan(&bought); err != nil {
				return nil, err
			}
		default:
			if err := tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity), 0) FROM seckill_order_links WHERE activity_id = ? AND user_id = ?", activityID, userID).Scan(&bought); err != nil {
				return nil, err
			}
		}
		if bought+quantity > item.UserLimitQty {
			return nil, ErrActivityLimitExceeded
		}
	}

	ret, err := tx.ExecContext(ctx, "UPDATE seckill_activity_items SET available_stock = available_stock - ?, sold_stock = sold_stock + ? WHERE id = ? AND activity_id = ? AND available_stock >= ?", quantity, quantity, itemID, activityID, quantity)
	if err != nil {
		return nil, err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrActivityOutOfStock
	}
	remain := item.AvailableStock - quantity
	ledgerKey := strings.TrimSpace(idempotencyKey) + ":reserve"
	if _, err := tx.ExecContext(ctx, insertStockLedgerSQL,
		time.Now().UnixNano(),
		activityID,
		itemID,
		0,
		"purchase_reserve",
		-quantity,
		remain,
		ledgerKey,
	); err != nil {
		if isDuplicateEntry(err) {
			return nil, ErrIdempotencyConflict
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.PurchaseReservation{
		ActivityID:        activityID,
		ActivityItemID:    itemID,
		ProductID:         item.ProductID,
		Quantity:          quantity,
		RemainStock:       remain,
		SeckillPriceCent:  item.SeckillPriceCent,
		SKUCode:           item.SKUCode,
		SnapshotName:      item.SnapshotName,
		SnapshotMainImage: item.SnapshotMainImage,
	}, nil
}

// CompensateReleasePurchase 对失败链路执行数据库补偿回补。
func (r *MySQLSeckillRepository) CompensateReleasePurchase(ctx context.Context, activityID, itemID, quantity int64, idempotencyKey string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activityID <= 0 || itemID <= 0 || quantity <= 0 {
		return fmt.Errorf("release params invalid")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return fmt.Errorf("idempotency key empty")
	}
	ledgerKey := idempotencyKey + ":release"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ledgerID := time.Now().UnixNano()
	inserted, err := insertStockLedgerClaimTx(ctx, tx, ledgerID, activityID, itemID, 0, "purchase_compensate_release", quantity, ledgerKey)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	ret, err := tx.ExecContext(ctx, "UPDATE seckill_activity_items SET available_stock = available_stock + ?, sold_stock = IF(sold_stock >= ?, sold_stock - ?, 0) WHERE id = ? AND activity_id = ?", quantity, quantity, quantity, itemID, activityID)
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

	var remain int64
	if err := tx.QueryRowContext(ctx, "SELECT available_stock FROM seckill_activity_items WHERE id = ? AND activity_id = ? LIMIT 1", itemID, activityID).Scan(&remain); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE seckill_stock_ledger SET remain_after = ? WHERE id = ? LIMIT 1", remain, ledgerID); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateOrderLink 写入活动订单关联关系。
func (r *MySQLSeckillRepository) CreateOrderLink(ctx context.Context, link *model.OrderLink) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if link == nil {
		return fmt.Errorf("order link is nil")
	}
	_, err := r.db.ExecContext(ctx, insertOrderLinkSQL,
		link.ID,
		link.OrderID,
		link.OrderNo,
		link.UserID,
		link.ActivityID,
		link.ActivityItemID,
		link.Quantity,
		link.OrderStatus,
		link.PaymentStatus,
		link.CloseReason,
		link.LastSyncedAt,
	)
	return err
}

// SyncOrderLinkState 回流同步订单状态快照。
func (r *MySQLSeckillRepository) SyncOrderLinkState(ctx context.Context, sync *model.OrderStateSync) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if sync == nil {
		return fmt.Errorf("order state sync is nil")
	}
	if sync.OrderID <= 0 || sync.ActivityID <= 0 || sync.ActivityItemID <= 0 || sync.Quantity <= 0 {
		return fmt.Errorf("order state sync params invalid")
	}
	linkID := sync.OrderID
	if sync.LastSyncedAt.IsZero() {
		sync.LastSyncedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, upsertOrderLinkSQL,
		linkID,
		sync.OrderID,
		sync.OrderNo,
		sync.UserID,
		sync.ActivityID,
		sync.ActivityItemID,
		sync.Quantity,
		sync.OrderStatus,
		sync.PaymentStatus,
		sync.CloseReason,
		sync.LastSyncedAt,
	)
	return err
}

// ReleasePurchaseByOrder 在关闭类事件里回补活动库存并记账。
func (r *MySQLSeckillRepository) ReleasePurchaseByOrder(ctx context.Context, activityID, itemID, orderID, quantity int64, idempotencyKey string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activityID <= 0 || itemID <= 0 || orderID <= 0 || quantity <= 0 {
		return fmt.Errorf("release params invalid")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return fmt.Errorf("idempotency key empty")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ledgerID := time.Now().UnixNano()
	inserted, err := insertStockLedgerClaimTx(ctx, tx, ledgerID, activityID, itemID, orderID, "order_close_release", quantity, idempotencyKey)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	ret, err := tx.ExecContext(ctx, "UPDATE seckill_activity_items SET available_stock = available_stock + ?, sold_stock = IF(sold_stock >= ?, sold_stock - ?, 0) WHERE id = ? AND activity_id = ?", quantity, quantity, quantity, itemID, activityID)
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
	var remain int64
	if err := tx.QueryRowContext(ctx, "SELECT available_stock FROM seckill_activity_items WHERE id = ? AND activity_id = ? LIMIT 1", itemID, activityID).Scan(&remain); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE seckill_stock_ledger SET remain_after = ? WHERE id = ? LIMIT 1", remain, ledgerID); err != nil {
		return err
	}
	return tx.Commit()
}

// insertStockLedgerClaimTx 先插入库存台账幂等记录，再由调用方继续执行库存变更。
func insertStockLedgerClaimTx(ctx context.Context, tx *sql.Tx, ledgerID, activityID, itemID, orderID int64, eventType string, delta int64, idempotencyKey string) (bool, error) {
	_, err := tx.ExecContext(ctx, insertStockLedgerSQL,
		ledgerID,
		activityID,
		itemID,
		orderID,
		eventType,
		delta,
		0,
		idempotencyKey,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListOrderLinks 分页查询活动订单追溯列表。
func (r *MySQLSeckillRepository) ListOrderLinks(ctx context.Context, query OrderLinkListQuery) ([]*model.OrderLink, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	whereParts := []string{"activity_id = ?"}
	args := []any{query.ActivityID}
	if query.OrderStatus > 0 {
		whereParts = append(whereParts, "order_status = ?")
		args = append(args, query.OrderStatus)
	}
	if query.PaymentStatus > 0 {
		whereParts = append(whereParts, "payment_status = ?")
		args = append(args, query.PaymentStatus)
	}
	if query.UserID > 0 {
		whereParts = append(whereParts, "user_id = ?")
		args = append(args, query.UserID)
	}
	where := " WHERE " + strings.Join(whereParts, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM seckill_order_links"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.OrderLink{}, 0, nil
	}
	offset := (query.Page - 1) * query.PageSize
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, order_id, order_no, user_id, activity_id, activity_item_id, quantity, order_status, payment_status, close_reason, last_synced_at, created_at, updated_at FROM seckill_order_links"+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		append(args, query.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]*model.OrderLink, 0, query.PageSize)
	for rows.Next() {
		var item model.OrderLink
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.OrderNo,
			&item.UserID,
			&item.ActivityID,
			&item.ActivityItemID,
			&item.Quantity,
			&item.OrderStatus,
			&item.PaymentStatus,
			&item.CloseReason,
			&item.LastSyncedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		copyItem := item
		list = append(list, &copyItem)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// RecordTraffic 记录原始埋点并聚合到分钟表。
func (r *MySQLSeckillRepository) RecordTraffic(ctx context.Context, event *model.TrafficEvent) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if event == nil {
		return fmt.Errorf("traffic event is nil")
	}
	_, err := r.db.ExecContext(ctx, insertTrafficRawSQL,
		event.ActivityID,
		event.ActivityItemID,
		event.EventType,
		event.UserID,
		event.ClientID,
		event.OccurredAt,
		event.IdempotencyKey,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return nil
		}
		return err
	}
	bucket := event.OccurredAt.UTC().Truncate(time.Minute)
	incPV, incUV, incClick, incAttempt, incSuccess, incFail, incPaySuccess, incOrderClosed := eventIncrements(event)
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO seckill_traffic_agg_minute (bucket_minute, activity_id, activity_item_id, pv, uv, click, purchase_attempt, purchase_success, purchase_fail, pay_success, order_closed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE pv = pv + VALUES(pv), uv = uv + VALUES(uv), click = click + VALUES(click), purchase_attempt = purchase_attempt + VALUES(purchase_attempt), purchase_success = purchase_success + VALUES(purchase_success), purchase_fail = purchase_fail + VALUES(purchase_fail), pay_success = pay_success + VALUES(pay_success), order_closed = order_closed + VALUES(order_closed)",
		bucket,
		event.ActivityID,
		event.ActivityItemID,
		incPV,
		incUV,
		incClick,
		incAttempt,
		incSuccess,
		incFail,
		incPaySuccess,
		incOrderClosed,
	)
	return err
}

// ListTraffic 查询分钟聚合流量数据。
func (r *MySQLSeckillRepository) ListTraffic(ctx context.Context, activityID, activityItemID int64, from, to time.Time) ([]*model.TrafficBucket, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	whereParts := []string{"activity_id = ?", "bucket_minute >= ?", "bucket_minute <= ?"}
	args := []any{activityID, from, to}
	if activityItemID > 0 {
		whereParts = append(whereParts, "activity_item_id = ?")
		args = append(args, activityItemID)
	}
	query := "SELECT bucket_minute, activity_id, activity_item_id, pv, uv, click, purchase_attempt, purchase_success, purchase_fail, pay_success, order_closed FROM seckill_traffic_agg_minute WHERE " + strings.Join(whereParts, " AND ") + " ORDER BY bucket_minute ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*model.TrafficBucket, 0, 64)
	for rows.Next() {
		var item model.TrafficBucket
		if err := rows.Scan(
			&item.BucketMinute,
			&item.ActivityID,
			&item.ActivityItemID,
			&item.PV,
			&item.UV,
			&item.Click,
			&item.PurchaseAttempt,
			&item.PurchaseSuccess,
			&item.PurchaseFail,
			&item.PaySuccess,
			&item.OrderClosed,
		); err != nil {
			return nil, err
		}
		copyItem := item
		list = append(list, &copyItem)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
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

type scanner interface {
	Scan(dest ...any) error
}

// scanActivity 扫描活动行结果。
func scanActivity(s scanner) (*model.Activity, error) {
	var (
		item      model.Activity
		style     sql.NullString
		deletedAt sql.NullTime
	)
	if err := s.Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&style,
		&item.StartAt,
		&item.EndAt,
		&item.Status,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	if style.Valid {
		item.StyleConfigJSON = style.String
	}
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}
	copyItem := item
	return &copyItem, nil
}

// scanActivityItem 扫描活动商品行结果。
func scanActivityItem(s scanner) (*model.ActivityItem, error) {
	var item model.ActivityItem
	if err := s.Scan(
		&item.ID,
		&item.ActivityID,
		&item.ProductID,
		&item.SKUCode,
		&item.SnapshotName,
		&item.SnapshotMainImage,
		&item.OriginPriceCent,
		&item.SeckillPriceCent,
		&item.ReservedStockTotal,
		&item.AvailableStock,
		&item.SoldStock,
		&item.UserLimitMode,
		&item.UserLimitWindowSec,
		&item.UserLimitQty,
		&item.MaxQtyPerOrder,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityItemNotFound
		}
		return nil, err
	}
	copyItem := item
	return &copyItem, nil
}

// nullableJSONString 把空字符串转为 SQL NULL。
func nullableJSONString(raw string) any {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	return v
}

// eventIncrements 把事件类型映射为聚合增量。
func eventIncrements(event *model.TrafficEvent) (pv, uv, click, attempt, success, fail, paySuccess, orderClosed int64) {
	if event == nil {
		return
	}
	hasIdentity := event.UserID > 0 || strings.TrimSpace(event.ClientID) != ""
	switch event.EventType {
	case model.TrafficEventPV:
		pv = 1
		if hasIdentity {
			uv = 1
		}
	case model.TrafficEventClick:
		click = 1
	case model.TrafficEventPurchaseAttempt:
		attempt = 1
	case model.TrafficEventPurchaseSuccess:
		success = 1
	case model.TrafficEventPurchaseFail:
		fail = 1
	case model.TrafficEventPaySuccess:
		paySuccess = 1
	case model.TrafficEventOrderClosed:
		orderClosed = 1
	}
	return
}

// isDuplicateEntry 判断是否是 MySQL 唯一键冲突。
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
