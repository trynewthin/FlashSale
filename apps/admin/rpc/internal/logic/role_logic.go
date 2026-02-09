// logic 包包含相关应用代码。
package logic

import (
	"strings"
	"time"

	"flashsale/apps/admin/rpc/internal/model"
	"flashsale/apps/admin/rpc/internal/repository"
	"flashsale/apps/admin/rpc/pb"
	"flashsale/pkg/base/errorx"
)

// CreateRole 创建角色。
func (l *AdminLogic) CreateRole(in *pb.CreateRoleReq) (*pb.CreateRoleResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	code, err := normalizeRoleCode(in.RoleCode)
	if err != nil {
		return nil, err
	}
	name, err := normalizeDisplayName(in.RoleName)
	if err != nil {
		return nil, err
	}
	status, err := normalizeRoleStatus(in.Status)
	if err != nil {
		return nil, err
	}
	role := &model.Role{
		ID:       l.svcCtx.IDNode.Generate().Int64(),
		RoleCode: code,
		RoleName: name,
		Status:   status,
	}
	if err := l.svcCtx.AdminRepo.CreateRole(l.ctx, role); err != nil {
		return nil, mapRepoErr(err)
	}
	created, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, role.ID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	created.Domains, _ = l.svcCtx.AdminRepo.ListRoleDomains(l.ctx, created.ID)
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "create_role", "role", created.ID, "success", map[string]any{"role_code": created.RoleCode})
	return &pb.CreateRoleResp{Role: toRoleView(created)}, nil
}

// UpdateRole 更新角色。
func (l *AdminLogic) UpdateRole(in *pb.UpdateRoleReq) (*pb.UpdateRoleResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.RoleId, "role_id"); err != nil {
		return nil, err
	}
	name, err := normalizeDisplayName(in.RoleName)
	if err != nil {
		return nil, err
	}
	status, err := normalizeRoleStatus(in.Status)
	if err != nil {
		return nil, err
	}
	current, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if current.IsSystem {
		return nil, errorx.New(errorx.CodeSysBadRequest, "系统角色不允许编辑")
	}
	if err := l.svcCtx.AdminRepo.UpdateRole(l.ctx, in.RoleId, name, status, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	updated, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	updated.Domains, _ = l.svcCtx.AdminRepo.ListRoleDomains(l.ctx, updated.ID)
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "update_role", "role", in.RoleId, "success", map[string]any{"status": status})
	return &pb.UpdateRoleResp{Role: toRoleView(updated)}, nil
}

// DeleteRole 删除角色。
func (l *AdminLogic) DeleteRole(in *pb.DeleteRoleReq) (*pb.DeleteRoleResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.RoleId, "role_id"); err != nil {
		return nil, err
	}
	role, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if role.IsSystem {
		return nil, errorx.New(errorx.CodeSysBadRequest, "系统角色不允许删除")
	}
	bindings, err := l.svcCtx.AdminRepo.CountRoleBindings(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if bindings > 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "角色已绑定管理员，不能删除")
	}
	if err := l.svcCtx.AdminRepo.DeleteRole(l.ctx, in.RoleId); err != nil {
		return nil, mapRepoErr(err)
	}
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "delete_role", "role", in.RoleId, "success", map[string]any{})
	return &pb.DeleteRoleResp{Success: true}, nil
}

// GetRole 获取角色详情。
func (l *AdminLogic) GetRole(in *pb.GetRoleReq) (*pb.GetRoleResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.RoleId, "role_id"); err != nil {
		return nil, err
	}
	role, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	role.Domains, _ = l.svcCtx.AdminRepo.ListRoleDomains(l.ctx, role.ID)
	return &pb.GetRoleResp{Role: toRoleView(role)}, nil
}

// ListRoles 查询角色列表。
func (l *AdminLogic) ListRoles(in *pb.ListRolesReq) (*pb.ListRolesResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	status := int8(in.Status)
	if status != model.RoleStatusEnabled && status != model.RoleStatusDisabled {
		status = -1
	}
	items, total, err := l.svcCtx.AdminRepo.ListRoles(l.ctx, repository.RoleListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(in.Keyword),
		Status:   status,
	})
	if err != nil {
		return nil, mapRepoErr(err)
	}
	views := make([]*pb.RoleView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		item.Domains, _ = l.svcCtx.AdminRepo.ListRoleDomains(l.ctx, item.ID)
		views = append(views, toRoleView(item))
	}
	return &pb.ListRolesResp{Items: views, Total: total, Page: page, PageSize: pageSize}, nil
}

// SetRoleDomains 设置角色领域权限。
func (l *AdminLogic) SetRoleDomains(in *pb.SetRoleDomainsReq) (*pb.SetRoleDomainsResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.RoleId, "role_id"); err != nil {
		return nil, err
	}
	role, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, in.RoleId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if role.IsSystem {
		return nil, errorx.New(errorx.CodeSysBadRequest, "系统角色不允许修改权限域")
	}
	domains, err := normalizeDomains(in.Domains)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.AdminRepo.ReplaceRoleDomains(l.ctx, in.RoleId, domains); err != nil {
		return nil, mapRepoErr(err)
	}
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "set_role_domains", "role", in.RoleId, "success", map[string]any{"domains": domains})
	return &pb.SetRoleDomainsResp{Success: true}, nil
}
