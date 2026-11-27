package pool

import (
	"errors"
	"testing"
	"time"
)

func TestUpstreamCooldownScopeIsolation(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	edit := "      - alias: coding-default\n"
	replacement := "      - alias: coding-other\n        upstream_model: fixture-other\n        routing: fixed\n        primary_relay: warp-1\n        fallback_relays: []\n        affinity: none\n      - alias: coding-default\n"
	a := readyAt(t, []string{edit, replacement}, func() time.Time { return now })
	recorder, ok := any(a).(interface {
		RecordUpstreamResponse(string, string, int, string, string) error
	})
	if !ok {
		t.Fatal("pool belum mencatat batas upstream scoped")
	}
	if err := recorder.RecordUpstreamResponse("primary", "coding-default", 429, "120", "model"); err != nil {
		t.Fatal(err)
	}
	admit := func(alias string) error {
		t.Helper()
		r, ok := a.Begin(100)
		if !ok {
			t.Fatal("begin fixture ditolak")
		}
		defer r.Release()
		return admitOnly(r, "primary", alias, 0)
	}
	if err := admit("coding-default"); !errors.Is(err, ErrNoEligibleRelay) {
		t.Fatalf("scope cooldown tidak memblokir model: %v", err)
	}
	if err := admit("coding-other"); err != nil {
		t.Fatalf("cooldown model memblokir model lain pada relay sama: %v", err)
	}
	if a.View().Relays[0].CooldownActive || !a.View().Ready {
		t.Fatal("cooldown scoped mengubah kesehatan relay atau readiness")
	}
}

func TestUnknown429AppliesUpstreamCooldownAndRedactedClassification(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	edit := "      - alias: coding-default\n"
	replacement := "      - alias: coding-other\n        upstream_model: fixture-other\n        routing: fixed\n        primary_relay: warp-1\n        fallback_relays: []\n        affinity: none\n      - alias: coding-default\n"
	a := readyAt(t, []string{edit, replacement}, func() time.Time { return now })
	if err := a.RecordUpstreamResponse("primary", "coding-default", 429, "120", "unknown"); err != nil {
		t.Fatal(err)
	}
	if !a.upstreamCoolingLocked("primary", "coding-default", now) || !a.upstreamCoolingLocked("primary", "coding-other", now) {
		t.Fatal("scope tidak diketahui tidak membatasi upstream-wide")
	}
	if got := a.lastUpstreamErrors[modelKey("primary", "coding-default")]; got != (upstreamError{Status: 429, Category: "rate_limit"}) {
		t.Fatalf("metadata upstream tidak teredaksi/terklasifikasi: %#v", got)
	}
	if a.View().Relays[0].CooldownActive || !a.View().Ready {
		t.Fatal("cooldown upstream mengubah status relay")
	}
}

func TestExpiredRetryAfterDoesNotActivateCooldown(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	a := readyAt(t, nil, func() time.Time { return now })
	if err := a.RecordUpstreamResponse("primary", "coding-default", 429, "Wed, 21 Oct 2015 07:28:00 GMT", ""); err != nil {
		t.Fatal(err)
	}
	if a.upstreamCoolingLocked("primary", "coding-default", now) || !a.upstreamCooldowns["primary"].IsZero() {
		t.Fatal("tanggal Retry-After kedaluwarsa membuat cooldown aktif")
	}
}

func TestUnrepresentableRetryAfterDoesNotActivateCooldown(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	a := readyAt(t, nil, func() time.Time { return now })
	if err := a.RecordUpstreamResponse("primary", "coding-default", 429, "9223372037", ""); err != nil {
		t.Fatal(err)
	}
	if _, valid, _ := retryAfterDeadline("9223372037", now); valid {
		t.Fatal("Retry-After yang tidak dapat direpresentasikan dianggap valid")
	}
	if a.upstreamCoolingLocked("primary", "coding-default", now) {
		t.Fatal("Retry-After di luar time.Duration membuat cooldown")
	}
}

func TestUpstreamErrorCategoriesAreFiniteAndRedacted(t *testing.T) {
	a := readyAt(t, nil, time.Now)
	for _, tc := range []struct {
		status   int
		category string
	}{
		{400, "request"}, {401, "authentication"}, {403, "access"},
		{404, "request"}, {422, "request"}, {500, "upstream"}, {599, "upstream"},
	} {
		if err := a.RecordUpstreamResponse("primary", "coding-default", tc.status, "private-marker", ""); err != nil {
			t.Fatal(err)
		}
		if got := a.lastUpstreamErrors[modelKey("primary", "coding-default")]; got != (upstreamError{Status: tc.status, Category: tc.category}) {
			t.Fatalf("status %d dicatat sebagai metadata yang salah: %#v", tc.status, got)
		}
	}
}
