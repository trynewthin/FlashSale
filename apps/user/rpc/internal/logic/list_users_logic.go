// logic 包包含相关应用代码。
package logic

import (
	"context"
	"strings"

	"flashsale/apps/user/rpc/internal/repository"
	"flashsale/apps/user/rpc/internal/svc"
	"flashsale/apps/user/rpc/pb"
	"flashsale/pkg/base/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ListUsersLogic 封装用户列表查询接口的业务依赖。
type ListUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewListUsersLogic 创建 ListUsersLogic。
func NewListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsersLogic {
	return &ListUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListUsers 分页查询用户列表（管理端调用）。
func (l *ListUsersLogic) ListUsers(in *pb.ListUsersReq) (*pb.ListUsersResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.UserRepo == nil {
		return nil, errorx.New(errorx.CodeSysInternal, "服务未初始化")
	}

	page, pageSize := normalizePagination(in.Page, in.PageSize)
	status := int8(in.Status)
	if status != 0 && status != 1 {
		status = -1 // 不过滤
	}

	items, total, err := l.svcCtx.UserRepo.ListUsers(l.ctx, repository.UserListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(in.Keyword),
		Status:   status,
	})
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeDBError, "查询用户列表失败", err)
	}

	views := make([]*pb.UserView, 0, len(items))
	for _, u := range items {
		if u == nil {
			continue
		}
		view := &pb.UserView{
			UserId:        u.ID,
			Phone:         u.Phone,
			Nickname:      u.Nickname,
			Status:        int32(u.Status),
			LastLoginIp:   u.LastLoginIP,
			CreatedAtUnix: u.CreatedAt.Unix(),
			UpdatedAtUnix: u.UpdatedAt.Unix(),
		}
		if u.LastLoginAt != nil {
			view.LastLoginAtUnix = u.LastLoginAt.Unix()
		}
		views = append(views, view)
	}
	return &pb.ListUsersResp{
		List:     views,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
