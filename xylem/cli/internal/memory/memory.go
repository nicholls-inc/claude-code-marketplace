// Package memory provides mission-scoped, typed memory storage for agent
// sessions. It supports three memory types (procedural, semantic, episodic),
// structured handoff artifacts, ephemeral scratchpads, and a session-level KV
// store.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// MemoryType classifies memory entries into one of three categories.
type MemoryType string

const (
	// Procedural memory stores rules and conventions.
	Procedural MemoryType = "procedural"
	// Semantic memory stores learned facts and knowledge.
	Semantic MemoryType = "semantic"
	// Episodic memory stores examples and past patterns.
	Episodic MemoryType = "episodic"
)

// validMemoryTypes enumerates the accepted MemoryType values.
var validMemoryTypes = map[MemoryType]bool{
	Procedural: true,
	Semantic:   true,
	Episodic:   true,
}

// maxValueLen is the maximum byte length for a sanitized value.
const maxValueLen = 1 << 20 // 1 MiB

// Entry is a single memory record stored on disk.
type Entry struct {
	Type      MemoryType `json:"type"`
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	MissionID string     `json:"mission_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int        `json:"version"`
	Tags      []string   `json:"tags,omitempty"`
}

// ValidationResult reports whether an Entry passes validation checks.
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ValidateEntry checks an Entry for structural correctness.
func ValidateEntry(entry Entry) ValidationResult {
	var errs []string
	if strings.TrimSpace(entry.Key) == "" {
		errs = append(errs, "key must not be empty")
	}
	if strings.TrimSpace(entry.Value) == "" {
		errs = append(errs, "value must not be empty")
	}
	if !validMemoryTypes[entry.Type] {
		errs = append(errs, fmt.Sprintf("invalid memory type: %q", entry.Type))
	}
	if strings.TrimSpace(entry.MissionID) == "" {
		errs = append(errs, "mission_id must not be empty")
	}
	return ValidationResult{
		Valid:  len(errs) == 0,
		Errors: errs,
	}
}

// SanitizeValue strips control characters (except \n and \t) from value and
// truncates to maxValueLen bytes.
func SanitizeValue(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	s := b.String()
	if len(s) > maxValueLen {
		s = s[:maxValueLen]
	}
	return s
}

// Store provides mission-scoped, filesystem-backed memory storage.
type Store struct {
	missionID string
	basePath  string
}

// NewStore creates a Store rooted at basePath for the given mission. It
// returns an error if missionID is empty.
func NewStore(missionID string, basePath string) (*Store, error) {
	if strings.TrimSpace(missionID) == "" {
		return nil, fmt.Errorf("new store: mission ID must not be empty")
	}
	if strings.TrimSpace(basePath) == "" {
		return nil, fmt.Errorf("new store: base path must not be empty")
	}
	return &Store{missionID: missionID, basePath: basePath}, nil
}

// entryPath returns the filesystem path for an entry.
func (s *Store) entryPath(memType MemoryType, key string) string {
	return filepath.Join(s.basePath, s.missionID, string(memType), key+".json")
}

// typeDir returns the directory for a given memory type under this mission.
func (s *Store) typeDir(memType MemoryType) string {
	return filepath.Join(s.basePath, s.missionID, string(memType))
}

// Write validates, sanitizes, and persists an Entry to disk. The entry's
// MissionID must match the store's mission.
func (s *Store) Write(entry Entry) error {
	if entry.MissionID != s.missionID {
		return fmt.Errorf("write: entry mission %q does not match store mission %q", entry.MissionID, s.missionID)
	}

	entry.Value = SanitizeValue(entry.Value)

	vr := ValidateEntry(entry)
	if !vr.Valid {
		return fmt.Errorf("write: validation failed: %s", strings.Join(vr.Errors, "; "))
	}

	dir := s.typeDir(entry.Type)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("write: create dir: %w", err)
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("write: marshal: %w", err)
	}

	path := s.entryPath(entry.Type, entry.Key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write: save file: %w", err)
	}
	return nil
}

// Read loads a single entry from disk. It enforces mission isolation — the
// entry's MissionID must match the store's.
func (s *Store) Read(memType MemoryType, key string) (*Entry, error) {
	if !validMemoryTypes[memType] {
		return nil, fmt.Errorf("read: invalid memory type: %q", memType)
	}

	path := s.entryPath(memType, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("read: unmarshal: %w", err)
	}

	if entry.MissionID != s.missionID {
		return nil, fmt.Errorf("read: cross-mission access denied (store=%q, entry=%q)", s.missionID, entry.MissionID)
	}
	return &entry, nil
}

// List returns all entries of the given type for this mission, sorted by key.
func (s *Store) List(memType MemoryType) ([]Entry, error) {
	if !validMemoryTypes[memType] {
		return nil, fmt.Errorf("list: invalid memory type: %q", memType)
	}

	dir := s.typeDir(memType)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list: read dir: %w", err)
	}

	var entries []Entry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("list: read file %s: %w", f.Name(), err)
		}
		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("list: unmarshal %s: %w", f.Name(), err)
		}
		if e.MissionID != s.missionID {
			continue // skip cross-mission entries
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries, nil
}

// Delete removes an entry from disk.
func (s *Store) Delete(memType MemoryType, key string) error {
	if !validMemoryTypes[memType] {
		return fmt.Errorf("delete: invalid memory type: %q", memType)
	}

	path := s.entryPath(memType, key)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// HandoffArtifact captures session outcome for structured handoff between
// sessions.
type HandoffArtifact struct {
	MissionID  string   `json:"mission_id"`
	SessionID  string   `json:"session_id"`
	Completed  []string `json:"completed,omitempty"`
	Failed     []string `json:"failed,omitempty"`
	Unresolved []string `json:"unresolved,omitempty"`
	NextSteps  []string `json:"next_steps,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewHandoff creates a HandoffArtifact stamped with the current time.
func NewHandoff(missionID, sessionID string) *HandoffArtifact {
	return &HandoffArtifact{
		MissionID: missionID,
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
}

// handoffFileName returns the deterministic filename for a handoff artifact.
func handoffFileName(missionID, sessionID string) string {
	return fmt.Sprintf("handoff_%s_%s.json", missionID, sessionID)
}

// Save persists the handoff artifact to dir.
func (h *HandoffArtifact) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save handoff: create dir: %w", err)
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("save handoff: marshal: %w", err)
	}
	path := filepath.Join(dir, handoffFileName(h.MissionID, h.SessionID))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save handoff: write: %w", err)
	}
	return nil
}

// LoadHandoff reads a handoff artifact from dir.
func LoadHandoff(missionID, sessionID, dir string) (*HandoffArtifact, error) {
	path := filepath.Join(dir, handoffFileName(missionID, sessionID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load handoff: %w", err)
	}
	var h HandoffArtifact
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("load handoff: unmarshal: %w", err)
	}
	return &h, nil
}

// Scratchpad provides ephemeral key-value notes with promotion support.
type Scratchpad struct {
	mu       sync.RWMutex
	entries  map[string]string
	promoted map[string]bool
}

// NewScratchpad creates an empty Scratchpad.
func NewScratchpad() *Scratchpad {
	return &Scratchpad{
		entries:  make(map[string]string),
		promoted: make(map[string]bool),
	}
}

// Set writes a key-value pair to the scratchpad.
func (sp *Scratchpad) Set(key, value string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.entries[key] = value
}

// Get returns the value for key and whether it exists.
func (sp *Scratchpad) Get(key string) (string, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	v, ok := sp.entries[key]
	return v, ok
}

// Promote marks a scratchpad entry as promoted. Returns an error if the key
// does not exist.
func (sp *Scratchpad) Promote(key string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if _, ok := sp.entries[key]; !ok {
		return fmt.Errorf("promote: key %q not found", key)
	}
	sp.promoted[key] = true
	return nil
}

// PromotedEntries returns all promoted key-value pairs.
func (sp *Scratchpad) PromotedEntries() map[string]string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	result := make(map[string]string)
	for k := range sp.promoted {
		if v, ok := sp.entries[k]; ok {
			result[k] = v
		}
	}
	return result
}

// KVStore is a goroutine-safe, session-level key-value store.
type KVStore struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewKVStore creates an empty KVStore.
func NewKVStore() *KVStore {
	return &KVStore{data: make(map[string]any)}
}

// Set stores a value under the given key.
func (kv *KVStore) Set(key string, value any) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.data[key] = value
}

// Get retrieves a value by key. The second return value reports existence.
func (kv *KVStore) Get(key string) (any, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	v, ok := kv.data[key]
	return v, ok
}

// Delete removes a key from the store.
func (kv *KVStore) Delete(key string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.data, key)
}

// Keys returns all keys in sorted order.
func (kv *KVStore) Keys() []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	keys := make([]string, 0, len(kv.data))
	for k := range kv.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
