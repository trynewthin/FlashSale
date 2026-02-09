// logic 包包含相关应用代码。
package logic

import (
	"sort"
	"strings"
	"time"

	"flashsale/apps/admin/rpc/internal/model"
	"flashsale/apps/admin/rpc/internal/repository"
	"flashsale/apps/admin/rpc/pb"
	"flashsale/pkg/base/errorx"
)

// CreateAdmin 创建管理员。
func (l *AdminLogic) CreateAdmin(in *pb.CreateAdminReq) (*pb.CreateAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	username, err := normalizeUsername(in.Username)
	if err != nil {
		return nil, err
	}
	displayName, err := normalizeDisplayName(in.DisplayName)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordStrength(in.Password); err != nil {
		return nil, err
	}
	scope, err := normalizeDataScope(in.DataScope)
	if err != nil {
		return nil, err
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return nil, err
	}
	hashed, err := hashPassword(in.Password)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	admin := &model.Admin{
		ID:           l.svcCtx.IDNode.Generate().Int64(),
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: hashed,
		Status:       status,
		DataScope:    scope,
	}
	if err := l.svcCtx.AdminRepo.CreateAdmin(l.ctx, admin); err != nil {
		writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "create_admin", "admin", 0, "failed", map[string]any{"username": username})
		return nil, mapRepoErr(err)
	}
	created, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, admin.ID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	_ = setAdminRelations(l.ctx, l.svcCtx.AdminRepo, created)
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "create_admin", "admin", created.ID, "success", map[string]any{"username": created.Username})
	return &pb.CreateAdminResp{Admin: toAdminView(created)}, nil
}

// UpdateAdmin 更新管理员资料。
func (l *AdminLogic) UpdateAdmin(in *pb.UpdateAdminReq) (*pb.UpdateAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	displayName, err := normalizeDisplayName(in.DisplayName)
	if err != nil {
		return nil, err
	}
	scope, err := normalizeDataScope(in.DataScope)
	if err != nil {
		return nil, err
	}
	current, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if current.IsSuperAdmin && scope != model.DataScopeAll {
		return nil, errorx.New(errorx.CodeSysBadRequest, "超级管理员不可降权为 self")
	}
	if err := l.svcCtx.AdminRepo.UpdateAdminProfile(l.ctx, in.AdminId, displayName, scope, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	updated, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	_ = setAdminRelations(l.ctx, l.svcCtx.AdminRepo, updated)
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "update_admin", "admin", in.AdminId, "success", map[string]any{"data_scope": scope})
	return &pb.UpdateAdminResp{Admin: toAdminView(updated)}, nil
}

// SetAdminStatus 设置管理员状态。
func (l *AdminLogic) SetAdminStatus(in *pb.SetAdminStatusReq) (*pb.SetAdminStatusResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return nil, err
	}
	target, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if target.IsSuperAdmin && status == model.AdminStatusDisabled {
		count, countErr := l.svcCtx.AdminRepo.CountSuperAdmins(l.ctx)
		if countErr != nil {
			return nil, mapRepoErr(countErr)
		}
		if count <= 1 {
			return nil, errorx.New(errorx.CodeSysBadRequest, "至少保留一个超级管理员")
		}
	}
	if err := l.svcCtx.AdminRepo.SetAdminStatus(l.ctx, in.AdminId, status, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "set_admin_status", "admin", in.AdminId, "success", map[string]any{"status": status})
	return &pb.SetAdminStatusResp{Success: true}, nil
}

// ResetAdminPassword 重置管理员密码。
func (l *AdminLogic) ResetAdminPassword(in *pb.ResetAdminPasswordReq) (*pb.ResetAdminPasswordResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	if err := validatePasswordStrength(in.NewPassword); err != nil {
		return nil, err
	}
	hashed, err := hashPassword(in.NewPassword)
	if err != nil {
		return nil, errorx.Wrap(errorx.CodeSysInternal, "密码加密失败", err)
	}
	now := time.Now()
	if err := l.svcCtx.AdminRepo.UpdateAdminPassword(l.ctx, in.AdminId, hashed, now); err != nil {
		return nil, mapRepoErr(err)
	}
	_ = l.svcCtx.AdminRepo.RevokeRefreshTokensByAdmin(l.ctx, in.AdminId, now)
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "reset_admin_password", "admin", in.AdminId, "success", map[string]any{})
	return &pb.ResetAdminPasswordResp{Success: true}, nil
}

// DeleteAdmin 删除管理员。
func (l *AdminLogic) DeleteAdmin(in *pb.DeleteAdminReq) (*pb.DeleteAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	target, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if target.IsSuperAdmin {
		return nil, errorx.New(errorx.CodeSysBadRequest, "超级管理员不可删除")
	}
	if err := l.svcCtx.AdminRepo.SoftDeleteAdmin(l.ctx, in.AdminId, time.Now()); err != nil {
		return nil, mapRepoErr(err)
	}
	_ = l.svcCtx.AdminRepo.RevokeRefreshTokensByAdmin(l.ctx, in.AdminId, time.Now())
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "delete_admin", "admin", in.AdminId, "success", map[string]any{})
	return &pb.DeleteAdminResp{Success: true}, nil
}

// GetAdmin 获取管理员详情。
func (l *AdminLogic) GetAdmin(in *pb.GetAdminReq) (*pb.GetAdminResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	admin, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if err := setAdminRelations(l.ctx, l.svcCtx.AdminRepo, admin); err != nil {
		return nil, mapRepoErr(err)
	}
	return &pb.GetAdminResp{Admin: toAdminView(admin)}, nil
}

// ListAdmins 分页查询管理员。
func (l *AdminLogic) ListAdmins(in *pb.ListAdminsReq) (*pb.ListAdminsResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	page, pageSize := normalizePagination(in.Page, in.PageSize)
	status := int8(in.Status)
	if status != model.AdminStatusEnabled && status != model.AdminStatusDisabled {
		status = -1
	}
	items, total, err := l.svcCtx.AdminRepo.ListAdmins(l.ctx, repository.AdminListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(in.Keyword),
		Status:   status,
	})
	if err != nil {
		return nil, mapRepoErr(err)
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		_ = setAdminRelations(l.ctx, l.svcCtx.AdminRepo, item)
	}
	views := make([]*pb.AdminView, 0, len(items))
	for _, item := range items {
		views = append(views, toAdminView(item))
	}
	return &pb.ListAdminsResp{Items: views, Total: total, Page: page, PageSize: pageSize}, nil
}

// BindAdminRoles 绑定管理员角色。
func (l *AdminLogic) BindAdminRoles(in *pb.BindAdminRolesReq) (*pb.BindAdminRolesResp, error) {
	if in == nil {
		return nil, errorx.New(errorx.CodeSysBadRequest, "请求不能为空")
	}
	if err := mustPositiveID(in.OperatorAdminId, "operator_admin_id"); err != nil {
		return nil, err
	}
	if err := mustPositiveID(in.AdminId, "admin_id"); err != nil {
		return nil, err
	}
	target, err := l.svcCtx.AdminRepo.FindAdminByID(l.ctx, in.AdminId)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if target.IsSuperAdmin && len(in.RoleIds) == 0 {
		return nil, errorx.New(errorx.CodeSysBadRequest, "超级管理员必须保留角色")
	}
	roleSet := make(map[int64]struct{})
	roleIDs := make([]int64, 0, len(in.RoleIds))
	for _, roleID := range in.RoleIds {
		if roleID <= 0 {
			continue
		}
		if _, ok := roleSet[roleID]; ok {
			continue
		}
		if _, err := l.svcCtx.AdminRepo.FindRoleByID(l.ctx, roleID); err != nil {
			return nil, mapRepoErr(err)
		}
		roleSet[roleID] = struct{}{}
		roleIDs = append(roleIDs, roleID)
	}
	sort.Slice(roleIDs, func(i, j int) bool { return roleIDs[i] < roleIDs[j] })
	if err := l.svcCtx.AdminRepo.ReplaceAdminRoles(l.ctx, in.AdminId, roleIDs); err != nil {
		return nil, mapRepoErr(err)
	}
	writeAudit(l.ctx, l.svcCtx, in.OperatorAdminId, "bind_admin_roles", "admin", in.AdminId, "success", map[string]any{"role_ids": roleIDs})
	return &pb.BindAdminRolesResp{Success: true}, nil
}
