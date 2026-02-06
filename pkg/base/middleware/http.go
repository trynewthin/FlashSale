// Package middleware 提供 HTTP 链路追踪、恢复和请求日志中间件。
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"flashsale/pkg/base/errorx"
	"flashsale/pkg/base/responsex"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// contextKey 是中间件内部上下文键类型。
type contextKey string

const (
	traceIDHeader            = "X-Trace-ID"
	traceIDKey    contextKey = "trace_id"
)

// Trace 确保每个请求都有 trace_id，并写回响应头。
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceIDHeader)
		if traceID == "" {
			traceID = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), traceIDKey, traceID)
		w.Header().Set(traceIDHeader, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover 捕获 panic 并返回统一错误响应。
func Recover(logger *zap.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.ByteString("stack", debug.Stack()),
				)
				writeJSON(w, http.StatusInternalServerError, responsex.Fail(errorx.New(errorx.CodeSysInternal, "internal error")), TraceIDFromContext(r.Context()))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLogger 记录请求方法、路径、耗时和 trace_id。
func RequestLogger(logger *zap.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", time.Since(start)),
			zap.String("trace_id", TraceIDFromContext(r.Context())),
		)
	})
}

// TraceIDFromContext 从上下文读取 trace_id。
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// writeJSON 写入统一 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, payload responsex.Envelope, traceID string) {
	payload.TraceID = traceID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
