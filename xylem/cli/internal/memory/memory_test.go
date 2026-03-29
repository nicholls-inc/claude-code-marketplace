package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
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

func TestSanitizeValueUTF8Boundary(t *testing.T) {
	// Build a string of multi-byte runes that, when byte-sliced at maxValueLen,
	// would split a rune. U+1F600 (😀) is 4 bytes in UTF-8.
	emoji := "😀" // 4 bytes
	count := (maxValueLen / len(emoji)) + 1
	input := strings.Repeat(emoji, count)

	got := SanitizeValue(input)

	if !utf8.ValidString(got) {
		t.Fatal("truncated value is not valid UTF-8")
	}
	if len(got) > maxValueLen {
		t.Fatalf("truncated value exceeds maxValueLen: %d > %d", len(got), maxValueLen)
	}
	// The result should be the largest multiple of 4 that fits in maxValueLen.
	expectedLen := (maxValueLen / len(emoji)) * len(emoji)
	if len(got) != expectedLen {
		t.Fatalf("expected len %d, got %d", expectedLen, len(got))
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

// ---------- Path Traversal ----------

func TestStorePathTraversal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	tests := []struct {
		name string
		key  string
	}{
		{"dot-dot escape", "../escape"},
		{"slash in key", "sub/dir"},
		{"backslash in key", "sub\\dir"},
		{"dot-dot only", ".."},
		{"complex traversal", "../../etc/passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := makeEntry("m1", tt.key, "malicious", Procedural)
			err := s.Write(e)
			if err == nil {
				t.Fatalf("expected error for key %q, got nil", tt.key)
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("expected path validation error, got: %v", err)
			}
		})
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
	if len(loaded.Unresolved) != 1 || loaded.Unresolved[0] != "task-c" {
		t.Fatalf("unresolved mismatch: %v", loaded.Unresolved)
	}
	if loaded.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
	if delta := time.Since(loaded.CreatedAt); delta > 5*time.Second {
		t.Fatalf("CreatedAt too far in the past: %v", delta)
	}
}

func TestHandoffLoadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadHandoff("m1", "nope", dir)
	if err == nil {
		t.Fatal("expected error loading missing handoff")
	}
}

// ---------- Progress ----------

func TestCreateProgressValid(t *testing.T) {
	dir := t.TempDir()
	tasks := []string{"build", "test", "deploy"}
	pf, err := CreateProgress("m1", tasks, dir)
	if err != nil {
		t.Fatalf("create progress: %v", err)
	}
	if pf.MissionID != "m1" {
		t.Fatalf("mission ID = %q, want %q", pf.MissionID, "m1")
	}
	if len(pf.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(pf.Items))
	}
	for i, item := range pf.Items {
		if item.Task != tasks[i] {
			t.Fatalf("item %d task = %q, want %q", i, item.Task, tasks[i])
		}
		if item.Status != "pending" {
			t.Fatalf("item %d status = %q, want %q", i, item.Status, "pending")
		}
		if item.StartedAt != nil {
			t.Fatalf("item %d StartedAt should be nil", i)
		}
		if item.CompletedAt != nil {
			t.Fatalf("item %d CompletedAt should be nil", i)
		}
	}
	if pf.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero UpdatedAt")
	}

	// Verify file exists and contains valid JSON.
	path := filepath.Join(dir, "progress_m1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("progress file is empty")
	}
}

func TestCreateProgressInvalidMissionID(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name      string
		missionID string
	}{
		{"path traversal", "../escape"},
		{"empty", ""},
		{"slash", "a/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateProgress(tt.missionID, []string{"task"}, dir)
			if err == nil {
				t.Fatalf("expected error for mission ID %q", tt.missionID)
			}
		})
	}
}

func TestLoadProgressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tasks := []string{"alpha", "beta"}
	created, err := CreateProgress("m1", tasks, dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	loaded, err := LoadProgress("m1", dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.MissionID != created.MissionID {
		t.Fatalf("mission ID mismatch: got %q, want %q", loaded.MissionID, created.MissionID)
	}
	if len(loaded.Items) != len(created.Items) {
		t.Fatalf("items count mismatch: got %d, want %d", len(loaded.Items), len(created.Items))
	}
	for i := range loaded.Items {
		if loaded.Items[i].Task != created.Items[i].Task {
			t.Fatalf("item %d task mismatch: got %q, want %q", i, loaded.Items[i].Task, created.Items[i].Task)
		}
		if loaded.Items[i].Status != created.Items[i].Status {
			t.Fatalf("item %d status mismatch: got %q, want %q", i, loaded.Items[i].Status, created.Items[i].Status)
		}
	}
}

func TestUpdateProgressStatus(t *testing.T) {
	dir := t.TempDir()
	_, err := CreateProgress("m1", []string{"build", "test"}, dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := UpdateProgress("m1", "build", "in_progress", dir); err != nil {
		t.Fatalf("update: %v", err)
	}

	loaded, err := LoadProgress("m1", dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Items[0].Status != "in_progress" {
		t.Fatalf("status = %q, want %q", loaded.Items[0].Status, "in_progress")
	}
	if loaded.Items[0].StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
	if loaded.Items[0].CompletedAt != nil {
		t.Fatal("expected CompletedAt to be nil")
	}
	// Second task should be unchanged.
	if loaded.Items[1].Status != "pending" {
		t.Fatalf("second task status = %q, want %q", loaded.Items[1].Status, "pending")
	}
}

func TestUpdateProgressCompletion(t *testing.T) {
	dir := t.TempDir()
	_, err := CreateProgress("m1", []string{"build"}, dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := UpdateProgress("m1", "build", "in_progress", dir); err != nil {
		t.Fatalf("update to in_progress: %v", err)
	}
	if err := UpdateProgress("m1", "build", "completed", dir); err != nil {
		t.Fatalf("update to completed: %v", err)
	}

	loaded, err := LoadProgress("m1", dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Items[0].Status != "completed" {
		t.Fatalf("status = %q, want %q", loaded.Items[0].Status, "completed")
	}
	if loaded.Items[0].StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
	if loaded.Items[0].CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestUpdateProgressUnknownTask(t *testing.T) {
	dir := t.TempDir()
	_, err := CreateProgress("m1", []string{"build"}, dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = UpdateProgress("m1", "nonexistent", "in_progress", dir)
	if err == nil {
		t.Fatal("expected error for unknown task")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got: %v", err)
	}
}

func TestStartSessionBothExist(t *testing.T) {
	dir := t.TempDir()

	// Create handoff.
	h := NewHandoff("m1", "s1")
	h.Completed = []string{"task-a"}
	if err := h.Save(dir); err != nil {
		t.Fatalf("save handoff: %v", err)
	}

	// Create progress.
	_, err := CreateProgress("m1", []string{"build", "test"}, dir)
	if err != nil {
		t.Fatalf("create progress: %v", err)
	}

	ctx, err := StartSession("m1", "s1", dir)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	if ctx.Handoff == nil {
		t.Fatal("expected Handoff to be non-nil")
	}
	if ctx.Handoff.MissionID != "m1" {
		t.Fatalf("handoff mission = %q, want %q", ctx.Handoff.MissionID, "m1")
	}
	if ctx.Progress == nil {
		t.Fatal("expected Progress to be non-nil")
	}
	if ctx.Progress.MissionID != "m1" {
		t.Fatalf("progress mission = %q, want %q", ctx.Progress.MissionID, "m1")
	}
	if len(ctx.Progress.Items) != 2 {
		t.Fatalf("progress items = %d, want 2", len(ctx.Progress.Items))
	}
}

func TestStartSessionMissingFiles(t *testing.T) {
	dir := t.TempDir()

	ctx, err := StartSession("m1", "s1", dir)
	if err != nil {
		t.Fatalf("expected no error for missing files, got: %v", err)
	}
	if ctx.Handoff != nil {
		t.Fatal("expected Handoff to be nil")
	}
	if ctx.Progress != nil {
		t.Fatal("expected Progress to be nil")
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

// ---------- Semantic Validation ----------

func TestSemanticContradictionDetected(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	s.SetValidator(DefaultSemanticValidator())

	e1 := makeEntry("m1", "go-version", "1.21", Semantic)
	if err := s.Write(e1); err != nil {
		t.Fatalf("write first: %v", err)
	}

	e2 := makeEntry("m1", "go-version", "1.22", Semantic)
	existing, err := s.List(Semantic)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	result := s.validator.Validate(e2, existing)

	found := false
	for _, c := range result.Checks {
		if c.Check == "contradiction" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected contradiction warning when value changes for same key")
	}
}

func TestSemanticNoContradictionOnSameValue(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	s.SetValidator(DefaultSemanticValidator())

	e := makeEntry("m1", "go-version", "1.21", Semantic)
	if err := s.Write(e); err != nil {
		t.Fatalf("write: %v", err)
	}

	existing, err := s.List(Semantic)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	result := s.validator.Validate(e, existing)

	for _, c := range result.Checks {
		if c.Check == "contradiction" {
			t.Fatal("unexpected contradiction for same key+value")
		}
	}
}

func TestSemanticHallucinationRepeatedChars(t *testing.T) {
	v := DefaultSemanticValidator()
	e := Entry{Type: Semantic, Key: "test", Value: "aaaaaaaaa", MissionID: "m1"}
	result := v.Validate(e, nil)

	found := false
	for _, c := range result.Checks {
		if c.Check == "hallucination" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected hallucination warning for repeated characters")
	}
}

func TestSemanticHallucinationPlaceholder(t *testing.T) {
	v := DefaultSemanticValidator()
	for _, placeholder := range []string{"TBD", "TODO", "N/A", "tbd", "n/a"} {
		e := Entry{Type: Semantic, Key: "test", Value: placeholder, MissionID: "m1"}
		result := v.Validate(e, nil)

		found := false
		for _, c := range result.Checks {
			if c.Check == "hallucination" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected hallucination warning for placeholder %q", placeholder)
		}
	}
}

func TestSemanticHallucinationCleanValue(t *testing.T) {
	v := DefaultSemanticValidator()
	e := Entry{Type: Semantic, Key: "test", Value: "Go is a statically typed language", MissionID: "m1"}
	result := v.Validate(e, nil)

	for _, c := range result.Checks {
		if c.Check == "hallucination" {
			t.Fatalf("unexpected hallucination warning for clean value: %s", c.Message)
		}
	}
}

func TestSemanticDuplicationDetected(t *testing.T) {
	v := DefaultSemanticValidator()
	existing := []Entry{
		{Type: Semantic, Key: "lang", Value: "Go is great", MissionID: "m1"},
	}
	e := Entry{Type: Semantic, Key: "language", Value: "Go is great", MissionID: "m1"}
	result := v.Validate(e, existing)

	found := false
	for _, c := range result.Checks {
		if c.Check == "duplication" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected duplication warning for identical value under different key")
	}
}

func TestSemanticValidatorNilSkipsChecks(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	// Deliberately do NOT call SetValidator — validator is nil.

	// Write an entry with a placeholder value that would trigger hallucination
	// if the validator were active.
	e := makeEntry("m1", "placeholder-key", "TBD", Semantic)
	if err := s.Write(e); err != nil {
		t.Fatalf("write should succeed with nil validator: %v", err)
	}
}

func TestStoreWriteSemanticError(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("m1", dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	// Configure a validator with MaxKeyReuse=1 so the second write triggers an error.
	v := DefaultSemanticValidator()
	v.MaxKeyReuse = 1
	s.SetValidator(v)

	e := makeEntry("m1", "reused-key", "first value", Semantic)
	if err := s.Write(e); err != nil {
		t.Fatalf("first write: %v", err)
	}

	e2 := makeEntry("m1", "reused-key", "second value", Semantic)
	err = s.Write(e2)
	if err == nil {
		t.Fatal("expected semantic error when MaxKeyReuse exceeded")
	}
	if !strings.Contains(err.Error(), "semantic validation failed") {
		t.Fatalf("expected semantic validation error, got: %v", err)
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

	// All keys must have been deleted.
	if remaining := len(kv.Keys()); remaining != 0 {
		t.Fatalf("expected 0 keys after deletion, got %d", remaining)
	}
}

