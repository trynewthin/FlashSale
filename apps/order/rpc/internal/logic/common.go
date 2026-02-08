// logic 包包含相关应用代码。
package logic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"flashsale/apps/order/rpc/internal/model"
	"flashsale/apps/order/rpc/pb"
	"flashsale/pkg/base/errorx"
)

const (
	defaultPage     int64 = 1
	defaultPageSize int64 = 20
	maxPageSize     int64 = 100

	reviewAmountThresholdCent int64 = 100000
	payTimeoutDur                   = 15 * time.Minute
	reviewTimeoutDur                = 30 * time.Minute
	autoReceiveDur                  = 7 * 24 * time.Hour
	virtualRefundDur                = 1 * time.Minute
	backgroundBatchSize             = 100
)

var phonePattern = regexp.MustCompile(`^[0-9+\-]{6,20}$`)

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

func normalizeOrderSource(v int32) (int8, error) {
	if v == 0 {
		return model.OrderSourceNormal, nil
	}
	if v != int32(model.OrderSourceNormal) && v != int32(model.OrderSourceSeckill) {
		return 0, errorx.New(errorx.CodeSysBadRequest, "order_source 非法")
	}
	return int8(v), nil
}

func normalizeReceiverInfo(name, phone, addr, remark string) (string, string, string, string, error) {
	name = strings.TrimSpace(name)
	if l := utf8.RuneCountInString(name); l < 1 || l > 64 {
		return "", "", "", "", errorx.New(errorx.CodeOrderInvalidReceiverInfo, "收货人长度需在1到64之间")
	}
	phone = strings.TrimSpace(phone)
	if !phonePattern.MatchString(phone) {
		return "", "", "", "", errorx.New(errorx.CodeOrderInvalidReceiverInfo, "收货手机号格式非法")
	}
	addr = strings.TrimSpace(addr)
	if l := utf8.RuneCountInString(addr); l < 1 || l > 512 {
		return "", "", "", "", errorx.New(errorx.CodeOrderInvalidReceiverInfo, "收货地址长度需在1到512之间")
	}
	remark = strings.TrimSpace(remark)
	if utf8.RuneCountInString(remark) > 200 {
		return "", "", "", "", errorx.New(errorx.CodeSysBadRequest, "买家备注长度不能超过200")
	}
	return name, phone, addr, remark, nil
}

func normalizeTrackingNo(v string) (string, error) {
	v = strings.TrimSpace(v)
	if l := utf8.RuneCountInString(v); l < 1 || l > 64 {
		return "", errorx.New(errorx.CodeSysBadRequest, "运单号长度需在1到64之间")
	}
	return v, nil
}

func normalizePayField(v string, field string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errorx.New(errorx.CodeSysBadRequest, fmt.Sprintf("%s 不能为空", field))
	}
	if utf8.RuneCountInString(v) > max {
		return "", errorx.New(errorx.CodeSysBadRequest, fmt.Sprintf("%s 长度不能超过%d", field, max))
	}
	return v, nil
}

func needManualReview(orderSource int8, totalAmountCent int64) bool {
	if orderSource == model.OrderSourceSeckill {
		return true
	}
	return totalAmountCent >= reviewAmountThresholdCent
}

func isManualPendingForCancel(o *model.Order) bool {
	return o != nil &&
		o.OrderStatus == model.OrderStatusPendingReview &&
		o.PaymentStatus == model.PaymentStatusPaid &&
		o.ReviewStatus == model.ReviewStatusManualPending
}

func mustJSON(v map[string]any) string {
	if len(v) == 0 {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func toOrderView(m *model.Order) *pb.OrderView {
	if m == nil {
		return nil
	}
	out := &pb.OrderView{
		OrderId:               m.ID,
		OrderNo:               m.OrderNo,
		UserId:                m.UserID,
		OrderSource:           int32(m.OrderSource),
		ProductId:             m.ProductID,
		SeckillActivityId:     m.SeckillActivityID,
		SeckillActivityItemId: m.SeckillActivityItemID,
		SkuCode:               m.SkuCode,
		ProductName:           m.ProductName,
		MainImage:             m.MainImage,
		UnitPriceCent:         m.UnitPriceCent,
		Quantity:              m.Quantity,
		TotalAmountCent:       m.TotalAmountCent,
		OrderStatus:           int32(m.OrderStatus),
		PaymentStatus:         int32(m.PaymentStatus),
		ReviewStatus:          int32(m.ReviewStatus),
		ShippingStatus:        int32(m.ShippingStatus),
		RefundStatus:          int32(m.RefundStatus),
		ReviewMode:            int32(m.ReviewMode),
		PayChannel:            m.PayChannel,
		PayReference:          m.PayReference,
		ReceiverName:          m.ReceiverName,
		ReceiverPhone:         m.ReceiverPhone,
		ReceiverAddress:       m.ReceiverAddress,
		BuyerRemark:           m.BuyerRemark,
		ReviewedBy:            m.ReviewedBy,
		ReviewReason:          m.ReviewReason,
		ShippedBy:             m.ShippedBy,
		TrackingNo:            m.TrackingNo,
		CloseReason:           m.CloseReason,
		CreatedAtUnix:         m.CreatedAt.Unix(),
		UpdatedAtUnix:         m.UpdatedAt.Unix(),
	}
	if m.PaidAt != nil {
		out.PaidAtUnix = m.PaidAt.Unix()
	}
	if m.ReviewDueAt != nil {
		out.ReviewDueAtUnix = m.ReviewDueAt.Unix()
	}
	if m.ReviewedAt != nil {
		out.ReviewedAtUnix = m.ReviewedAt.Unix()
	}
	if m.ShippedAt != nil {
		out.ShippedAtUnix = m.ShippedAt.Unix()
	}
	if m.RefundDueAt != nil {
		out.RefundDueAtUnix = m.RefundDueAt.Unix()
	}
	if m.RefundedAt != nil {
		out.RefundedAtUnix = m.RefundedAt.Unix()
	}
	if m.ClosedAt != nil {
		out.ClosedAtUnix = m.ClosedAt.Unix()
	}
	return out
}
