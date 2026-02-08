// seckillrpc 包包含相关应用代码。
package seckillrpc

import (
	"context"

	"flashsale/apps/seckill/rpc/pb"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	CreateActivityReq = pb.CreateActivityReq
	CreateActivityResp = pb.CreateActivityResp
	UpdateActivityReq = pb.UpdateActivityReq
	UpdateActivityResp = pb.UpdateActivityResp
	DeleteActivityReq = pb.DeleteActivityReq
	DeleteActivityResp = pb.DeleteActivityResp
	GetActivityAdminReq = pb.GetActivityAdminReq
	GetActivityAdminResp = pb.GetActivityAdminResp
	ListActivitiesAdminReq = pb.ListActivitiesAdminReq
	ListActivitiesAdminResp = pb.ListActivitiesAdminResp
	UpsertActivityItemReq = pb.UpsertActivityItemReq
	UpsertActivityItemResp = pb.UpsertActivityItemResp
	RemoveActivityItemReq = pb.RemoveActivityItemReq
	RemoveActivityItemResp = pb.RemoveActivityItemResp
	PublishActivityReq = pb.PublishActivityReq
	PublishActivityResp = pb.PublishActivityResp
	OfflineActivityReq = pb.OfflineActivityReq
	OfflineActivityResp = pb.OfflineActivityResp
	GetActivityTrafficReq = pb.GetActivityTrafficReq
	GetActivityTrafficResp = pb.GetActivityTrafficResp
	ListActivityOrdersReq = pb.ListActivityOrdersReq
	ListActivityOrdersResp = pb.ListActivityOrdersResp
	ListActivitiesPublicReq = pb.ListActivitiesPublicReq
	ListActivitiesPublicResp = pb.ListActivitiesPublicResp
	GetActivityPublicReq = pb.GetActivityPublicReq
	GetActivityPublicResp = pb.GetActivityPublicResp
	PurchaseReq = pb.PurchaseReq
	PurchaseResp = pb.PurchaseResp
	TrackEventReq = pb.TrackEventReq
	TrackEventResp = pb.TrackEventResp

	// SeckillRpc 定义秒杀 RPC 客户端接口。
	SeckillRpc interface {
		CreateActivity(ctx context.Context, in *CreateActivityReq, opts ...grpc.CallOption) (*CreateActivityResp, error)
		UpdateActivity(ctx context.Context, in *UpdateActivityReq, opts ...grpc.CallOption) (*UpdateActivityResp, error)
		DeleteActivity(ctx context.Context, in *DeleteActivityReq, opts ...grpc.CallOption) (*DeleteActivityResp, error)
		GetActivityAdmin(ctx context.Context, in *GetActivityAdminReq, opts ...grpc.CallOption) (*GetActivityAdminResp, error)
		ListActivitiesAdmin(ctx context.Context, in *ListActivitiesAdminReq, opts ...grpc.CallOption) (*ListActivitiesAdminResp, error)
		UpsertActivityItem(ctx context.Context, in *UpsertActivityItemReq, opts ...grpc.CallOption) (*UpsertActivityItemResp, error)
		RemoveActivityItem(ctx context.Context, in *RemoveActivityItemReq, opts ...grpc.CallOption) (*RemoveActivityItemResp, error)
		PublishActivity(ctx context.Context, in *PublishActivityReq, opts ...grpc.CallOption) (*PublishActivityResp, error)
		OfflineActivity(ctx context.Context, in *OfflineActivityReq, opts ...grpc.CallOption) (*OfflineActivityResp, error)
		GetActivityTraffic(ctx context.Context, in *GetActivityTrafficReq, opts ...grpc.CallOption) (*GetActivityTrafficResp, error)
		ListActivityOrders(ctx context.Context, in *ListActivityOrdersReq, opts ...grpc.CallOption) (*ListActivityOrdersResp, error)
		ListActivitiesPublic(ctx context.Context, in *ListActivitiesPublicReq, opts ...grpc.CallOption) (*ListActivitiesPublicResp, error)
		GetActivityPublic(ctx context.Context, in *GetActivityPublicReq, opts ...grpc.CallOption) (*GetActivityPublicResp, error)
		Purchase(ctx context.Context, in *PurchaseReq, opts ...grpc.CallOption) (*PurchaseResp, error)
		TrackEvent(ctx context.Context, in *TrackEventReq, opts ...grpc.CallOption) (*TrackEventResp, error)
	}

	defaultSeckillRpc struct {
		cli zrpc.Client
	}
)

// NewSeckillRpc 创建秒杀 RPC 客户端。
func NewSeckillRpc(cli zrpc.Client) SeckillRpc {
	return &defaultSeckillRpc{cli: cli}
}

func (m *defaultSeckillRpc) CreateActivity(ctx context.Context, in *CreateActivityReq, opts ...grpc.CallOption) (*CreateActivityResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).CreateActivity(ctx, in, opts...)
}

func (m *defaultSeckillRpc) UpdateActivity(ctx context.Context, in *UpdateActivityReq, opts ...grpc.CallOption) (*UpdateActivityResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).UpdateActivity(ctx, in, opts...)
}

func (m *defaultSeckillRpc) DeleteActivity(ctx context.Context, in *DeleteActivityReq, opts ...grpc.CallOption) (*DeleteActivityResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).DeleteActivity(ctx, in, opts...)
}

func (m *defaultSeckillRpc) GetActivityAdmin(ctx context.Context, in *GetActivityAdminReq, opts ...grpc.CallOption) (*GetActivityAdminResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).GetActivityAdmin(ctx, in, opts...)
}

func (m *defaultSeckillRpc) ListActivitiesAdmin(ctx context.Context, in *ListActivitiesAdminReq, opts ...grpc.CallOption) (*ListActivitiesAdminResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).ListActivitiesAdmin(ctx, in, opts...)
}

func (m *defaultSeckillRpc) UpsertActivityItem(ctx context.Context, in *UpsertActivityItemReq, opts ...grpc.CallOption) (*UpsertActivityItemResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).UpsertActivityItem(ctx, in, opts...)
}

func (m *defaultSeckillRpc) RemoveActivityItem(ctx context.Context, in *RemoveActivityItemReq, opts ...grpc.CallOption) (*RemoveActivityItemResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).RemoveActivityItem(ctx, in, opts...)
}

func (m *defaultSeckillRpc) PublishActivity(ctx context.Context, in *PublishActivityReq, opts ...grpc.CallOption) (*PublishActivityResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).PublishActivity(ctx, in, opts...)
}

func (m *defaultSeckillRpc) OfflineActivity(ctx context.Context, in *OfflineActivityReq, opts ...grpc.CallOption) (*OfflineActivityResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).OfflineActivity(ctx, in, opts...)
}

func (m *defaultSeckillRpc) GetActivityTraffic(ctx context.Context, in *GetActivityTrafficReq, opts ...grpc.CallOption) (*GetActivityTrafficResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).GetActivityTraffic(ctx, in, opts...)
}

func (m *defaultSeckillRpc) ListActivityOrders(ctx context.Context, in *ListActivityOrdersReq, opts ...grpc.CallOption) (*ListActivityOrdersResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).ListActivityOrders(ctx, in, opts...)
}

func (m *defaultSeckillRpc) ListActivitiesPublic(ctx context.Context, in *ListActivitiesPublicReq, opts ...grpc.CallOption) (*ListActivitiesPublicResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).ListActivitiesPublic(ctx, in, opts...)
}

func (m *defaultSeckillRpc) GetActivityPublic(ctx context.Context, in *GetActivityPublicReq, opts ...grpc.CallOption) (*GetActivityPublicResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).GetActivityPublic(ctx, in, opts...)
}

func (m *defaultSeckillRpc) Purchase(ctx context.Context, in *PurchaseReq, opts ...grpc.CallOption) (*PurchaseResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).Purchase(ctx, in, opts...)
}

func (m *defaultSeckillRpc) TrackEvent(ctx context.Context, in *TrackEventReq, opts ...grpc.CallOption) (*TrackEventResp, error) {
	return pb.NewSeckillRpcClient(m.cli.Conn()).TrackEvent(ctx, in, opts...)
}
