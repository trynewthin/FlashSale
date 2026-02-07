// Package handlerx 提供 HTTP handler 常用参数解析工具。
package handlerx

import (
	"net/http"
	"strconv"
	"strings"
)

// ParsePathInt64 从路径参数中解析正整数。
func ParsePathInt64(r *http.Request, key string) (int64, bool) {
	if r == nil {
		return 0, false
	}
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// ParseQueryInt64 从 query 参数中解析 int64，失败时返回默认值。
func ParseQueryInt64(r *http.Request, key string, defaultValue int64) int64 {
	if r == nil {
		return defaultValue
	}
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return defaultValue
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultValue
	}
	return n
}

// ParsePagination 解析分页参数并应用边界值。
func ParsePagination(r *http.Request, defaultPage, defaultPageSize, maxPageSize int64) (int64, int64) {
	page := ParseQueryInt64(r, "page", defaultPage)
	pageSize := ParseQueryInt64(r, "page_size", defaultPageSize)
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// ParseBoolQuery 从 query 参数中解析布尔值。
func ParseBoolQuery(r *http.Request, key string) bool {
	if r == nil {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
