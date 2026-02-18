package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrace_GeneratesTraceID(t *testing.T) {
	var gotTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = TraceIDFromContext(r.Context())
	})
	handler := Trace(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if gotTraceID == "" {
		t.Fatal("trace id should be generated")
	}
	if w.Header().Get("X-Trace-ID") == "" {
		t.Fatal("X-Trace-ID header should be set")
	}
	if w.Header().Get("X-Trace-ID") != gotTraceID {
		t.Fatal("header and context trace ID should match")
	}
}

func TestTrace_PreservesExistingTraceID(t *testing.T) {
	var gotTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = TraceIDFromContext(r.Context())
	})
	handler := Trace(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "custom-trace-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if gotTraceID != "custom-trace-id" {
		t.Fatalf("expected custom-trace-id, got %s", gotTraceID)
	}
}

func TestRecover_CatchesPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := Recover(nil, inner)
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	// Should not panic
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRecover_NoPanic(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Recover(nil, inner)
	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequestLogger_Runs(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := RequestLogger(nil, inner)
	req := httptest.NewRequest("GET", "/log-test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Fatal("inner handler should have been called")
	}
}

func TestTraceIDFromContext_NilContext(t *testing.T) {
	result := TraceIDFromContext(nil)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestTraceIDFromContext_NoValue(t *testing.T) {
	result := TraceIDFromContext(context.Background())
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}
