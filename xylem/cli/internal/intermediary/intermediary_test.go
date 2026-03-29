package intermediary

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// mockExecutor records whether Execute was called and optionally returns an error.
type mockExecutor struct {
	mu        sync.Mutex
	callCount int
	err       error
}

func (m *mockExecutor) Execute(_ context.Context, _ Intent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.err
}

func (m *mockExecutor) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// newTestAuditLog creates an AuditLog in a temporary directory.
func newTestAuditLog(t *testing.T) *AuditLog {
	t.Helper()
	return NewAuditLog(filepath.Join(t.TempDir(), "audit.jsonl"))
}

func TestValidateIntent(t *testing.T) {
	tests := []struct {
		name    string
		intent  Intent
		wantErr error
	}{
		{
			name: "valid intent",
			intent: Intent{
				Action:        "file.write",
				Resource:      "/tmp/output.txt",
				AgentID:       "agent-1",
				Justification: "writing results",
			},
			wantErr: nil,
		},
		{
			name: "missing action",
			intent: Intent{
				Resource: "/tmp/output.txt",
				AgentID:  "agent-1",
			},
			wantErr: ErrEmptyAction,
		},
		{
			name: "missing resource",
			intent: Intent{
				Action:  "file.write",
				AgentID: "agent-1",
			},
			wantErr: ErrEmptyResource,
		},
		{
			name: "missing agent ID",
			intent: Intent{
				Action:   "file.write",
				Resource: "/tmp/output.txt",
			},
			wantErr: ErrEmptyAgentID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIntent(tt.intent)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		policies   []Policy
		intent     Intent
		wantEffect Effect
		wantReason string
	}{
		{
			name: "single allow rule matches",
			policies: []Policy{{
				Name: "allow-reads",
				Rules: []Rule{{
					Action: "file.read", Resource: "*", Effect: Allow,
				}},
			}},
			intent:     Intent{Action: "file.read", Resource: "/etc/hosts", AgentID: "a"},
			wantEffect: Allow,
			wantReason: `matched rule in policy "allow-reads"`,
		},
		{
			name: "single deny rule matches",
			policies: []Policy{{
				Name: "deny-writes",
				Rules: []Rule{{
					Action: "file.write", Resource: "*", Effect: Deny,
				}},
			}},
			intent:     Intent{Action: "file.write", Resource: "/etc/passwd", AgentID: "a"},
			wantEffect: Deny,
			wantReason: `matched rule in policy "deny-writes"`,
		},
		{
			name: "first match wins",
			policies: []Policy{{
				Name: "mixed",
				Rules: []Rule{
					{Action: "file.write", Resource: "/tmp/*", Effect: Allow},
					{Action: "file.write", Resource: "*", Effect: Deny},
				},
			}},
			intent:     Intent{Action: "file.write", Resource: "/tmp/test", AgentID: "a"},
			wantEffect: Allow,
			wantReason: `matched rule in policy "mixed"`,
		},
		{
			name:       "no match defaults to deny",
			policies:   []Policy{},
			intent:     Intent{Action: "file.write", Resource: "/foo", AgentID: "a"},
			wantEffect: Deny,
			wantReason: "no matching rule; default deny",
		},
		{
			name: "glob pattern on action",
			policies: []Policy{{
				Name: "all-file-ops",
				Rules: []Rule{{
					Action: "file.*", Resource: "*", Effect: Allow,
				}},
			}},
			intent:     Intent{Action: "file.delete", Resource: "/tmp/x", AgentID: "a"},
			wantEffect: Allow,
		},
		{
			name: "require approval effect",
			policies: []Policy{{
				Name: "review-deploys",
				Rules: []Rule{{
					Action: "deploy.*", Resource: "*", Effect: RequireApproval,
				}},
			}},
			intent:     Intent{Action: "deploy.production", Resource: "my-service", AgentID: "a"},
			wantEffect: RequireApproval,
		},
		{
			name: "cross-policy first match",
			policies: []Policy{
				{Name: "p1", Rules: []Rule{{Action: "net.*", Resource: "*", Effect: Deny}}},
				{Name: "p2", Rules: []Rule{{Action: "net.ping", Resource: "*", Effect: Allow}}},
			},
			intent:     Intent{Action: "net.ping", Resource: "8.8.8.8", AgentID: "a"},
			wantEffect: Deny, // p1 matches first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inter := NewIntermediary(tt.policies, newTestAuditLog(t), &mockExecutor{})
			result := inter.Evaluate(tt.intent)
			if result.Effect != tt.wantEffect {
				t.Fatalf("effect: got %q, want %q", result.Effect, tt.wantEffect)
			}
			if tt.wantReason != "" && result.Reason != tt.wantReason {
				t.Fatalf("reason: got %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestSubmit_AllowedIntentExecutes(t *testing.T) {
	exec := &mockExecutor{}
	al := newTestAuditLog(t)
	inter := NewIntermediary([]Policy{{
		Name:  "allow-all",
		Rules: []Rule{{Action: "*", Resource: "*", Effect: Allow}},
	}}, al, exec)

	intent := Intent{Action: "file.write", Resource: "/tmp/out", AgentID: "agent-1", Justification: "test"}
	effect, err := inter.Submit(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if effect != Allow {
		t.Fatalf("effect: got %q, want %q", effect, Allow)
	}
	if exec.calls() != 1 {
		t.Fatalf("executor calls: got %d, want 1", exec.calls())
	}

	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries: got %d, want 1", len(entries))
	}
	if entries[0].Decision != Allow {
		t.Fatalf("audit decision: got %q, want %q", entries[0].Decision, Allow)
	}
}

func TestSubmit_DeniedIntentDoesNotExecute(t *testing.T) {
	exec := &mockExecutor{}
	al := newTestAuditLog(t)
	inter := NewIntermediary([]Policy{{
		Name:  "deny-all",
		Rules: []Rule{{Action: "*", Resource: "*", Effect: Deny}},
	}}, al, exec)

	intent := Intent{Action: "file.write", Resource: "/tmp/out", AgentID: "agent-1", Justification: "test"}
	effect, err := inter.Submit(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if effect != Deny {
		t.Fatalf("effect: got %q, want %q", effect, Deny)
	}
	if exec.calls() != 0 {
		t.Fatalf("executor calls: got %d, want 0 (denied intent must not execute)", exec.calls())
	}
}

func TestSubmit_ExecutorErrorCaptured(t *testing.T) {
	execErr := errors.New("disk full")
	exec := &mockExecutor{err: execErr}
	al := newTestAuditLog(t)
	inter := NewIntermediary([]Policy{{
		Name:  "allow-all",
		Rules: []Rule{{Action: "*", Resource: "*", Effect: Allow}},
	}}, al, exec)

	intent := Intent{Action: "file.write", Resource: "/tmp/out", AgentID: "agent-1", Justification: "test"}
	effect, err := inter.Submit(context.Background(), intent)
	if effect != Allow {
		t.Fatalf("effect: got %q, want %q", effect, Allow)
	}
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected executor error, got %v", err)
	}

	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries: got %d, want 1", len(entries))
	}
	if entries[0].Error != "disk full" {
		t.Fatalf("audit error: got %q, want %q", entries[0].Error, "disk full")
	}
}

func TestSubmit_RequireApprovalNotExecuted(t *testing.T) {
	exec := &mockExecutor{}
	al := newTestAuditLog(t)
	inter := NewIntermediary([]Policy{{
		Name:  "review",
		Rules: []Rule{{Action: "*", Resource: "*", Effect: RequireApproval}},
	}}, al, exec)

	intent := Intent{Action: "deploy.prod", Resource: "my-svc", AgentID: "agent-1", Justification: "ship it"}
	effect, err := inter.Submit(context.Background(), intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if effect != RequireApproval {
		t.Fatalf("effect: got %q, want %q", effect, RequireApproval)
	}
	if exec.calls() != 0 {
		t.Fatalf("executor calls: got %d, want 0 (require_approval must not execute)", exec.calls())
	}

	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries: got %d, want 1", len(entries))
	}
	if entries[0].Decision != RequireApproval {
		t.Fatalf("audit decision: got %q, want %q", entries[0].Decision, RequireApproval)
	}
}

func TestSubmit_InvalidIntentDenied(t *testing.T) {
	exec := &mockExecutor{}
	al := newTestAuditLog(t)
	inter := NewIntermediary([]Policy{{
		Name:  "allow-all",
		Rules: []Rule{{Action: "*", Resource: "*", Effect: Allow}},
	}}, al, exec)

	intent := Intent{Action: "", Resource: "/tmp/out", AgentID: "agent-1"}
	effect, err := inter.Submit(context.Background(), intent)
	if effect != Deny {
		t.Fatalf("effect: got %q, want %q", effect, Deny)
	}
	if !errors.Is(err, ErrEmptyAction) {
		t.Fatalf("expected ErrEmptyAction, got %v", err)
	}
	if exec.calls() != 0 {
		t.Fatalf("executor calls: got %d, want 0", exec.calls())
	}

	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries: got %d, want 1 (validation failures are still audited)", len(entries))
	}
}

func TestAuditLog_AppendAndReadRoundTrip(t *testing.T) {
	al := newTestAuditLog(t)

	entries := []AuditEntry{
		{Intent: Intent{Action: "a", Resource: "r", AgentID: "1"}, Decision: Allow},
		{Intent: Intent{Action: "b", Resource: "s", AgentID: "2"}, Decision: Deny},
	}

	for _, e := range entries {
		if err := al.Append(e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := al.Entries()
	if err != nil {
		t.Fatalf("entries: %v", err)
	}
	if len(got) != len(entries) {
		t.Fatalf("entries count: got %d, want %d", len(got), len(entries))
	}
	for i, e := range got {
		if e.Intent.Action != entries[i].Intent.Action {
			t.Errorf("entry[%d] action: got %q, want %q", i, e.Intent.Action, entries[i].Intent.Action)
		}
		if e.Decision != entries[i].Decision {
			t.Errorf("entry[%d] decision: got %q, want %q", i, e.Decision, entries[i].Decision)
		}
	}
}

func TestAuditLog_JSONLFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	al := NewAuditLog(path)

	entry := AuditEntry{
		Intent:   Intent{Action: "file.read", Resource: "/etc/hosts", AgentID: "a1"},
		Decision: Allow,
	}
	if err := al.Append(entry); err != nil {
		t.Fatalf("append: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d", len(lines))
	}

	var parsed AuditEntry
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("unmarshal JSONL line: %v", err)
	}
	if parsed.Intent.Action != "file.read" {
		t.Fatalf("parsed action: got %q, want %q", parsed.Intent.Action, "file.read")
	}
}

func TestAuditLog_ConcurrentAppendSafety(t *testing.T) {
	al := newTestAuditLog(t)

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			entry := AuditEntry{
				Intent:   Intent{Action: "test", Resource: "r", AgentID: "a"},
				Decision: Allow,
			}
			if err := al.Append(entry); err != nil {
				t.Errorf("concurrent append %d: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("entries: %v", err)
	}
	if len(entries) != n {
		t.Fatalf("entries count: got %d, want %d", len(entries), n)
	}
}

func TestAuditLog_EntriesOnNonexistentFile(t *testing.T) {
	al := NewAuditLog(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	entries, err := al.Entries()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(entries))
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"file.read", "file.read", true},
		{"file.read", "file.write", false},
		{"file.*", "file.read", true},
		{"file.*", "file.write", true},
		{"file.*", "net.connect", false},
		{"?ile.read", "file.read", true},
		{"?ile.read", "xile.read", true},
		{"?ile.read", "xxile.read", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := MatchGlob(tt.pattern, tt.value)
			if got != tt.want {
				t.Fatalf("MatchGlob(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}
