// logic 包包含相关应用代码。
package logic

import "testing"

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name           string
		page, pageSize int64
		wantP, wantPS  int64
	}{
		{"zero values", 0, 0, 1, 20},
		{"negative values", -1, -5, 1, 20},
		{"normal values", 2, 10, 2, 10},
		{"over max page_size", 1, 200, 1, 20},
		{"boundary page_size=100", 1, 100, 1, 100},
		{"boundary page_size=101", 1, 101, 1, 20},
		{"page=1 pageSize=1", 1, 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ps := normalizePagination(tt.page, tt.pageSize)
			if p != tt.wantP || ps != tt.wantPS {
				t.Errorf("normalizePagination(%d, %d) = (%d, %d), want (%d, %d)",
					tt.page, tt.pageSize, p, ps, tt.wantP, tt.wantPS)
			}
		})
	}
}
