package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"
)

// CreateActivity 创建活动主记录。
func (r *MySQLSeckillRepository) CreateActivity(ctx context.Context, activity *model.Activity) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activity == nil {
		return fmt.Errorf("activity is nil")
	}
	_, err := r.db.ExecContext(ctx, insertActivitySQL,
		activity.ID,
		activity.Title,
		activity.Description,
		nullableJSONString(activity.StyleConfigJSON),
		activity.StartAt,
		activity.EndAt,
		activity.Status,
		activity.CreatedBy,
		activity.UpdatedBy,
	)
	return err
}

// UpdateActivity 更新活动基础信息。
func (r *MySQLSeckillRepository) UpdateActivity(ctx context.Context, activity *model.Activity) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	if activity == nil {
		return fmt.Errorf("activity is nil")
	}
	ret, err := r.db.ExecContext(ctx, updateActivitySQL,
		activity.Title,
		activity.Description,
		nullableJSONString(activity.StyleConfigJSON),
		activity.StartAt,
		activity.EndAt,
		activity.UpdatedBy,
		activity.ID,
	)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// SoftDeleteActivity 软删除活动。
func (r *MySQLSeckillRepository) SoftDeleteActivity(ctx context.Context, activityID, adminID int64, at time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, deleteActivitySQL, at, model.ActivityStatusOffline, adminID, at, activityID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// FindActivityByID 按主键查询活动。
func (r *MySQLSeckillRepository) FindActivityByID(ctx context.Context, activityID int64, includeDeleted bool) (*model.Activity, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repository db is nil")
	}
	query := "SELECT id, title, description, style_config_json, start_at, end_at, status, created_by, updated_by, created_at, updated_at, deleted_at FROM seckill_activities WHERE id = ?"
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " LIMIT 1"
	row := r.db.QueryRowContext(ctx, query, activityID)
	return scanActivity(row)
}

// ListActivities 分页查询活动列表。
func (r *MySQLSeckillRepository) ListActivities(ctx context.Context, query ActivityListQuery) ([]*model.Activity, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repository db is nil")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	whereParts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if !query.IncludeDeleted {
		whereParts = append(whereParts, "deleted_at IS NULL")
	}
	if query.Status >= 0 {
		whereParts = append(whereParts, "status = ?")
		args = append(args, query.Status)
	}
	if query.PublicOnly {
		whereParts = append(whereParts, "status = ?")
		args = append(args, model.ActivityStatusPublished)
	}
	if strings.TrimSpace(query.Keyword) != "" {
		whereParts = append(whereParts, "title LIKE ?")
		args = append(args, "%"+strings.TrimSpace(query.Keyword)+"%")
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	var total int64
	countSQL := "SELECT COUNT(1) FROM seckill_activities" + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.Activity{}, 0, nil
	}
	offset := (query.Page - 1) * query.PageSize
	listSQL := "SELECT id, title, description, style_config_json, start_at, end_at, status, created_by, updated_by, created_at, updated_at, deleted_at FROM seckill_activities" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, listSQL, append(args, query.PageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*model.Activity, 0, query.PageSize)
	for rows.Next() {
		item, err := scanActivity(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkActivityStatus 更新活动状态。
func (r *MySQLSeckillRepository) MarkActivityStatus(ctx context.Context, activityID int64, status int8, adminID int64, now time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repository db is nil")
	}
	ret, err := r.db.ExecContext(ctx, "UPDATE seckill_activities SET status = ?, updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", status, adminID, now, activityID)
	if err != nil {
		return err
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrActivityNotFound
	}
	return nil
}
