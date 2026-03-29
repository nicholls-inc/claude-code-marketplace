package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// tempDirForRapid creates a temp directory that is cleaned up when the test
// completes. rapid.T does not have TempDir, so we use os.MkdirTemp and
// register cleanup via the outer *testing.T.
func tempDirForRapid(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "memory-prop-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// ---------- Round-trip property ----------

func TestPropertyWriteReadRoundTrip(t *testing.T) {
	dir := tempDirForRapid(t)
	rapid.Check(t, func(rt *rapid.T) {
		// Each iteration uses a unique sub-directory to avoid collisions.
		subDir := filepath.Join(dir, rapid.StringMatching(`[a-z]{10}`).Draw(rt, "subdir"))
		missionID := rapid.StringMatching(`[a-z][a-z0-9]{0,9}`).Draw(rt, "missionID")
		key := rapid.StringMatching(`[a-z][a-z0-9]{0,9}`).Draw(rt, "key")
		value := rapid.StringOf(rapid.RuneFrom([]rune{'a', 'b', 'c', ' ', '\n', '\t'})).Draw(rt, "value")
		if strings.TrimSpace(value) == "" {
			value = "default"
		}
		memType := rapid.SampledFrom([]MemoryType{Procedural, Semantic, Episodic}).Draw(rt, "type")

		s, err := NewStore(missionID, subDir)
		if err != nil {
			rt.Fatalf("new store: %v", err)
		}

		entry := Entry{
			Type:      memType,
			Key:       key,
			Value:     value,
			MissionID: missionID,
			Version:   1,
		}
		if err := s.Write(entry); err != nil {
			rt.Fatalf("write: %v", err)
		}

		got, err := s.Read(memType, key)
		if err != nil {
			rt.Fatalf("read: %v", err)
		}
		if got.Key != key {
			rt.Fatalf("key mismatch: got %q, want %q", got.Key, key)
		}
		if got.Value != SanitizeValue(value) {
			rt.Fatalf("value mismatch: got %q, want %q", got.Value, SanitizeValue(value))
		}
		if got.Type != memType {
			rt.Fatalf("type mismatch: got %q, want %q", got.Type, memType)
		}
	})
}

// ---------- Cross-mission isolation property ----------

func TestPropertyCrossMissionIsolation(t *testing.T) {
	dir := tempDirForRapid(t)
	rapid.Check(t, func(rt *rapid.T) {
		subDir := filepath.Join(dir, rapid.StringMatching(`[a-z]{10}`).Draw(rt, "subdir"))
		m1 := rapid.StringMatching(`m1[a-z]{0,5}`).Draw(rt, "mission1")
		m2 := rapid.StringMatching(`m2[a-z]{0,5}`).Draw(rt, "mission2")
		key := rapid.StringMatching(`[a-z]{1,8}`).Draw(rt, "key")
		memType := rapid.SampledFrom([]MemoryType{Procedural, Semantic, Episodic}).Draw(rt, "type")

		s1, _ := NewStore(m1, subDir)
		s2, _ := NewStore(m2, subDir)

		entry := Entry{
			Type:      memType,
			Key:       key,
			Value:     "secret",
			MissionID: m1,
			Version:   1,
		}
		if err := s1.Write(entry); err != nil {
			rt.Fatalf("write: %v", err)
		}

		// s2 must not be able to read s1's entry.
		_, err := s2.Read(memType, key)
		if err == nil {
			rt.Fatal("cross-mission read should have failed")
		}
	})
}

// ---------- Sanitization property ----------

func TestPropertySanitizationNoControlChars(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		input := rapid.String().Draw(rt, "input")
		result := SanitizeValue(input)

		for i, r := range result {
			if r == '\n' || r == '\t' {
				continue
			}
			if unicode.IsControl(r) {
				rt.Fatalf("control char %U at position %d in sanitized value", r, i)
			}
		}
		if len(result) > maxValueLen {
			rt.Fatalf("sanitized value too long: %d > %d", len(result), maxValueLen)
		}
	})
}

// ---------- KVStore operations property ----------

func TestPropertyKVStoreSetGet(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		kv := NewKVStore()

		keys := rapid.SliceOfN(rapid.StringMatching(`[a-z]{1,10}`), 1, 20).Draw(rt, "keys")
		values := rapid.SliceOfN(rapid.Int(), len(keys), len(keys)).Draw(rt, "values")

		for i, k := range keys {
			kv.Set(k, values[i])
		}

		// Last-write wins for each key.
		last := make(map[string]int)
		for i, k := range keys {
			last[k] = values[i]
		}

		for k, wantV := range last {
			got, ok := kv.Get(k)
			if !ok {
				rt.Fatalf("key %q missing", k)
			}
			if got.(int) != wantV {
				rt.Fatalf("Get(%q) = %v, want %v", k, got, wantV)
			}
		}

		// Delete all, confirm gone.
		for k := range last {
			kv.Delete(k)
		}
		for k := range last {
			_, ok := kv.Get(k)
			if ok {
				rt.Fatalf("key %q still present after delete", k)
			}
		}
	})
}

// ---------- Scratchpad promotion property ----------

func TestPropertyScratchpadPromotion(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		sp := NewScratchpad()

		n := rapid.IntRange(1, 20).Draw(rt, "n")
		keys := make([]string, n)
		for i := 0; i < n; i++ {
			keys[i] = rapid.StringMatching(`[a-z]{1,8}`).Draw(rt, "key")
			sp.Set(keys[i], keys[i]+"-val")
		}

		// Promote a random subset.
		promoteSet := make(map[string]bool)
		for _, k := range keys {
			if rapid.Bool().Draw(rt, "promote-"+k) {
				_ = sp.Promote(k)
				promoteSet[k] = true
			}
		}

		promoted := sp.PromotedEntries()
		// Every promoted key must appear.
		for k := range promoteSet {
			v, ok := promoted[k]
			if !ok {
				rt.Fatalf("promoted key %q missing from PromotedEntries", k)
			}
			if v != k+"-val" {
				rt.Fatalf("promoted value for %q = %q, want %q", k, v, k+"-val")
			}
		}
		// No non-promoted key should appear.
		for k := range promoted {
			if !promoteSet[k] {
				rt.Fatalf("non-promoted key %q appeared in PromotedEntries", k)
			}
		}
	})
}

// ---------- Progress round-trip property ----------

func TestPropProgressRoundTrip(t *testing.T) {
	dir := tempDirForRapid(t)
	rapid.Check(t, func(rt *rapid.T) {
		subDir := filepath.Join(dir, rapid.StringMatching(`[a-z]{10}`).Draw(rt, "subdir"))
		missionID := rapid.StringMatching(`[a-z][a-z0-9]{0,9}`).Draw(rt, "missionID")
		nTasks := rapid.IntRange(0, 10).Draw(rt, "nTasks")
		tasks := make([]string, nTasks)
		for i := 0; i < nTasks; i++ {
			tasks[i] = rapid.StringMatching(`[a-z]{1,12}`).Draw(rt, "task")
		}

		created, err := CreateProgress(missionID, tasks, subDir)
		if err != nil {
			rt.Fatalf("create progress: %v", err)
		}

		loaded, err := LoadProgress(missionID, subDir)
		if err != nil {
			rt.Fatalf("load progress: %v", err)
		}

		if loaded.MissionID != created.MissionID {
			rt.Fatalf("mission ID mismatch: got %q, want %q", loaded.MissionID, created.MissionID)
		}
		if len(loaded.Items) != len(created.Items) {
			rt.Fatalf("items count mismatch: got %d, want %d", len(loaded.Items), len(created.Items))
		}
		for i := range loaded.Items {
			if loaded.Items[i].Task != created.Items[i].Task {
				rt.Fatalf("item %d task mismatch: got %q, want %q", i, loaded.Items[i].Task, created.Items[i].Task)
			}
			if loaded.Items[i].Status != "pending" {
				rt.Fatalf("item %d status = %q, want %q", i, loaded.Items[i].Status, "pending")
			}
		}
	})
}

// ---------- StartSession never panics property ----------

func TestPropStartSessionNeverPanics(t *testing.T) {
	dir := tempDirForRapid(t)
	rapid.Check(t, func(rt *rapid.T) {
		subDir := filepath.Join(dir, rapid.StringMatching(`[a-z]{10}`).Draw(rt, "subdir"))
		missionID := rapid.StringMatching(`[a-z][a-z0-9]{0,9}`).Draw(rt, "missionID")
		sessionID := rapid.StringMatching(`[a-z][a-z0-9]{0,9}`).Draw(rt, "sessionID")

		// Must not panic regardless of whether files exist.
		ctx, err := StartSession(missionID, sessionID, subDir)
		if err != nil {
			rt.Fatalf("start session returned error for missing files: %v", err)
		}
		if ctx == nil {
			rt.Fatal("start session returned nil context")
		}
	})
}

// ---------- Validate rejects empty key or value ----------

func TestPropertyValidateRejectsEmptyKeyOrValue(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		key := rapid.StringMatching(`\s*`).Draw(rt, "key")     // whitespace-only or empty
		value := rapid.StringMatching(`\s*`).Draw(rt, "value") // whitespace-only or empty

		entry := Entry{
			Type:      Procedural,
			Key:       key,
			Value:     value,
			MissionID: "m1",
		}
		vr := ValidateEntry(entry)
		if vr.Valid {
			rt.Fatal("expected validation failure for empty/whitespace key+value")
		}
	})
}
