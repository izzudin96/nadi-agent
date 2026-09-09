package backoff

import (
	"testing"
	"time"
)

func TestBackoffSequence(t *testing.T) {
	b := New(time.Second, 15*time.Second)

	want := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		15 * time.Second, // capped
		15 * time.Second, // stays capped
	}
	for i, w := range want {
		if got := b.Next(); got != w {
			t.Fatalf("Next() #%d = %v, want %v", i, got, w)
		}
	}
}

func TestBackoffReset(t *testing.T) {
	b := New(time.Second, 10*time.Second)
	_ = b.Next() // 1s
	_ = b.Next() // 2s
	b.Reset()
	if got := b.Current(); got != time.Second {
		t.Fatalf("Current() after Reset = %v, want 1s", got)
	}
}

func TestBackoffMaxBelowMin(t *testing.T) {
	b := New(10*time.Second, time.Second) // max < min
	if got := b.Next(); got != 10*time.Second {
		t.Fatalf("Next() = %v, want 10s (max clamped to min)", got)
	}
}
