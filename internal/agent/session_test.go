package agent

import (
	"context"
	"testing"
	"time"
)

func TestIdleExpiryAndApproval(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	closed := 0
	cache := NewSessions(func() time.Time { return now })
	cache.Put("account", func(context.Context) { closed++ })
	now = now.Add(59 * time.Minute)
	if !cache.Touch("account") {
		t.Fatal("session expired early")
	}
	now = now.Add(59 * time.Minute)
	cache.Expire(context.Background())
	if closed != 0 {
		t.Fatal("approved use did not extend session")
	}
	now = now.Add(time.Minute)
	cache.Expire(context.Background())
	if closed != 1 || cache.Touch("account") {
		t.Fatal("idle session was retained")
	}
}

func TestStatusDoesNotExtendExpiry(t *testing.T) {
	now := time.Now()
	cache := NewSessions(func() time.Time { return now })
	cache.Put("a", func(context.Context) {})
	now = now.Add(59 * time.Minute)
	if len(cache.Status()) != 1 {
		t.Fatal("missing session")
	}
	now = now.Add(time.Minute)
	cache.Expire(context.Background())
	if len(cache.Status()) != 0 {
		t.Fatal("status refreshed session")
	}
}
