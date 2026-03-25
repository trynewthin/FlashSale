package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

var stockLedgerSeq atomic.Int64

func nextStockLedgerID() int64 {
	for {
		prev := stockLedgerSeq.Load()
		next := time.Now().UnixNano()
		if next <= prev {
			next = prev + 1
		}
		if stockLedgerSeq.CompareAndSwap(prev, next) {
			return next
		}
	}
}

// ReservePurchase 在数据库侧执行抢购扣减与台账写入。
func (r *MySQLSeckillRepository) ReservePurchase(ctx context.Context, activityID, itemID, userID, quantity int64, idempotencyKey string, now time.Time) (_ *model.PurchaseReservation, err error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	if activityID <= 0 || itemID <= 0 || quantity <= 0 {
		return nil, fmt.Errorf("reserve params invalid")
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ledgerID := nextStockLedgerID()
	ledgerKey := strings.TrimSpace(idempotencyKey) + ":reserve"
	if _, err := tx.ExecContext(ctx, insertStockLedgerSQL,
		ledgerID,
		activityID,
		itemID,
		0,
		"purchase_reserve",
		-quantity,
		0,
		ledgerKey,
	); err != nil {
		if isDuplicateEntry(err) {
			return nil, ErrIdempotencyConflict
		}
		return nil, err
	}

	// 热路径联表校验活动状态/时间窗口，减少一次前置查询往返。
	ret, err := tx.ExecContext(ctx,
		"UPDATE seckill_activity_items AS i "+
			"INNER JOIN seckill_activities AS a ON a.id = i.activity_id "+
			"SET i.available_stock = i.available_stock - ?, i.sold_stock = i.sold_stock + ? "+
			"WHERE i.id = ? AND i.activity_id = ? AND i.status = ? AND i.available_stock >= ? AND i.max_qty_per_order >= ? "+
			"AND a.deleted_at IS NULL AND a.status = ? AND a.start_at <= ? AND a.end_at >= ?",
		quantity,
		quantity,
		itemID,
		activityID,
		model.ItemStatusEnabled,
		quantity,
		quantity,
		model.ActivityStatusPublished,
		now,
		now,
	)
	if err != nil {
		return nil, err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		errClassify := r.classifyReserveMiss(ctx, activityID, itemID, quantity, now)
		if errClassify != nil {
			return nil, errClassify
		}
		return nil, ErrActivityStateConflict
	}
	var remain int64
	if r.reserveDBUserLimitCheck {
		var (
			userLimitMode   int8
			userLimitWindow int64
			userLimitQty    int64
		)
		if err := tx.QueryRowContext(ctx,
			"SELECT available_stock, user_limit_mode, user_limit_window_sec, user_limit_qty "+
				"FROM seckill_activity_items WHERE id = ? AND activity_id = ? LIMIT 1",
			itemID,
			activityID,
		).Scan(&remain, &userLimitMode, &userLimitWindow, &userLimitQty); err != nil {
			return nil, err
		}
		if userLimitQty > 0 {
			var bought int64
			switch userLimitMode {
			case model.UserLimitModeWindowCompleted:
				if userLimitWindow <= 0 {
					return nil, ErrActivityLimitExceeded
				}
				windowStart := now.Add(-time.Duration(userLimitWindow) * time.Second)
				if err := tx.QueryRowContext(ctx,
					"SELECT COALESCE(SUM(quantity), 0) FROM seckill_order_links WHERE activity_id = ? AND user_id = ? AND order_status = ? AND close_reason IN (?, ?) AND last_synced_at >= ?",
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
			if bought+quantity > userLimitQty {
				return nil, ErrActivityLimitExceeded
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE seckill_stock_ledger SET remain_after = ? WHERE id = ? LIMIT 1",
			remain,
			ledgerID,
		); err != nil {
			return nil, err
		}
	}
	if r.reserveDBUserLimitCheck && remain <= 0 {
		// 兼容上层可能需要库存回传的场景；关闭 DB 限购校验时不再额外查询。
		remain = 0
	}
	if !r.reserveDBUserLimitCheck {
		// 热路径在最终一致口径下跳过额外查询，将库存台账 remain_after 留给后续链路消费与对账使用。
		remain = 0
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.PurchaseReservation{
		ActivityID:     activityID,
		ActivityItemID: itemID,
		Quantity:       quantity,
		RemainStock:    remain,
	}, nil
}

// classifyReserveMiss 在原子扣减未命中时，返回更精确的业务错误。
func (r *MySQLSeckillRepository) classifyReserveMiss(ctx context.Context, activityID, itemID, quantity int64, now time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	var (
		dbActivityID sql.NullInt64
		deletedAt    sql.NullTime
		status       sql.NullInt64
		startAt      sql.NullTime
		endAt        sql.NullTime
		dbItemID     sql.NullInt64
		itemStatus   sql.NullInt64
		available    sql.NullInt64
		maxQty       sql.NullInt64
	)
	if err := r.db.QueryRowContext(ctx,
		"SELECT a.id, a.deleted_at, a.status, a.start_at, a.end_at, i.id, i.status, i.available_stock, i.max_qty_per_order "+
			"FROM seckill_activities AS a "+
			"LEFT JOIN seckill_activity_items AS i ON i.activity_id = a.id AND i.id = ? "+
			"WHERE a.id = ? LIMIT 1",
		itemID,
		activityID,
	).Scan(&dbActivityID, &deletedAt, &status, &startAt, &endAt, &dbItemID, &itemStatus, &available, &maxQty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrActivityNotFound
		}
		return err
	}
	if !dbActivityID.Valid || deletedAt.Valid {
		return ErrActivityNotFound
	}
	if !status.Valid || !startAt.Valid || !endAt.Valid {
		return ErrActivityStateConflict
	}
	if int8(status.Int64) != model.ActivityStatusPublished || now.Before(startAt.Time) || now.After(endAt.Time) {
		return ErrActivityStateConflict
	}
	if !dbItemID.Valid {
		return ErrActivityItemNotFound
	}
	if !itemStatus.Valid || int8(itemStatus.Int64) != model.ItemStatusEnabled {
		return ErrActivityItemNotFound
	}
	if maxQty.Valid && quantity > maxQty.Int64 {
		return ErrActivityLimitExceeded
	}
	if !available.Valid || available.Int64 < quantity {
		return ErrActivityOutOfStock
	}
	return ErrActivityStateConflict
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
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ledgerID := nextStockLedgerID()
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
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ledgerID := nextStockLedgerID()
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
