package api

import (
	"encoding/json"
	"net/http"
)

type apiResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func WriteOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(apiResp{Code: "OK", Data: data})
}

func WriteErr(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResp{Code: "ERROR", Message: message})
}

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

func WriteSSEMessage(w http.ResponseWriter, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("data: " + string(raw) + "\n\n"))
	return err
}

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
