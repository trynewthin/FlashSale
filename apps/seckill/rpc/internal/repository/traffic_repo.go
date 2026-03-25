package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

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
	incPV, _, incClick, incAttempt, incSuccess, incFail, incPaySuccess, incOrderClosed := eventIncrements(event)
	incUV, err := r.resolveUVIncrement(ctx, event, bucket)
	if err != nil {
		return err
	}
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

// resolveUVIncrement 返回 UV 增量：同分钟同访客只计一次。
func (r *MySQLSeckillRepository) resolveUVIncrement(ctx context.Context, event *model.TrafficEvent, bucket time.Time) (int64, error) {
	if event == nil || event.EventType != model.TrafficEventPV {
		return 0, nil
	}
	identity := visitorIdentity(event)
	if identity == "" {
		return 0, nil
	}
	_, err := r.db.ExecContext(ctx, insertTrafficRawSQL,
		event.ActivityID,
		event.ActivityItemID,
		trafficUVMarkerEvent,
		event.UserID,
		strings.TrimSpace(event.ClientID),
		bucket,
		buildUVMarkerIdempotencyKey(event.ActivityID, event.ActivityItemID, bucket, identity),
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return 0, nil
		}
		return 0, err
	}
	return 1, nil
}

// eventIncrements 把事件类型映射为聚合增量。
func eventIncrements(event *model.TrafficEvent) (pv, uv, click, attempt, success, fail, paySuccess, orderClosed int64) {
	if event == nil {
		return
	}
	switch event.EventType {
	case model.TrafficEventPV:
		pv = 1
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

// visitorIdentity 统一提取 UV 去重身份。
func visitorIdentity(event *model.TrafficEvent) string {
	if event == nil {
		return ""
	}
	if event.UserID > 0 {
		return fmt.Sprintf("u:%d", event.UserID)
	}
	clientID := strings.TrimSpace(event.ClientID)
	if clientID == "" {
		return ""
	}
	return "c:" + clientID
}

// buildUVMarkerIdempotencyKey 生成分钟级访客去重键。
func buildUVMarkerIdempotencyKey(activityID, activityItemID int64, bucket time.Time, identity string) string {
	return fmt.Sprintf("uv:%d:%d:%s:%s", activityID, activityItemID, bucket.UTC().Format("200601021504"), strings.TrimSpace(identity))
}
