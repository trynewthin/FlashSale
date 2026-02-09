// logic 包包含相关应用代码。
package logic

import (
	"strconv"
	"strings"
	"time"

	productpb "flashsale/apps/product/rpc/pb"
	"flashsale/apps/seckill/rpc/internal/model"
	"flashsale/apps/seckill/rpc/internal/repository"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/grpcerr"
	"flashsale/pkg/base/rpcmeta"
)

// CreateActivity 创建秒杀活动。
func (l *SeckillLogic) CreateActivity(in *pb.CreateActivityReq) (*pb.CreateActivityResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeAuthForbidden, "admin_id 非法")
	}
	title, desc, style, startAt, endAt, err := normalizeActivityBase(in.Title, in.Description, in.StyleConfigJson, in.StartAtUnix, in.EndAtUnix)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	activity := &model.Activity{
		ID:              l.svcCtx.IDNode.Generate().Int64(),
		Title:           title,
		Description:     desc,
		StyleConfigJSON: style,
		StartAt:         startAt,
		EndAt:           endAt,
		Status:          model.ActivityStatusDraft,
		CreatedBy:       in.AdminId,
		UpdatedBy:       in.AdminId,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := l.svcCtx.SeckillRepo.CreateActivity(l.ctx, activity); err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "创建活动失败", err)
	}
	return &pb.CreateActivityResp{Activity: toActivityAdmin(activity, nil)}, nil
}

// UpdateActivity 更新秒杀活动。
func (l *SeckillLogic) UpdateActivity(in *pb.UpdateActivityReq) (*pb.UpdateActivityResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if in.ActivityId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 admin_id 非法")
	}
	old, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if old.Status == model.ActivityStatusPublished {
		return nil, errorx.New(errorx.CodeSeckillActivityNotPublished, "已发布活动不可修改")
	}
	title, desc, style, startAt, endAt, err := normalizeActivityBase(in.Title, in.Description, in.StyleConfigJson, in.StartAtUnix, in.EndAtUnix)
	if err != nil {
		return nil, err
	}
	old.Title = title
	old.Description = desc
	old.StyleConfigJSON = style
	old.StartAt = startAt
	old.EndAt = endAt
	old.UpdatedBy = in.AdminId
	if err := l.svcCtx.SeckillRepo.UpdateActivity(l.ctx, old); err != nil {
		return nil, mapRepoErr(err)
	}
	items, _ := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, old.ID, false)
	updated, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, old.ID, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.UpdateActivityResp{Activity: toActivityAdmin(updated, items)}, nil
}

// DeleteActivity 软删除秒杀活动。
func (l *SeckillLogic) DeleteActivity(in *pb.DeleteActivityReq) (*pb.DeleteActivityResp, error) {
	if in == nil || in.ActivityId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 admin_id 非法")
	}
	if err := l.svcCtx.SeckillRepo.SoftDeleteActivity(l.ctx, in.ActivityId, in.AdminId, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.DeleteActivityResp{ActivityId: in.ActivityId}, nil
}

// GetActivityAdmin 查询管理侧活动详情。
func (l *SeckillLogic) GetActivityAdmin(in *pb.GetActivityAdminReq) (*pb.GetActivityAdminResp, error) {
	if in == nil || in.ActivityId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法")
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	items, err := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动商品失败", err)
	}
	return &pb.GetActivityAdminResp{Activity: toActivityAdmin(activity, items)}, nil
}

// ListActivitiesAdmin 查询管理侧活动列表。
func (l *SeckillLogic) ListActivitiesAdmin(in *pb.ListActivitiesAdminReq) (*pb.ListActivitiesAdminResp, error) {
	if in == nil {
		in = &pb.ListActivitiesAdminReq{Status: -1}
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	status := int8(in.Status)
	if in.Status < 0 {
		status = -1
	}
	list, total, err := l.svcCtx.SeckillRepo.ListActivities(l.ctx, repository.ActivityListQuery{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        strings.TrimSpace(in.Keyword),
		Status:         status,
		IncludeDeleted: in.IncludeDeleted,
		PublicOnly:     false,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动列表失败", err)
	}
	out := make([]*pb.ActivityAdmin, 0, len(list))
	for _, activity := range list {
		items, _ := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, activity.ID, false)
		out = append(out, toActivityAdmin(activity, items))
	}
	return &pb.ListActivitiesAdminResp{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// UpsertActivityItem 创建或更新活动商品。
func (l *SeckillLogic) UpsertActivityItem(in *pb.UpsertActivityItemReq) (*pb.UpsertActivityItemResp, error) {
	if _, err := normalizeItemConfig(in); err != nil {
		return nil, err
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if activity.Status == model.ActivityStatusPublished {
		return nil, errorx.New(errorx.CodeSeckillActivityNotPublished, "已发布活动不可修改商品")
	}

	token, err := l.svcCtx.IssueProductManagementToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成商品服务令牌失败", err)
	}
	rpcCtx := rpcmeta.WithAccessToken(l.ctx, token)
	productResp, err := l.svcCtx.ProductRPCCli.GetProductAdmin(rpcCtx, &productpb.GetProductAdminReq{ProductId: in.ProductId})
	if err != nil {
		appErr := grpcerr.FromStatus(err)
		if appErr != nil {
			return nil, appErr
		}
		return nil, errorx.Wrap(errorx.CodeSysInternal, "查询商品失败", err)
	}
	if productResp == nil || productResp.Product == nil {
		return nil, errorx.New(errorx.CodeProductNotFound, "商品不存在")
	}

	itemID := in.ItemId
	isCreate := false
	if itemID <= 0 {
		itemID = l.svcCtx.IDNode.Generate().Int64()
		isCreate = true
	}
	item := &model.ActivityItem{
		ID:                 itemID,
		ActivityID:         in.ActivityId,
		ProductID:          in.ProductId,
		SKUCode:            productResp.Product.SkuCode,
		SnapshotName:       productResp.Product.Name,
		SnapshotMainImage:  productResp.Product.MainImage,
		OriginPriceCent:    productResp.Product.PriceCent,
		SeckillPriceCent:   in.SeckillPriceCent,
		ReservedStockTotal: in.ReservedStockTotal,
		UserLimitMode:      int8(in.UserLimitMode),
		UserLimitWindowSec: in.UserLimitWindowSec,
		UserLimitQty:       in.UserLimitQty,
		MaxQtyPerOrder:     in.MaxQtyPerOrder,
		Status:             int8(in.Status),
	}
	stored, err := l.svcCtx.SeckillRepo.UpsertActivityItem(l.ctx, item, isCreate)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.UpsertActivityItemResp{Item: toActivityAdmin(activity, []*model.ActivityItem{stored}).Items[0]}, nil
}

// RemoveActivityItem 删除活动商品。
func (l *SeckillLogic) RemoveActivityItem(in *pb.RemoveActivityItemReq) (*pb.RemoveActivityItemResp, error) {
	if in == nil || in.ActivityId <= 0 || in.ItemId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id、item_id 或 admin_id 非法")
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if activity.Status == model.ActivityStatusPublished {
		return nil, errorx.New(errorx.CodeSeckillActivityNotPublished, "已发布活动不可删除商品")
	}
	if err := l.svcCtx.SeckillRepo.RemoveActivityItem(l.ctx, in.ActivityId, in.ItemId, in.AdminId); err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.RemoveActivityItemResp{ActivityId: in.ActivityId, ItemId: in.ItemId}, nil
}

// PublishActivity 发布活动并预占库存。
func (l *SeckillLogic) PublishActivity(in *pb.PublishActivityReq) (*pb.PublishActivityResp, error) {
	if in == nil || in.ActivityId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 admin_id 非法")
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if activity.Status == model.ActivityStatusPublished {
		return nil, errorx.New(errorx.CodeSeckillActivityNotPublished, "活动已发布")
	}
	if !activity.EndAt.After(activity.StartAt) {
		return nil, errorx.New(errorx.CodeSeckillInvalidConfig, "活动时间非法")
	}
	items, err := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动商品失败", err)
	}
	if len(items) == 0 {
		return nil, errorx.New(errorx.CodeSeckillInvalidConfig, "活动至少配置一个商品")
	}
	enabledCount := 0
	for _, item := range items {
		if item == nil || item.Status != model.ItemStatusEnabled {
			continue
		}
		if item.ReservedStockTotal <= 0 {
			return nil, errorx.New(errorx.CodeSeckillInvalidConfig, "启用商品预占库存必须大于0")
		}
		enabledCount++
	}
	if enabledCount == 0 {
		return nil, errorx.New(errorx.CodeSeckillInvalidConfig, "活动至少配置一个启用商品")
	}
	token, err := l.svcCtx.IssueProductManagementToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成商品服务令牌失败", err)
	}
	reserved := make([]*model.ActivityItem, 0, len(items))
	published := false
	defer func() {
		if published {
			return
		}
		l.rollbackReservedActivityStocks(reserved, token)
	}()
	for _, item := range items {
		if item == nil || item.Status != model.ItemStatusEnabled {
			continue
		}
		_, err := l.svcCtx.ProductRPCCli.ReserveStockForActivity(rpcmeta.WithAccessToken(l.ctx, token), &productpb.ReserveStockForActivityReq{
			ProductId:      item.ProductID,
			Quantity:       item.ReservedStockTotal,
			ActivityId:     item.ActivityID,
			ActivityItemId: item.ID,
			IdempotencyKey: "ACT:" + fmtID(item.ActivityID) + ":ITEM:" + fmtID(item.ID) + ":reserve",
		})
		if err != nil {
			appErr := grpcerr.FromStatus(err)
			if appErr != nil {
				return nil, appErr
			}
			return nil, errorx.Wrap(errorx.CodeSysInternal, "预占库存失败", err)
		}
		reserved = append(reserved, item)
	}
	if err := l.svcCtx.SeckillRepo.ResetAvailableStockByReserved(l.ctx, in.ActivityId); err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "刷新活动库存失败", err)
	}
	refreshedItems, err := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动商品失败", err)
	}
	for _, item := range refreshedItems {
		if item == nil || item.Status != model.ItemStatusEnabled {
			continue
		}
		l.setCacheStock(l.ctx, item.ID, item.AvailableStock)
	}
	if err := l.svcCtx.SeckillRepo.MarkActivityStatus(l.ctx, in.ActivityId, model.ActivityStatusPublished, in.AdminId, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	published = true
	updated, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	updatedItems := refreshedItems
	return &pb.PublishActivityResp{Activity: toActivityAdmin(updated, updatedItems)}, nil
}

// OfflineActivity 下线活动并释放未售库存。
func (l *SeckillLogic) OfflineActivity(in *pb.OfflineActivityReq) (*pb.OfflineActivityResp, error) {
	if in == nil || in.ActivityId <= 0 || in.AdminId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 或 admin_id 非法")
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	items, err := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动商品失败", err)
	}
	token, err := l.svcCtx.IssueProductManagementToken()
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "生成商品服务令牌失败", err)
	}
	if activity.Status == model.ActivityStatusPublished {
		var releaseErr error
		for _, item := range items {
			if item == nil || item.AvailableStock <= 0 {
				if item != nil {
					l.deleteCacheStock(l.ctx, item.ID)
				}
				continue
			}
			_, err := l.svcCtx.ProductRPCCli.ReleaseStockForActivity(rpcmeta.WithAccessToken(l.ctx, token), &productpb.ReleaseStockForActivityReq{
				ProductId:      item.ProductID,
				Quantity:       item.AvailableStock,
				ActivityId:     item.ActivityID,
				ActivityItemId: item.ID,
				IdempotencyKey: "ACT:" + fmtID(item.ActivityID) + ":ITEM:" + fmtID(item.ID) + ":offline:release",
			})
			if err != nil {
				if appErr := grpcerr.FromStatus(err); appErr != nil {
					if releaseErr == nil {
						releaseErr = appErr
					}
					continue
				}
				if releaseErr == nil {
					releaseErr = errorx.Wrap(errorx.CodeSysInternal, "释放活动库存失败", err)
				}
				continue
			}
			l.deleteCacheStock(l.ctx, item.ID)
		}
		if releaseErr != nil {
			return nil, releaseErr
		}
	}
	if err := l.svcCtx.SeckillRepo.MarkActivityStatus(l.ctx, in.ActivityId, model.ActivityStatusOffline, in.AdminId, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	updated, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	updatedItems, _ := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, false)
	return &pb.OfflineActivityResp{Activity: toActivityAdmin(updated, updatedItems)}, nil
}

// GetActivityTraffic 查询活动流量统计。
func (l *SeckillLogic) GetActivityTraffic(in *pb.GetActivityTrafficReq) (*pb.GetActivityTrafficResp, error) {
	if in == nil || in.ActivityId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法")
	}
	to := time.Now().UTC().Truncate(time.Minute)
	if in.ToMinuteUnix > 0 {
		to = time.Unix(in.ToMinuteUnix, 0).UTC().Truncate(time.Minute)
	}
	from := to.Add(-1 * time.Hour)
	if in.FromMinuteUnix > 0 {
		from = time.Unix(in.FromMinuteUnix, 0).UTC().Truncate(time.Minute)
	}
	if from.After(to) {
		return nil, errorx.New(errorx.CodeSysBadRequest, "时间范围非法")
	}
	rows, err := l.svcCtx.SeckillRepo.ListTraffic(l.ctx, in.ActivityId, in.ActivityItemId, from, to)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询流量数据失败", err)
	}
	return &pb.GetActivityTrafficResp{Rows: toTrafficRows(rows)}, nil
}

// ListActivityOrders 查询活动订单追溯列表。
func (l *SeckillLogic) ListActivityOrders(in *pb.ListActivityOrdersReq) (*pb.ListActivityOrdersResp, error) {
	if in == nil || in.ActivityId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	list, total, err := l.svcCtx.SeckillRepo.ListOrderLinks(l.ctx, repository.OrderLinkListQuery{
		ActivityID:    in.ActivityId,
		Page:          page,
		PageSize:      pageSize,
		OrderStatus:   int8(in.OrderStatus),
		PaymentStatus: int8(in.PaymentStatus),
		UserID:        in.UserId,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动订单失败", err)
	}
	return &pb.ListActivityOrdersResp{List: toOrderLinkView(list), Total: total, Page: page, PageSize: pageSize}, nil
}

// ListActivitiesPublic 查询用户侧活动列表。
func (l *SeckillLogic) ListActivitiesPublic(in *pb.ListActivitiesPublicReq) (*pb.ListActivitiesPublicResp, error) {
	if in == nil {
		in = &pb.ListActivitiesPublicReq{}
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	list, total, err := l.svcCtx.SeckillRepo.ListActivities(l.ctx, repository.ActivityListQuery{
		Page:       page,
		PageSize:   pageSize,
		Status:     model.ActivityStatusPublished,
		PublicOnly: true,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动列表失败", err)
	}
	out := make([]*pb.ActivityPublic, 0, len(list))
	for _, activity := range list {
		items, _ := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, activity.ID, true)
		out = append(out, toActivityPublic(activity, items))
	}
	return &pb.ListActivitiesPublicResp{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetActivityPublic 查询用户侧活动详情。
func (l *SeckillLogic) GetActivityPublic(in *pb.GetActivityPublicReq) (*pb.GetActivityPublicResp, error) {
	if in == nil || in.ActivityId <= 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "activity_id 非法")
	}
	activity, err := l.svcCtx.SeckillRepo.FindActivityByID(l.ctx, in.ActivityId, false)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if activity.Status != model.ActivityStatusPublished {
		return nil, errorx.New(errorx.CodeSeckillActivityNotPublished, "活动未发布")
	}
	items, err := l.svcCtx.SeckillRepo.ListActivityItems(l.ctx, in.ActivityId, true)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询活动商品失败", err)
	}
	return &pb.GetActivityPublicResp{Activity: toActivityPublic(activity, items)}, nil
}

func fmtID(v int64) string {
	return strconv.FormatInt(v, 10)
}

// rollbackReservedActivityStocks 回滚发布流程中已预占的商品库存。
func (l *SeckillLogic) rollbackReservedActivityStocks(items []*model.ActivityItem, token string) {
	if l == nil || l.svcCtx == nil || len(items) == 0 {
		return
	}
	for _, done := range items {
		if done == nil || done.ReservedStockTotal <= 0 {
			continue
		}
		_, err := l.svcCtx.ProductRPCCli.ReleaseStockForActivity(rpcmeta.WithAccessToken(l.ctx, token), &productpb.ReleaseStockForActivityReq{
			ProductId:      done.ProductID,
			Quantity:       done.ReservedStockTotal,
			ActivityId:     done.ActivityID,
			ActivityItemId: done.ID,
			IdempotencyKey: "ACT:" + fmtID(done.ActivityID) + ":ITEM:" + fmtID(done.ID) + ":reserve:rollback",
		})
		if err != nil {
			l.Logger.Errorf("rollback reserved activity stock failed: activity_id=%d item_id=%d err=%v", done.ActivityID, done.ID, err)
		}
		l.deleteCacheStock(l.ctx, done.ID)
	}
}
