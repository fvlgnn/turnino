package main

import "testing"

func TestEnvOr(t *testing.T) {
	t.Setenv("TURNINO_TEST_VALUE", "configured")
	if got := envOr("TURNINO_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envOr() = %q, want configured", got)
	}

	t.Setenv("TURNINO_TEST_VALUE", "")
	if got := envOr("TURNINO_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("envOr() = %q, want fallback", got)
	}
}
