// Package handler 提供 ops-control 的 HTTP 路由与瘦 handler。
// 每个 handler 仅负责请求解析、调用 service 层函数、序列化响应。
package handler

import (
	"encoding/json"
	"net/http"
)

// ─── 通用响应 ───

type apiResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// WriteOK 返回成功响应。
func WriteOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(apiResp{Code: "OK", Data: data})
}

// WriteErr 返回错误响应。
func WriteErr(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResp{Code: "ERROR", Message: message})
}

// WriteSSEEvent 发送 SSE 事件。
func WriteSSEEvent(w http.ResponseWriter, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if event != "" {
		if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("data: " + string(raw) + "\n\n")); err != nil {
		return err
	}
	return nil
}

// ParseBoolQuery 解析布尔查询参数。
func ParseBoolQuery(r *http.Request, key string, defaultValue bool) bool {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultValue
	}
	switch raw {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return defaultValue
	}
}
