package buffer

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// Buffer is a disk-backed queue of serialized heartbeat payloads, one JSON
// object per line (append-only JSONL). It is safe for concurrent use.
//
// The in-memory entries slice is the source of truth; the file persists it so
// unsent heartbeats survive a restart (at-least-once: on a crash after send,
// an entry may be re-sent).
type Buffer struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	entries  [][]byte // oldest first
	size     int64    // sum of len(entry)+1 (including the newline)
}

// New creates or loads a Buffer. maxSizeMB is the cap in megabytes.
func New(path string, maxSizeMB int) (*Buffer, error) {
	b := &Buffer{path: path, maxBytes: int64(maxSizeMB) * 1024 * 1024}
	if err := b.load(); err != nil {
		return nil, err
	}
	return b, nil
}

// load reads existing JSONL lines into memory on startup.
func (b *Buffer) load() error {
	f, err := os.Open(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh start
		}
		return fmt.Errorf("opening buffer file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // allow large lines
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		b.entries = append(b.entries, append([]byte(nil), line...))
		b.size += int64(len(line)) + 1
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading buffer file: %w", err)
	}

	if b.dropOldestToFit() {
		return b.writeAll()
	}
	return nil
}

// Append adds a payload to the buffer. If the payload alone exceeds the cap it
// is dropped silently — the buffer never errors just because it is "too full".
func (b *Buffer) Append(data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if int64(len(data))+1 > b.maxBytes {
		return nil // cannot ever fit; drop
	}

	f, err := os.OpenFile(b.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening buffer file for append: %w", err)
	}
	if _, err := f.Write(append(append([]byte(nil), data...), '\n')); err != nil {
		f.Close()
		return fmt.Errorf("appending to buffer file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing buffer file: %w", err)
	}

	b.entries = append(b.entries, append([]byte(nil), data...))
	b.size += int64(len(data)) + 1

	if b.dropOldestToFit() {
		return b.writeAll()
	}
	return nil
}

// Peek returns the oldest entry without removing it.
func (b *Buffer) Peek() ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) == 0 {
		return nil, false
	}
	return b.entries[0], true
}

// Pop removes and returns the oldest entry (after a successful send). The
// caller should Persist() after a batch of pops to compact the file.
func (b *Buffer) Pop() ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) == 0 {
		return nil, false
	}
	e := b.entries[0]
	b.entries = b.entries[1:]
	b.size -= int64(len(e)) + 1
	return e, true
}

// Persist rewrites the file from the in-memory entries, compacting out any
// entries already popped.
func (b *Buffer) Persist() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.writeAll()
}

// Len returns the number of buffered entries.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

// SizeBytes returns the approximate buffered size in bytes.
func (b *Buffer) SizeBytes() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

// dropOldestToFit removes oldest entries until size <= maxBytes. Returns true
// if anything was dropped. Caller must hold b.mu.
func (b *Buffer) dropOldestToFit() bool {
	dropped := false
	for b.size > b.maxBytes && len(b.entries) > 0 {
		b.size -= int64(len(b.entries[0])) + 1
		b.entries = b.entries[1:]
		dropped = true
	}
	return dropped
}

// writeAll writes all entries to a temp file then atomically renames it over
// the buffer file. Caller must hold b.mu.
func (b *Buffer) writeAll() error {
	tmp := b.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("creating buffer temp file: %w", err)
	}
	for _, e := range b.entries {
		if _, err := f.Write(append(append([]byte(nil), e...), '\n')); err != nil {
			f.Close()
			return fmt.Errorf("writing buffer temp file: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing buffer temp file: %w", err)
	}
	if err := os.Rename(tmp, b.path); err != nil {
		return fmt.Errorf("renaming buffer file: %w", err)
	}
	return nil
}
