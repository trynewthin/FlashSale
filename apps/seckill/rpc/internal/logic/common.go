// logic 包包含相关应用代码。
package logic

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultPage     int64 = 1
	defaultPageSize int64 = 20
	maxPageSize     int64 = 100
)

// SeckillLogic 封装秒杀逻辑公共依赖。
type SeckillLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSeckillLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SeckillLogic {
	return &SeckillLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
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

func normalizeActivityBase(title, desc, style string, startUnix, endUnix int64) (string, string, string, time.Time, time.Time, error) {
	title = strings.TrimSpace(title)
	if l := utf8.RuneCountInString(title); l < 1 || l > 120 {
		return "", "", "", time.Time{}, time.Time{}, errorx.New(errorx.CodeSeckillInvalidConfig, "活动标题长度需在1到120之间")
	}
	desc = strings.TrimSpace(desc)
	if utf8.RuneCountInString(desc) > 4000 {
		return "", "", "", time.Time{}, time.Time{}, errorx.New(errorx.CodeSeckillInvalidConfig, "活动描述长度不能超过4000")
	}
	style = strings.TrimSpace(style)
	if utf8.RuneCountInString(style) > 8192 {
		return "", "", "", time.Time{}, time.Time{}, errorx.New(errorx.CodeSeckillInvalidConfig, "活动样式长度不能超过8192")
	}
	if style != "" && !json.Valid([]byte(style)) {
		return "", "", "", time.Time{}, time.Time{}, errorx.New(errorx.CodeSeckillInvalidConfig, "style_config_json 非法")
	}
	startAt := time.Unix(startUnix, 0)
	endAt := time.Unix(endUnix, 0)
	if startAt.IsZero() || endAt.IsZero() || !endAt.After(startAt) {
		return "", "", "", time.Time{}, time.Time{}, errorx.New(errorx.CodeSeckillInvalidConfig, "活动时间范围非法")
	}
	return title, desc, style, startAt, endAt, nil
}

func normalizeItemConfig(in *pb.UpsertActivityItemReq) (int8, error) {
	if in == nil {
		return 0, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.ActivityId <= 0 || in.ProductId <= 0 {
		return 0, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 product_id 非法")
	}
	if in.SeckillPriceCent <= 0 || in.ReservedStockTotal <= 0 {
		return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "秒杀价格与预占库存必须大于0")
	}
	if in.MaxQtyPerOrder <= 0 || in.MaxQtyPerOrder > 100 {
		return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "单次购买件数范围为1到100")
	}
	mode := int8(in.UserLimitMode)
	if mode != model.UserLimitModeNone && mode != model.UserLimitModeWindowCompleted {
		return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "user_limit_mode 非法")
	}
	if in.UserLimitQty < 0 || in.UserLimitWindowSec < 0 {
		return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "限购参数非法")
	}
	if mode == model.UserLimitModeWindowCompleted {
		if in.UserLimitWindowSec <= 0 || in.UserLimitQty <= 0 {
			return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "窗口限购模式要求 window_sec 与 user_limit_qty 均大于0")
		}
	}
	if in.Status != int32(model.ItemStatusEnabled) && in.Status != int32(model.ItemStatusDisabled) {
		return 0, errorx.New(errorx.CodeSeckillInvalidConfig, "item status 非法")
	}
	return mode, nil
}

func normalizeTrackEventType(eventType string) (string, error) {
	eventType = strings.TrimSpace(eventType)
	switch eventType {
	case model.TrafficEventPV,
		model.TrafficEventClick,
		model.TrafficEventPurchaseAttempt,
		model.TrafficEventPurchaseSuccess,
		model.TrafficEventPurchaseFail,
		model.TrafficEventPaySuccess,
		model.TrafficEventOrderClosed:
		return eventType, nil
	default:
		return "", errorx.New(errorx.CodeSysBadRequest, "event_type 非法")
	}
}

func mapRepoErr(err error) error {
	switch err {
	case nil:
		return nil
	case repository.ErrActivityNotFound:
		return errorx.New(errorx.CodeSeckillActivityNotFound, "活动不存在")
	case repository.ErrActivityItemNotFound:
		return errorx.New(errorx.CodeSeckillItemNotFound, "活动商品不存在")
	case repository.ErrActivityStateConflict:
		return errorx.New(errorx.CodeSeckillActivityNotPublished, "活动状态不允许当前操作")
	case repository.ErrActivityOutOfStock:
		return errorx.New(errorx.CodeSeckillOutOfStock, "库存不足")
	case repository.ErrActivityLimitExceeded:
		return errorx.New(errorx.CodeSeckillLimitExceeded, "限购超限")
	case repository.ErrIdempotencyConflict:
		return errorx.New(errorx.CodeSeckillPurchaseConflict, "请求冲突，请勿重复提交")
	default:
		if isContextCancellation(err) {
			return errorx.New(errorx.CodeSeckillPurchaseConflict, "请求冲突，请稍后重试")
		}
		if isMySQLTooManyConnections(err) {
			return errorx.New(errorx.CodeSeckillPurchaseConflict, "系统繁忙，请稍后重试")
		}
		if isMySQLTxnContention(err) {
			return errorx.New(errorx.CodeSeckillPurchaseConflict, "请求冲突，请稍后重试")
		}
		return errorx.Wrap(errorx.CodeDBError, "数据库操作失败", err)
	}
}

func isContextCancellation(err error) bool {
	return err == context.DeadlineExceeded || err == context.Canceled
}

func isMySQLTxnContention(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "error 1213") ||
		strings.Contains(msg, "deadlock found") ||
		strings.Contains(msg, "error 1205") ||
		strings.Contains(msg, "lock wait timeout")
}

// isMySQLTooManyConnections 判断是否为 MySQL 连接数打满导致的过载错误。
func isMySQLTooManyConnections(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "error 1040") || strings.Contains(msg, "too many connections")
}

// shouldLogReserveErrorAtErrorLevel 判断预扣失败是否需要按 error 级别记录。
// 说明：
//   - 秒杀高并发下，冲突类错误（超时/锁争用/库存不足/限购超限/幂等冲突）属于预期分支，
//     按 error 打日志会放大 IO 压力并污染告警。
//   - 非预期错误仍按 error 保留，便于故障定位。
func shouldLogReserveErrorAtErrorLevel(err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case repository.ErrActivityStateConflict,
		repository.ErrActivityOutOfStock,
		repository.ErrActivityLimitExceeded,
		repository.ErrIdempotencyConflict:
		return false
	}
	if isContextCancellation(err) || isMySQLTxnContention(err) || isMySQLTooManyConnections(err) {
		return false
	}
	return true
}

func toActivityAdmin(activity *model.Activity, items []*model.ActivityItem) *pb.ActivityAdmin {
	if activity == nil {
		return nil
	}
	out := &pb.ActivityAdmin{
		ActivityId:      activity.ID,
		Title:           activity.Title,
		Description:     activity.Description,
		StyleConfigJson: activity.StyleConfigJSON,
		StartAtUnix:     activity.StartAt.Unix(),
		EndAtUnix:       activity.EndAt.Unix(),
		Status:          int32(activity.Status),
		CreatedBy:       activity.CreatedBy,
		UpdatedBy:       activity.UpdatedBy,
		CreatedAtUnix:   activity.CreatedAt.Unix(),
		UpdatedAtUnix:   activity.UpdatedAt.Unix(),
		Items:           make([]*pb.ActivityItemAdmin, 0, len(items)),
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		out.Items = append(out.Items, &pb.ActivityItemAdmin{
			ItemId:             item.ID,
			ActivityId:         item.ActivityID,
			ProductId:          item.ProductID,
			SkuCode:            item.SKUCode,
			SnapshotName:       item.SnapshotName,
			SnapshotMainImage:  item.SnapshotMainImage,
			OriginPriceCent:    item.OriginPriceCent,
			SeckillPriceCent:   item.SeckillPriceCent,
			ReservedStockTotal: item.ReservedStockTotal,
			AvailableStock:     item.AvailableStock,
			SoldStock:          item.SoldStock,
			UserLimitMode:      int32(item.UserLimitMode),
			UserLimitWindowSec: item.UserLimitWindowSec,
			UserLimitQty:       item.UserLimitQty,
			MaxQtyPerOrder:     item.MaxQtyPerOrder,
			Status:             int32(item.Status),
			CreatedAtUnix:      item.CreatedAt.Unix(),
			UpdatedAtUnix:      item.UpdatedAt.Unix(),
		})
	}
	return out
}

func toActivityPublic(activity *model.Activity, items []*model.ActivityItem) *pb.ActivityPublic {
	if activity == nil {
		return nil
	}
	out := &pb.ActivityPublic{
		ActivityId:      activity.ID,
		Title:           activity.Title,
		Description:     activity.Description,
		StyleConfigJson: activity.StyleConfigJSON,
		StartAtUnix:     activity.StartAt.Unix(),
		EndAtUnix:       activity.EndAt.Unix(),
		Status:          int32(activity.Status),
		Items:           make([]*pb.ActivityItemPublic, 0, len(items)),
	}
	for _, item := range items {
		if item == nil || item.Status != model.ItemStatusEnabled {
			continue
		}
		out.Items = append(out.Items, &pb.ActivityItemPublic{
			ItemId:            item.ID,
			ProductId:         item.ProductID,
			SkuCode:           item.SKUCode,
			SnapshotName:      item.SnapshotName,
			SnapshotMainImage: item.SnapshotMainImage,
			OriginPriceCent:   item.OriginPriceCent,
			SeckillPriceCent:  item.SeckillPriceCent,
			InStock:           item.AvailableStock > 0,
			MaxQtyPerOrder:    item.MaxQtyPerOrder,
		})
	}
	return out
}

func toTrafficRows(items []*model.TrafficBucket) []*pb.TrafficBucket {
	rows := make([]*pb.TrafficBucket, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		rows = append(rows, &pb.TrafficBucket{
			BucketMinuteUnix: item.BucketMinute.Unix(),
			ActivityId:       item.ActivityID,
			ActivityItemId:   item.ActivityItemID,
			Pv:               item.PV,
			Uv:               item.UV,
			Click:            item.Click,
			PurchaseAttempt:  item.PurchaseAttempt,
			PurchaseSuccess:  item.PurchaseSuccess,
			PurchaseFail:     item.PurchaseFail,
			PaySuccess:       item.PaySuccess,
			OrderClosed:      item.OrderClosed,
		})
	}
	return rows
}

func toOrderLinkView(items []*model.OrderLink) []*pb.ActivityOrderView {
	out := make([]*pb.ActivityOrderView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &pb.ActivityOrderView{
			LinkId:           item.ID,
			OrderId:          item.OrderID,
			OrderNo:          item.OrderNo,
			UserId:           item.UserID,
			ActivityId:       item.ActivityID,
			ActivityItemId:   item.ActivityItemID,
			Quantity:         item.Quantity,
			OrderStatus:      int32(item.OrderStatus),
			PaymentStatus:    int32(item.PaymentStatus),
			CloseReason:      item.CloseReason,
			LastSyncedAtUnix: item.LastSyncedAt.Unix(),
			CreatedAtUnix:    item.CreatedAt.Unix(),
			UpdatedAtUnix:    item.UpdatedAt.Unix(),
		})
	}
	return out
}
