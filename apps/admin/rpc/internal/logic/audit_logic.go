// logic 包包含相关应用代码。
package logic

import (
	"strings"

	"flashsale/apps/admin/rpc/internal/repository"
	"flashsale/apps/admin/rpc/pb"
	"flashsale/pkg/base/errorx"
)

// ListAdminAuditLogs 查询管理员审计日志。
func (l *AdminLogic) ListAdminAuditLogs(in *pb.ListAdminAuditLogsReq) (*pb.ListAdminAuditLogsResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	items, total, err := l.svcCtx.AdminRepo.ListAuditLogs(l.ctx, repository.AuditLogListQuery{
		Page:      page,
		PageSize:  pageSize,
		AdminID:   in.AdminId,
		Action:    strings.TrimSpace(in.Action),
		TargetType: strings.TrimSpace(in.TargetType),
		TargetID:  in.TargetId,
	})
	if err != nil {
		return nil, mapRepoErr(err)
	}
	views := make([]*pb.AdminAuditLogView, 0, len(items))
	for _, item := range items {
		views = append(views, toAuditLogView(item))
	}
	return &pb.ListAdminAuditLogsResp{Items: views, Total: total, Page: page, PageSize: pageSize}, nil
}
