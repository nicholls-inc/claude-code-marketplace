package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---------- NewStore ----------

func TestNewStore(t *testing.T) {
	tests := []struct {
		name      string
		missionID string
		basePath  string
		wantErr   bool
	}{
		{"valid", "m-1", t.TempDir(), false},
		{"empty mission", "", t.TempDir(), true},
		{"whitespace mission", "  ", t.TempDir(), true},
		{"empty base path", "m-1", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewStore(tt.missionID, tt.basePath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s == nil {
				t.Fatal("expected non-nil store")
			}
		})
	}
}

// ---------- ValidateEntry ----------

func TestValidateEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   Entry
		wantOK  bool
		wantN   int // expected number of errors
	}{
		{
			name:   "valid procedural",
			entry:  Entry{Type: Procedural, Key: "k", Value: "v", MissionID: "m1"},
			wantOK: true,
		},
		{
			name:   "valid semantic",
			entry:  Entry{Type: Semantic, Key: "k", Value: "v", MissionID: "m1"},
			wantOK: true,
		},
		{
			name:   "valid episodic",
			entry:  Entry{Type: Episodic, Key: "k", Value: "v", MissionID: "m1"},
			wantOK: true,
		},
		{
			name:   "empty key",
			entry:  Entry{Type: Procedural, Key: "", Value: "v", MissionID: "m1"},
			wantOK: false, wantN: 1,
		},
		{
			name:   "empty value",
			entry:  Entry{Type: Procedural, Key: "k", Value: "", MissionID: "m1"},
			wantOK: false, wantN: 1,
		},
		{
			name:   "empty key and value",
			entry:  Entry{Type: Procedural, Key: "", Value: "", MissionID: "m1"},
			wantOK: false, wantN: 2,
		},
		{
			name:   "invalid type",
			entry:  Entry{Type: "unknown", Key: "k", Value: "v", MissionID: "m1"},
			wantOK: false, wantN: 1,
		},
		{
			name:   "empty mission",
			entry:  Entry{Type: Procedural, Key: "k", Value: "v", MissionID: ""},
			wantOK: false, wantN: 1,
		},
		{
			name:   "all invalid",
			entry:  Entry{Type: "bad", Key: "", Value: "", MissionID: ""},
			wantOK: false, wantN: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := ValidateEntry(tt.entry)
			if vr.Valid != tt.wantOK {
				t.Fatalf("Valid = %v, want %v; errors = %v", vr.Valid, tt.wantOK, vr.Errors)
			}
			if !tt.wantOK && len(vr.Errors) != tt.wantN {
				t.Fatalf("got %d errors, want %d: %v", len(vr.Errors), tt.wantN, vr.Errors)
			}
		})
	}
}

// ---------- SanitizeValue ----------

func TestSanitizeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"preserves newline", "a\nb", "a\nb"},
		{"preserves tab", "a\tb", "a\tb"},
		{"strips null", "a\x00b", "ab"},
		{"strips bell", "a\x07b", "ab"},
		{"strips escape", "a\x1bb", "ab"},
		{"strips carriage return", "a\rb", "ab"},
		{"mixed controls", "a\x00\n\x07\tb\x1b", "a\n\tb"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeValue(tt.input)
			if got != tt.want {
				t.Fatalf("SanitizeValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeValueTruncation(t *testing.T) {
	buf := make([]byte, maxValueLen+100)
	for i := range buf {
		buf[i] = 'a'
	}
	got := SanitizeValue(string(buf))
	if len(got) != maxValueLen {
		t.Fatalf("len = %d, want %d", len(got), maxValueLen)
	}
}

// ---------- Store CRUD ----------

func makeEntry(missionID, key, value string, memType MemoryType) Entry {
	now := time.Now()
	return Entry{
		Type:      memType,
		Key:       key,
		Value:     value,
		MissionID: missionID,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

func TestStoreWriteRead(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	tests := []struct {
		name    string
		memType MemoryType
		key     string
		value   string
	}{
		{"procedural", Procedural, "rule1", "always test"},
		{"semantic", Semantic, "fact1", "Go is compiled"},
		{"episodic", Episodic, "pattern1", "retry on timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := makeEntry("m1", tt.key, tt.value, tt.memType)
			if err := s.Write(e); err != nil {
				t.Fatalf("write: %v", err)
			}
			got, err := s.Read(tt.memType, tt.key)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if got.Key != tt.key || got.Value != tt.value || got.Type != tt.memType {
				t.Fatalf("round-trip mismatch: got %+v", got)
			}
		})
	}
}

func TestStoreWriteMissionMismatch(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	e := makeEntry("m-other", "k", "v", Procedural)
	if err := s.Write(e); err == nil {
		t.Fatal("expected error for mission mismatch write")
	}
}

func TestStoreWriteValidationFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	e := Entry{Type: Procedural, Key: "", Value: "v", MissionID: "m1"}
	if err := s.Write(e); err == nil {
		t.Fatal("expected validation error for empty key")
	}
}

func TestStoreMissionIsolation(t *testing.T) {
	dir := t.TempDir()
	s1, _ := NewStore("m1", dir)
	s2, _ := NewStore("m2", dir)

	e := makeEntry("m1", "secret", "classified", Procedural)
	if err := s1.Write(e); err != nil {
		t.Fatalf("write: %v", err)
	}

	// s2 cannot read m1's entry via filesystem path manipulation — the path
	// is scoped to m2, so the file simply does not exist.
	_, err := s2.Read(Procedural, "secret")
	if err == nil {
		t.Fatal("expected error: cross-mission read should fail")
	}
}

func TestStoreMissionIsolationTamperedFile(t *testing.T) {
	dir := t.TempDir()
	s1, _ := NewStore("m1", dir)
	s2, _ := NewStore("m2", dir)

	// Write via s1.
	e := makeEntry("m1", "secret", "classified", Procedural)
	if err := s1.Write(e); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Manually copy the file into m2's directory tree to simulate tampering.
	srcPath := filepath.Join(dir, "m1", "procedural", "secret.json")
	dstDir := filepath.Join(dir, "m2", "procedural")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, _ := os.ReadFile(srcPath)
	if err := os.WriteFile(filepath.Join(dstDir, "secret.json"), data, 0o644); err != nil {
		t.Fatalf("copy: %v", err)
	}

	// s2 should reject the file because the entry's MissionID is m1, not m2.
	_, err := s2.Read(Procedural, "secret")
	if err == nil {
		t.Fatal("expected cross-mission access denied error")
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	for _, key := range []string{"c", "a", "b"} {
		e := makeEntry("m1", key, "val-"+key, Semantic)
		if err := s.Write(e); err != nil {
			t.Fatalf("write %s: %v", key, err)
		}
	}

	entries, err := s.List(Semantic)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	// Sorted by key.
	if entries[0].Key != "a" || entries[1].Key != "b" || entries[2].Key != "c" {
		t.Fatalf("unexpected order: %v %v %v", entries[0].Key, entries[1].Key, entries[2].Key)
	}
}

func TestStoreListEmpty(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	entries, err := s.List(Procedural)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestStoreListInvalidType(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	_, err := s.List("invalid")
	if err == nil {
		t.Fatal("expected error for invalid memory type")
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	e := makeEntry("m1", "to-delete", "temp", Episodic)
	if err := s.Write(e); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := s.Delete(Episodic, "to-delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := s.Read(Episodic, "to-delete")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestStoreDeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	err := s.Delete(Procedural, "nope")
	if err == nil {
		t.Fatal("expected error deleting non-existent key")
	}
}

func TestStoreDeleteInvalidType(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	err := s.Delete("bad", "k")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestStoreReadInvalidType(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore("m1", dir)

	_, err := s.Read("bad", "k")
	if err == nil {
		t.Fatal("expected error for invalid memory type")
	}
}

// ---------- Handoff ----------

func TestHandoffSaveLoad(t *testing.T) {
	dir := t.TempDir()
	h := NewHandoff("m1", "s1")
	h.Completed = []string{"task-a"}
	h.Failed = []string{"task-b"}
	h.Unresolved = []string{"task-c"}
	h.NextSteps = []string{"retry task-b"}

	if err := h.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadHandoff("m1", "s1", dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.MissionID != "m1" || loaded.SessionID != "s1" {
		t.Fatalf("id mismatch: %+v", loaded)
	}
	if len(loaded.Completed) != 1 || loaded.Completed[0] != "task-a" {
		t.Fatalf("completed mismatch: %v", loaded.Completed)
	}
	if len(loaded.Failed) != 1 || loaded.Failed[0] != "task-b" {
		t.Fatalf("failed mismatch: %v", loaded.Failed)
	}
	if len(loaded.NextSteps) != 1 || loaded.NextSteps[0] != "retry task-b" {
		t.Fatalf("next_steps mismatch: %v", loaded.NextSteps)
	}
}

func TestHandoffLoadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadHandoff("m1", "nope", dir)
	if err == nil {
		t.Fatal("expected error loading missing handoff")
	}
}

// ---------- Scratchpad ----------

func TestScratchpadSetGet(t *testing.T) {
	sp := NewScratchpad()
	sp.Set("k1", "v1")

	v, ok := sp.Get("k1")
	if !ok || v != "v1" {
		t.Fatalf("Get(k1) = (%q, %v), want (v1, true)", v, ok)
	}

	_, ok = sp.Get("missing")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestScratchpadPromote(t *testing.T) {
	sp := NewScratchpad()
	sp.Set("a", "1")
	sp.Set("b", "2")
	sp.Set("c", "3")

	if err := sp.Promote("a"); err != nil {
		t.Fatalf("promote a: %v", err)
	}
	if err := sp.Promote("c"); err != nil {
		t.Fatalf("promote c: %v", err)
	}

	promoted := sp.PromotedEntries()
	if len(promoted) != 2 {
		t.Fatalf("got %d promoted, want 2", len(promoted))
	}
	if promoted["a"] != "1" || promoted["c"] != "3" {
		t.Fatalf("unexpected promoted: %v", promoted)
	}
}

func TestScratchpadPromoteMissing(t *testing.T) {
	sp := NewScratchpad()
	if err := sp.Promote("nope"); err == nil {
		t.Fatal("expected error promoting non-existent key")
	}
}

func TestScratchpadOverwrite(t *testing.T) {
	sp := NewScratchpad()
	sp.Set("k", "old")
	sp.Set("k", "new")

	v, _ := sp.Get("k")
	if v != "new" {
		t.Fatalf("expected overwrite, got %q", v)
	}
}

// ---------- KVStore ----------

func TestKVStoreBasic(t *testing.T) {
	kv := NewKVStore()
	kv.Set("a", 1)
	kv.Set("b", "two")

	v, ok := kv.Get("a")
	if !ok || v.(int) != 1 {
		t.Fatalf("Get(a) = (%v, %v), want (1, true)", v, ok)
	}

	v, ok = kv.Get("b")
	if !ok || v.(string) != "two" {
		t.Fatalf("Get(b) = (%v, %v), want (two, true)", v, ok)
	}

	_, ok = kv.Get("missing")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestKVStoreDelete(t *testing.T) {
	kv := NewKVStore()
	kv.Set("k", "v")
	kv.Delete("k")

	_, ok := kv.Get("k")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestKVStoreKeys(t *testing.T) {
	kv := NewKVStore()
	kv.Set("c", 3)
	kv.Set("a", 1)
	kv.Set("b", 2)

	keys := kv.Keys()
	if len(keys) != 3 {
		t.Fatalf("got %d keys, want 3", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("keys not sorted: %v", keys)
	}
}

func TestKVStoreConcurrentAccess(t *testing.T) {
	kv := NewKVStore()
	var wg sync.WaitGroup
	n := 100

	// Concurrent writers.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			kv.Set(fmt.Sprintf("key-%d", i), i)
		}(i)
	}
	wg.Wait()

	// All keys present.
	keys := kv.Keys()
	if len(keys) != n {
		t.Fatalf("got %d keys, want %d", len(keys), n)
	}

	// Concurrent readers + deleters.
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			kv.Get(fmt.Sprintf("key-%d", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			kv.Delete(fmt.Sprintf("key-%d", i))
		}(i)
	}
	wg.Wait()
}

