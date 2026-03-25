// repository 包包含相关应用代码。
package repository

import (
	"database/sql"
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
	trafficUVMarkerEvent = "uv_marker"
)

// MySQLSeckillRepository 是 SeckillRepository 的 MySQL 实现。
type MySQLSeckillRepository struct {
	db                      *sql.DB
	reserveDBUserLimitCheck bool
}

// NewMySQLSeckillRepository 创建 MySQL 秒杀仓储。
func NewMySQLSeckillRepository(db *sql.DB, reserveDBUserLimitCheck bool) *MySQLSeckillRepository {
	return &MySQLSeckillRepository{
		db:                      db,
		reserveDBUserLimitCheck: reserveDBUserLimitCheck,
	}
}
