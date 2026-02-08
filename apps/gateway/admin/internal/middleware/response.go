// middleware 包包含相关应用代码。
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
