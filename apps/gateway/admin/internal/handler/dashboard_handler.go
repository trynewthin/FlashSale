// handler 包包含相关应用代码。
package handler

import (
	"context"
	"net/http"
	"sync"

	"flashsale/apps/gateway/admin/internal/middleware"
	"flashsale/apps/gateway/admin/internal/svc"
	orderpb "flashsale/apps/order/rpc/pb"
	productpb "flashsale/apps/product/rpc/pb"
	seckillpb "flashsale/apps/seckill/rpc/pb"
	userpb "flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/rpcmeta"

	"go.uber.org/zap"
)

// DashboardHandler 处理运营工作台统计接口。
type DashboardHandler struct {
	svcCtx *svc.ServiceContext
}

// ensureInitialized 校验 DashboardHandler 所有依赖是否可用。
func (h *DashboardHandler) ensureInitialized() error {
	if h == nil || h.svcCtx == nil {
		return errorx.New(errorx.CodeSysInternal, "gateway not initialized")
	}
	if h.svcCtx.UserRPCCli == nil || h.svcCtx.ProductRPCCli == nil ||
		h.svcCtx.OrderRPCCli == nil || h.svcCtx.SeckillRPCCli == nil {
		return errorx.New(errorx.CodeSysInternal, "rpc client not initialized")
	}
	return nil
}

// logger 返回 svcCtx 中的 zap.Logger（若不可用则返回 nop logger）。
func (h *DashboardHandler) logger() *zap.Logger {
	if h != nil && h.svcCtx != nil && h.svcCtx.Logger != nil {
		return h.svcCtx.Logger
	}
	return zap.NewNop()
}

// DashboardStatsResp 聚合给前端的统计数据。
type DashboardStatsResp struct {
	Users    UserStats    `json:"users"`
	Products ProductStats `json:"products"`
	Orders   OrderStats   `json:"orders"`
	Seckill  SeckillStats `json:"seckill"`
}

// UserStats 用户统计信息。
type UserStats struct {
	Total int64 `json:"total"`
}

// ProductStats 商品统计信息。
type ProductStats struct {
	Total  int64 `json:"total"`
	OnSale int64 `json:"on_sale"`
}

// OrderStats 订单统计信息。
type OrderStats struct {
	Total          int64 `json:"total"`
	PendingPayment int64 `json:"pending_payment"`
	PendingReview  int64 `json:"pending_review"`
	PendingShip    int64 `json:"pending_ship"`
	Shipped        int64 `json:"shipped"`
	Closed         int64 `json:"closed"`
}

// SeckillStats 秒杀活动统计信息。
type SeckillStats struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Upcoming int64 `json:"upcoming"`
}

// GetStats 聚合各服务统计数据。
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureInitialized(); err != nil {
		writeFail(w, http.StatusInternalServerError, err)
		return
	}

	// 从 HTTP context 提取 access token，注入到 gRPC context
	token, ok := middleware.AccessTokenFromContext(r.Context())
	if !ok {
		writeFail(w, http.StatusUnauthorized, errorx.New(errorx.CodeAuthUnauthorized, "认证令牌缺失"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultRPCTimeout)
	defer cancel()
	ctx = rpcmeta.WithAccessToken(ctx, token)

	log := h.logger()
	var (
		mu   sync.Mutex
		resp DashboardStatsResp
	)
	var wg sync.WaitGroup

	// 1. 用户总数
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.svcCtx.UserRPCCli.ListUsers(ctx, &userpb.ListUsersReq{
			Page: 1, PageSize: 1, Status: -1,
		})
		if err != nil {
			log.Warn("dashboard: fetch user stats failed", zap.Error(err))
			return
		}
		mu.Lock()
		resp.Users.Total = res.Total
		mu.Unlock()
	}()

	// 2. 商品统计
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 全部商品（不含已删除）
		resAll, err := h.svcCtx.ProductRPCCli.ListProductsAdmin(ctx, &productpb.ListProductsAdminReq{
			Page: 1, PageSize: 1, IncludeDeleted: false,
		})
		if err != nil {
			log.Warn("dashboard: fetch product stats failed", zap.Error(err))
			return
		}
		// 在架商品（keyword="", include_deleted=false, 返回的就是 status=1 的在架商品总数）
		// 注意：ListProductsAdmin 返回的 total 已经包含了status过滤，
		// 但当前 proto 没有 status 过滤字段，所以我们只能用总数作近似值。
		// 若后续增加 status 过滤再细化。
		mu.Lock()
		resp.Products.Total = resAll.Total
		resp.Products.OnSale = resAll.Total // 未删除商品 ≈ 在架商品
		mu.Unlock()
	}()

	// 3. 订单统计（并发查各状态计数）
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 订单状态值定义（来自 order/rpc/internal/model）:
		// 10=待支付 20=待审核 30=待发货 40=已发货 90=已关闭
		// OrderStatus <= 0 不过滤（查全部）
		type statusQuery struct {
			status int32
			target *int64
		}
		queries := []statusQuery{
			{0, &resp.Orders.Total},           // 0=全部（不过滤）
			{10, &resp.Orders.PendingPayment}, // 10=待支付
			{20, &resp.Orders.PendingReview},  // 20=待审核
			{30, &resp.Orders.PendingShip},    // 30=待发货
			{40, &resp.Orders.Shipped},        // 40=已发货
			{90, &resp.Orders.Closed},         // 90=已关闭
		}
		var orderWg sync.WaitGroup
		for _, q := range queries {
			orderWg.Add(1)
			go func(sq statusQuery) {
				defer orderWg.Done()
				res, err := h.svcCtx.OrderRPCCli.ListOrdersAdmin(ctx, &orderpb.ListOrdersAdminReq{
					Page: 1, PageSize: 1, OrderStatus: sq.status,
				})
				if err != nil {
					log.Warn("dashboard: fetch order stats failed",
						zap.Int32("status", sq.status), zap.Error(err))
					return
				}
				mu.Lock()
				*sq.target = res.Total
				mu.Unlock()
			}(q)
		}
		orderWg.Wait()
	}()

	// 4. 秒杀活动统计
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 活动状态值（model.ActivityStatus*）: 0=草稿 1=已发布 2=已下线; -1=不过滤（查全部）
		type seckillQuery struct {
			status int32
			target *int64
		}
		queries := []seckillQuery{
			{-1, &resp.Seckill.Total},   // -1=全部（不过滤 status）
			{1, &resp.Seckill.Active},   // 1=已发布（进行中）
			{0, &resp.Seckill.Upcoming}, // 0=草稿（待上架）
		}
		var skWg sync.WaitGroup
		for _, q := range queries {
			skWg.Add(1)
			go func(sq seckillQuery) {
				defer skWg.Done()
				res, err := h.svcCtx.SeckillRPCCli.ListActivitiesAdmin(ctx, &seckillpb.ListActivitiesAdminReq{
					Page: 1, PageSize: 1, Status: sq.status,
				})
				if err != nil {
					log.Warn("dashboard: fetch seckill stats failed",
						zap.Int32("status", sq.status), zap.Error(err))
					return
				}
				mu.Lock()
				*sq.target = res.Total
				mu.Unlock()
			}(q)
		}
		skWg.Wait()
	}()

	wg.Wait()
	writeOK(w, resp)
}
