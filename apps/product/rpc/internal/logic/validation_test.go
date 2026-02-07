package logic

import (
	"testing"
)

func TestProductValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeName(" "); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := normalizeName("商品A"); err != nil {
		t.Fatalf("normalizeName failed: %v", err)
	}

	if _, err := normalizeMainImage(""); err == nil {
		t.Fatal("empty image should fail")
	}
	if _, err := normalizeMainImage("https://example.com/a.png"); err != nil {
		t.Fatalf("normalizeMainImage failed: %v", err)
	}

	if _, err := normalizePriceCent(0); err == nil {
		t.Fatal("price <=0 should fail")
	}
	if _, err := normalizePriceCent(1); err != nil {
		t.Fatalf("normalizePriceCent failed: %v", err)
	}

	if _, err := normalizeStock(-1); err == nil {
		t.Fatal("negative stock should fail")
	}
	if _, err := normalizeStock(0); err != nil {
		t.Fatalf("normalizeStock failed: %v", err)
	}

	if _, err := normalizeStatus(2); err == nil {
		t.Fatal("invalid status should fail")
	}
	if _, err := normalizeStatus(0); err != nil {
		t.Fatalf("normalizeStatus(0) failed: %v", err)
	}
	if _, err := normalizeStatus(1); err != nil {
		t.Fatalf("normalizeStatus(1) failed: %v", err)
	}

	p, s := normalizePagination(0, 999)
	if p != 1 || s != 100 {
		t.Fatalf("pagination mismatch: got page=%d size=%d", p, s)
	}
}
