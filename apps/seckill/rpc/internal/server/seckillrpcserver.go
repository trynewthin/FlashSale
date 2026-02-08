// server 包包含相关应用代码。
package server

import (
	"context"

	"flashsale/apps/seckill/rpc/internal/logic"
	"flashsale/apps/seckill/rpc/internal/svc"
	"flashsale/apps/seckill/rpc/pb"
	"flashsale/pkg/base/grpcerr"
)

// SeckillRpcServer 是秒杀 RPC 服务端实现。
type SeckillRpcServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedSeckillRpcServer
}

// NewSeckillRpcServer 创建秒杀 RPC 服务端。
func NewSeckillRpcServer(svcCtx *svc.ServiceContext) *SeckillRpcServer {
	return &SeckillRpcServer{svcCtx: svcCtx}
}

// CreateActivity 创建活动。
func (s *SeckillRpcServer) CreateActivity(ctx context.Context, in *pb.CreateActivityReq) (*pb.CreateActivityResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.CreateActivityReq{Title: req.Title, Description: req.Description, StyleConfigJson: req.StyleConfigJson, StartAtUnix: req.StartAtUnix, EndAtUnix: req.EndAtUnix, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.CreateActivity(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// UpdateActivity 更新活动。
func (s *SeckillRpcServer) UpdateActivity(ctx context.Context, in *pb.UpdateActivityReq) (*pb.UpdateActivityResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.UpdateActivityReq{ActivityId: req.ActivityId, Title: req.Title, Description: req.Description, StyleConfigJson: req.StyleConfigJson, StartAtUnix: req.StartAtUnix, EndAtUnix: req.EndAtUnix, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.UpdateActivity(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// DeleteActivity 删除活动。
func (s *SeckillRpcServer) DeleteActivity(ctx context.Context, in *pb.DeleteActivityReq) (*pb.DeleteActivityResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.DeleteActivityReq{ActivityId: req.ActivityId, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.DeleteActivity(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetActivityAdmin 查询管理端活动详情。
func (s *SeckillRpcServer) GetActivityAdmin(ctx context.Context, in *pb.GetActivityAdminReq) (*pb.GetActivityAdminResp, error) {
	if _, err := authorizeAdminSeckillDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.GetActivityAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListActivitiesAdmin 查询管理端活动列表。
func (s *SeckillRpcServer) ListActivitiesAdmin(ctx context.Context, in *pb.ListActivitiesAdminReq) (*pb.ListActivitiesAdminResp, error) {
	if _, err := authorizeAdminSeckillDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.ListActivitiesAdmin(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// UpsertActivityItem 新增或更新活动商品。
func (s *SeckillRpcServer) UpsertActivityItem(ctx context.Context, in *pb.UpsertActivityItemReq) (*pb.UpsertActivityItemResp, error) {
	if _, err := authorizeAdminSeckillDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.UpsertActivityItem(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// RemoveActivityItem 删除活动商品。
func (s *SeckillRpcServer) RemoveActivityItem(ctx context.Context, in *pb.RemoveActivityItemReq) (*pb.RemoveActivityItemResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.RemoveActivityItemReq{ActivityId: req.ActivityId, ItemId: req.ItemId, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.RemoveActivityItem(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// PublishActivity 发布活动。
func (s *SeckillRpcServer) PublishActivity(ctx context.Context, in *pb.PublishActivityReq) (*pb.PublishActivityResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.PublishActivityReq{ActivityId: req.ActivityId, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.PublishActivity(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// OfflineActivity 下线活动。
func (s *SeckillRpcServer) OfflineActivity(ctx context.Context, in *pb.OfflineActivityReq) (*pb.OfflineActivityResp, error) {
	adminID, err := authorizeAdminSeckillDomain(ctx)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	req := in
	if req != nil {
		req = &pb.OfflineActivityReq{ActivityId: req.ActivityId, AdminId: adminID}
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.OfflineActivity(req)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetActivityTraffic 查询活动流量统计。
func (s *SeckillRpcServer) GetActivityTraffic(ctx context.Context, in *pb.GetActivityTrafficReq) (*pb.GetActivityTrafficResp, error) {
	if _, err := authorizeAdminSeckillDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.GetActivityTraffic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListActivityOrders 查询活动订单追溯列表。
func (s *SeckillRpcServer) ListActivityOrders(ctx context.Context, in *pb.ListActivityOrdersReq) (*pb.ListActivityOrdersResp, error) {
	if _, err := authorizeAdminSeckillDomain(ctx); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.ListActivityOrders(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// ListActivitiesPublic 查询公开活动列表。
func (s *SeckillRpcServer) ListActivitiesPublic(ctx context.Context, in *pb.ListActivitiesPublicReq) (*pb.ListActivitiesPublicResp, error) {
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.ListActivitiesPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// GetActivityPublic 查询公开活动详情。
func (s *SeckillRpcServer) GetActivityPublic(ctx context.Context, in *pb.GetActivityPublicReq) (*pb.GetActivityPublicResp, error) {
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.GetActivityPublic(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// Purchase 处理秒杀购买请求。
func (s *SeckillRpcServer) Purchase(ctx context.Context, in *pb.PurchaseReq) (*pb.PurchaseResp, error) {
	if err := authorizeUser(ctx, in.GetUserId()); err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.Purchase(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}

// TrackEvent 处理秒杀埋点事件上报。
func (s *SeckillRpcServer) TrackEvent(ctx context.Context, in *pb.TrackEventReq) (*pb.TrackEventResp, error) {
	l := logic.NewSeckillLogic(ctx, s.svcCtx)
	resp, err := l.TrackEvent(in)
	if err != nil {
		return nil, grpcerr.ToStatus(err)
	}
	return resp, nil
}
