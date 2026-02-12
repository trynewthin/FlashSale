package handlerx

import (
	"encoding/json"
	"testing"
)

func TestJSONInt64_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    int64
		wantErr bool
	}{
		{name: "number", payload: `{"id":123}`, want: 123},
		{name: "string", payload: `{"id":"123"}`, want: 123},
		{name: "string with spaces", payload: `{"id":" 123 "}`, want: 123},
		{name: "negative", payload: `{"id":"-9"}`, want: -9},
		{name: "float number", payload: `{"id":1.2}`, wantErr: true},
		{name: "float string", payload: `{"id":"1.2"}`, wantErr: true},
		{name: "scientific", payload: `{"id":1e3}`, wantErr: true},
		{name: "empty string", payload: `{"id":""}`, wantErr: true},
		{name: "overflow", payload: `{"id":"9223372036854775808"}`, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req struct {
				ID JSONInt64 `json:"id"`
			}
			err := json.Unmarshal([]byte(tt.payload), &req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.ID.Int64() != tt.want {
				t.Fatalf("got %d, want %d", req.ID.Int64(), tt.want)
			}
		})
	}
}
