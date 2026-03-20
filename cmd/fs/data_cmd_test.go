package main

import "testing"

func TestDefaultNginxBaseURL_UsesNginxEnv(t *testing.T) {
	t.Setenv("FLASHSALE_NGINX_URL", "http://127.0.0.1:18000/")
	t.Setenv("FLASH_NGINX_HTTP_PORT", "")

	got := defaultNginxBaseURL("http://127.0.0.1:8083")
	if got != "http://127.0.0.1:18000" {
		t.Fatalf("defaultNginxBaseURL() = %q, want http://127.0.0.1:18000", got)
	}
}

func TestDefaultNginxBaseURL_UsesHostPortEnv(t *testing.T) {
	t.Setenv("FLASHSALE_NGINX_URL", "")
	t.Setenv("FLASH_NGINX_HTTP_PORT", "18000")

	got := defaultNginxBaseURL("http://127.0.0.1:8083")
	if got != "http://127.0.0.1:18000" {
		t.Fatalf("defaultNginxBaseURL() = %q, want http://127.0.0.1:18000", got)
	}
}

func TestDefaultNginxBaseURL_FallsBackToAdminBaseURL(t *testing.T) {
	t.Setenv("FLASHSALE_NGINX_URL", "")
	t.Setenv("FLASH_NGINX_HTTP_PORT", "")

	got := defaultNginxBaseURL("http://127.0.0.1:8083/")
	if got != "http://127.0.0.1:8083" {
		t.Fatalf("defaultNginxBaseURL() = %q, want http://127.0.0.1:8083", got)
	}
}
