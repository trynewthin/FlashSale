package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

// CreateOrderLink 写入活动订单关联关系。
func (r *MySQLSeckillRepository) CreateOrderLink(ctx context.Context, link *model.OrderLink) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if link == nil {
		return fmt.Errorf("order link is nil")
	}
	_, err := r.db.ExecContext(ctx, upsertOrderLinkSQL,
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
