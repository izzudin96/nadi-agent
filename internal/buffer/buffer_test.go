package buffer

import (
	"path/filepath"
	"testing"
)

func testPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "buffer.jsonl")
}

// newTestBuffer builds a Buffer with a tiny byte cap so drop-oldest behavior is
// easy to exercise without huge payloads.
func newTestBuffer(t *testing.T, maxBytes int64) *Buffer {
	t.Helper()
	b, err := New(testPath(t), 1) // 1MB; overridden below
	if err != nil {
		t.Fatal(err)
	}
	b.maxBytes = maxBytes
	return b
}

func TestAppendAndPeekPop(t *testing.T) {
	b := newTestBuffer(t, 1024)

	if err := b.Append([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := b.Append([]byte(`{"a":2}`)); err != nil {
		t.Fatal(err)
	}

	if got := b.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}

	// Oldest first.
	peek, ok := b.Peek()
	if !ok || string(peek) != `{"a":1}` {
		t.Fatalf("Peek() = %q, %v; want first entry", peek, ok)
	}
	popped, ok := b.Pop()
	if !ok || string(popped) != `{"a":1}` {
		t.Fatalf("Pop() = %q, %v; want first entry", popped, ok)
	}
	if got := b.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
}

func TestPersistSurvivesReload(t *testing.T) {
	path := testPath(t)
	b, err := New(path, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{`{"i":0}`, `{"i":1}`, `{"i":2}`} {
		if err := b.Append([]byte(payload)); err != nil {
			t.Fatal(err)
		}
	}

	// Reload from disk.
	b2, err := New(path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := b2.Len(); got != 3 {
		t.Fatalf("reloaded Len() = %d, want 3", got)
	}
	peek, _ := b2.Peek()
	if string(peek) != `{"i":0}` {
		t.Fatalf("reloaded oldest = %q, want {\"i\":0}", peek)
	}
}

func TestCapDropsOldest(t *testing.T) {
	// Each entry ~7 bytes + newline = 8. Cap of 20 bytes holds 2 entries (16B);
	// the third entry pushes it over, dropping the oldest.
	b := newTestBuffer(t, 20)

	for _, payload := range []string{`{"n":1}`, `{"n":2}`, `{"n":3}`} {
		if err := b.Append([]byte(payload)); err != nil {
			t.Fatal(err)
		}
	}

	if got := b.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2 (oldest dropped)", got)
	}
	peek, _ := b.Peek()
	if string(peek) != `{"n":2}` {
		t.Fatalf("oldest after trim = %q, want {\"n\":2}", peek)
	}

	// Verify the file matches memory after reload.
	b2, err := New(b.path, 1)
	if err != nil {
		t.Fatal(err)
	}
	b2.maxBytes = 20
	if got := b2.Len(); got != 2 {
		t.Fatalf("reloaded Len() = %d, want 2", got)
	}
}

func TestOversizedPayloadDropped(t *testing.T) {
	b := newTestBuffer(t, 8) // cap 8 bytes; entry must be < cap or it's dropped

	big := make([]byte, 100)
	for i := range big {
		big[i] = 'x'
	}
	if err := b.Append(big); err != nil {
		t.Fatal(err)
	}
	if got := b.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0 (oversized entry dropped, no error)", got)
	}
}

func TestEmptyBufferPeekPop(t *testing.T) {
	b := newTestBuffer(t, 1024)
	if _, ok := b.Peek(); ok {
		t.Fatal("Peek() on empty buffer should return false")
	}
	if _, ok := b.Pop(); ok {
		t.Fatal("Pop() on empty buffer should return false")
	}
}
