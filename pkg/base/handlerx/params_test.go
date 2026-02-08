// handlerx 包包含相关应用代码。
package handlerx

import (
	"net/http/httptest"
	"testing"
)

func TestParsePathInt64(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/products/1001", nil)
	req.SetPathValue("product_id", "1001")
	id, ok := ParsePathInt64(req, "product_id")
	if !ok || id != 1001 {
		t.Fatalf("ParsePathInt64 valid mismatch: id=%d ok=%v", id, ok)
	}

	reqBad := httptest.NewRequest("GET", "/api/v1/products/abc", nil)
	reqBad.SetPathValue("product_id", "abc")
	if _, ok := ParsePathInt64(reqBad, "product_id"); ok {
		t.Fatal("ParsePathInt64 should fail for non-numeric value")
	}

	reqZero := httptest.NewRequest("GET", "/api/v1/products/0", nil)
	reqZero.SetPathValue("product_id", "0")
	if _, ok := ParsePathInt64(reqZero, "product_id"); ok {
		t.Fatal("ParsePathInt64 should fail for non-positive value")
	}
}

func TestParseQueryInt64(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/products?page=2", nil)
	if got := ParseQueryInt64(req, "page", 1); got != 2 {
		t.Fatalf("ParseQueryInt64 mismatch: got=%d want=2", got)
	}

	reqInvalid := httptest.NewRequest("GET", "/api/v1/products?page=abc", nil)
	if got := ParseQueryInt64(reqInvalid, "page", 1); got != 1 {
		t.Fatalf("ParseQueryInt64 should fallback default, got=%d", got)
	}

	reqMissing := httptest.NewRequest("GET", "/api/v1/products", nil)
	if got := ParseQueryInt64(reqMissing, "page", 1); got != 1 {
		t.Fatalf("ParseQueryInt64 missing key should fallback default, got=%d", got)
	}
}

func TestParsePagination(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/products?page=3&page_size=999", nil)
	page, pageSize := ParsePagination(req, 1, 20, 100)
	if page != 3 || pageSize != 100 {
		t.Fatalf("ParsePagination mismatch: page=%d pageSize=%d", page, pageSize)
	}

	reqInvalid := httptest.NewRequest("GET", "/api/v1/products?page=-1&page_size=0", nil)
	page, pageSize = ParsePagination(reqInvalid, 1, 20, 100)
	if page != 1 || pageSize != 20 {
		t.Fatalf("ParsePagination invalid input fallback mismatch: page=%d pageSize=%d", page, pageSize)
	}
}

func TestParseBoolQuery(t *testing.T) {
	t.Parallel()

	trueCases := []string{"1", "true", "yes", "y", "on", "TRUE"}
	for _, v := range trueCases {
		req := httptest.NewRequest("GET", "/api/v1/products?include_deleted="+v, nil)
		if !ParseBoolQuery(req, "include_deleted") {
			t.Fatalf("ParseBoolQuery(%q) expected true", v)
		}
	}

	falseCases := []string{"0", "false", "no", "off", "", "abc"}
	for _, v := range falseCases {
		req := httptest.NewRequest("GET", "/api/v1/products?include_deleted="+v, nil)
		if ParseBoolQuery(req, "include_deleted") {
			t.Fatalf("ParseBoolQuery(%q) expected false", v)
		}
	}
}
