// Package middleware 提供网关中间件通用输出工具。
package middleware

import (
	"encoding/json"
	"net/http"

	"flashsale/pkg/base/responsex"
)

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responsex.Fail(err))
}
