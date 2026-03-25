package repository

import (
	"database/sql"
	"errors"
	"strings"

	"flashsale/apps/seckill/rpc/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type scanner interface {
	Scan(dest ...any) error
}

// scanActivity 扫描活动行结果。
func scanActivity(s scanner) (*model.Activity, error) {
	var (
		item      model.Activity
		style     sql.NullString
		deletedAt sql.NullTime
	)
	if err := s.Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&style,
		&item.StartAt,
		&item.EndAt,
		&item.Status,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	if style.Valid {
		item.StyleConfigJSON = style.String
	}
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}
	copyItem := item
	return &copyItem, nil
}

// scanActivityItem 扫描活动商品行结果。
func scanActivityItem(s scanner) (*model.ActivityItem, error) {
	var item model.ActivityItem
	if err := s.Scan(
		&item.ID,
		&item.ActivityID,
		&item.ProductID,
		&item.SKUCode,
		&item.SnapshotName,
		&item.SnapshotMainImage,
		&item.OriginPriceCent,
		&item.SeckillPriceCent,
		&item.ReservedStockTotal,
		&item.AvailableStock,
		&item.SoldStock,
		&item.UserLimitMode,
		&item.UserLimitWindowSec,
		&item.UserLimitQty,
		&item.MaxQtyPerOrder,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrActivityItemNotFound
		}
		return nil, err
	}
	copyItem := item
	return &copyItem, nil
}

// nullableJSONString 把空字符串转为 SQL NULL。
func nullableJSONString(raw string) any {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	return v
}

// isDuplicateEntry 判断是否是 MySQL 唯一键冲突。
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
