package catalog

import (
	"testing"
	"time"
)

// --- helper ---

func makeTool(name string, scope PermissionScope, tags []string) Tool {
	return Tool{
		Name:        name,
		Description: "A tool named " + name,
		Scope:       scope,
		Tags:        tags,
	}
}

// --- Register tests ---

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{
			name:    "valid tool",
			tool:    makeTool("read-file", ScopeReadOnly, nil),
			wantErr: false,
		},
		{
			name:    "empty name rejected",
			tool:    Tool{Name: "", Description: "desc", Scope: ScopeReadOnly},
			wantErr: true,
		},
		{
			name:    "empty description rejected",
			tool:    Tool{Name: "t", Description: "", Scope: ScopeReadOnly},
			wantErr: true,
		},
		{
			name:    "invalid scope rejected",
			tool:    Tool{Name: "t", Description: "d", Scope: 99},
			wantErr: true,
		},
		{
			name: "invalid param type rejected",
			tool: Tool{
				Name:        "t",
				Description: "d",
				Scope:       ScopeReadOnly,
				Parameters:  []Param{{Name: "p", Type: "unknown"}},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCatalog()
			err := c.Register(tc.tool)
			if (err != nil) != tc.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRegisterDuplicate(t *testing.T) {
	c := NewCatalog()
	tool := makeTool("dup", ScopeReadOnly, nil)
	if err := c.Register(tool); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := c.Register(tool); err == nil {
		t.Error("expected error on duplicate registration")
	}
}

// --- Get tests ---

func TestGet(t *testing.T) {
	c := NewCatalog()
	tool := makeTool("my-tool", ScopeReadOnly, []string{"io", "file"})
	if err := c.Register(tool); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, err := c.Get("my-tool")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "my-tool" {
		t.Errorf("got name %q, want %q", got.Name, "my-tool")
	}
	if got.Description != tool.Description {
		t.Errorf("got description %q, want %q", got.Description, tool.Description)
	}
	if got.Scope != tool.Scope {
		t.Errorf("got scope %d, want %d", got.Scope, tool.Scope)
	}
	if len(got.Tags) != len(tool.Tags) {
		t.Fatalf("got %d tags, want %d", len(got.Tags), len(tool.Tags))
	}
	for i, tag := range tool.Tags {
		if got.Tags[i] != tag {
			t.Errorf("tag[%d] = %q, want %q", i, got.Tags[i], tag)
		}
	}
}

func TestGetReturnsCopyIsolatedFromCatalog(t *testing.T) {
	c := NewCatalog()
	if err := c.Register(makeTool("iso", ScopeReadOnly, []string{"original"})); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := c.Get("iso")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// Mutate the returned copy's Tags slice.
	got.Tags[0] = "mutated"
	got.Tags = append(got.Tags, "extra")

	// Fetch again and verify the catalog's copy is unchanged.
	again, err := c.Get("iso")
	if err != nil {
		t.Fatalf("get after mutation: %v", err)
	}
	if len(again.Tags) != 1 {
		t.Fatalf("catalog tags length = %d, want 1", len(again.Tags))
	}
	if again.Tags[0] != "original" {
		t.Errorf("catalog tag = %q, want %q", again.Tags[0], "original")
	}
}

func TestGetNotFound(t *testing.T) {
	c := NewCatalog()
	_, err := c.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// --- List tests ---

func TestListEmpty(t *testing.T) {
	c := NewCatalog()
	if got := c.List(); len(got) != 0 {
		t.Errorf("expected empty list, got %d items", len(got))
	}
}

func TestListReturnsAll(t *testing.T) {
	c := NewCatalog()
	want := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	for name := range want {
		if err := c.Register(makeTool(name, ScopeReadOnly, nil)); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}
	got := c.List()
	if len(got) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(got))
	}
	gotNames := make(map[string]struct{}, len(got))
	for _, tool := range got {
		gotNames[tool.Name] = struct{}{}
	}
	for name := range want {
		if _, ok := gotNames[name]; !ok {
			t.Errorf("missing tool %q in List() result", name)
		}
	}
}

// --- ListByTag tests ---

func TestListByTag(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("a", ScopeReadOnly, []string{"io", "file"}))
	_ = c.Register(makeTool("b", ScopeReadOnly, []string{"io", "net"}))
	_ = c.Register(makeTool("c", ScopeReadOnly, []string{"compute"}))

	tests := []struct {
		tag      string
		wantLen  int
	}{
		{"io", 2},
		{"file", 1},
		{"compute", 1},
		{"missing", 0},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			got := c.ListByTag(tc.tag)
			if len(got) != tc.wantLen {
				t.Errorf("ListByTag(%q) returned %d tools, want %d", tc.tag, len(got), tc.wantLen)
			}
		})
	}
}

// --- Remove tests ---

func TestRemove(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("rm-me", ScopeReadOnly, nil))
	if err := c.Remove("rm-me"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	_, err := c.Get("rm-me")
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestRemoveNotFound(t *testing.T) {
	c := NewCatalog()
	if err := c.Remove("nope"); err == nil {
		t.Error("expected error removing nonexistent tool")
	}
}

// --- DetectOverlaps tests ---

func TestDetectOverlapsSymmetric(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(Tool{
		Name:        "read-file",
		Description: "reads a file from disk",
		Scope:       ScopeReadOnly,
		Tags:        []string{"io", "file", "read"},
	})
	_ = c.Register(Tool{
		Name:        "load-file",
		Description: "loads a file from disk",
		Scope:       ScopeReadOnly,
		Tags:        []string{"io", "file", "load"},
	})
	overlaps := c.DetectOverlaps()
	if len(overlaps) == 0 {
		t.Fatal("expected at least one overlap")
	}
	o := overlaps[0]
	// Symmetric: either order is fine, but both names must appear.
	if !((o.ToolA == "read-file" && o.ToolB == "load-file") || (o.ToolA == "load-file" && o.ToolB == "read-file")) {
		t.Errorf("unexpected overlap pair: %q and %q", o.ToolA, o.ToolB)
	}
	if o.Similarity <= 0 || o.Similarity > 1 {
		t.Errorf("similarity %f out of range (0,1]", o.Similarity)
	}
}

func TestDetectOverlapsNoOverlap(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(Tool{
		Name:        "a",
		Description: "completely unique function alpha",
		Scope:       ScopeReadOnly,
		Tags:        []string{"x"},
	})
	_ = c.Register(Tool{
		Name:        "b",
		Description: "totally different purpose beta",
		Scope:       ScopeReadOnly,
		Tags:        []string{"y"},
	})
	if overlaps := c.DetectOverlaps(); len(overlaps) != 0 {
		t.Errorf("expected no overlaps, got %d", len(overlaps))
	}
}

// --- Authorization tests ---

func TestAuthorize(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("read", ScopeReadOnly, nil))
	_ = c.Register(makeTool("write", ScopeWriteWithApproval, nil))
	_ = c.Register(makeTool("admin", ScopeFullAutonomy, nil))
	_ = c.SetRolePermissions(RolePermissions{
		Role:         "viewer",
		MaxScope:     ScopeReadOnly,
		AllowedTools: []string{"read"},
	})
	_ = c.SetRolePermissions(RolePermissions{
		Role:         "editor",
		MaxScope:     ScopeWriteWithApproval,
		AllowedTools: []string{"read", "write"},
	})
	_ = c.SetRolePermissions(RolePermissions{
		Role:         "superuser",
		MaxScope:     ScopeFullAutonomy,
		AllowedTools: []string{"read", "write", "admin"},
	})

	tests := []struct {
		name    string
		role    string
		tool    string
		want    bool
		wantErr bool
	}{
		{"viewer reads", "viewer", "read", true, false},
		{"viewer cannot write (scope too high)", "viewer", "write", false, false},
		{"editor reads", "editor", "read", true, false},
		{"editor writes", "editor", "write", true, false},
		{"editor cannot admin (scope too high)", "editor", "admin", false, false},
		{"superuser can admin", "superuser", "admin", true, false},
		{"unknown role", "ghost", "read", false, true},
		{"unknown tool", "viewer", "nope", false, true},
		{"tool not in allowed list", "viewer", "admin", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.Authorize(tc.role, tc.tool)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Authorize() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Authorize(%q, %q) = %v, want %v", tc.role, tc.tool, got, tc.want)
			}
		})
	}
}

// --- SetRolePermissions tests ---

func TestSetRolePermissionsValidation(t *testing.T) {
	c := NewCatalog()
	tests := []struct {
		name    string
		rp      RolePermissions
		wantErr bool
	}{
		{"valid", RolePermissions{Role: "a", MaxScope: ScopeReadOnly}, false},
		{"empty role", RolePermissions{Role: "", MaxScope: ScopeReadOnly}, true},
		{"invalid scope", RolePermissions{Role: "a", MaxScope: 0}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := c.SetRolePermissions(tc.rp)
			if (err != nil) != tc.wantErr {
				t.Errorf("SetRolePermissions() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- Metrics tests ---

func TestRecordUsageAndGetMetrics(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("t", ScopeReadOnly, nil))

	if err := c.RecordUsage("t", true, 100*time.Millisecond, 0.5); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := c.RecordUsage("t", false, 200*time.Millisecond, 1.0); err != nil {
		t.Fatalf("record: %v", err)
	}

	m, err := c.GetMetrics("t")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if m.Calls != 2 {
		t.Errorf("calls = %d, want 2", m.Calls)
	}
	if m.Failures != 1 {
		t.Errorf("failures = %d, want 1", m.Failures)
	}
	if m.TotalLatency != 300*time.Millisecond {
		t.Errorf("total latency = %v, want 300ms", m.TotalLatency)
	}
	if m.TotalTokenCost != 1.5 {
		t.Errorf("total token cost = %f, want 1.5", m.TotalTokenCost)
	}
	if m.LastUsed.IsZero() {
		t.Error("LastUsed should be non-zero after recording usage")
	}
	if elapsed := time.Since(m.LastUsed); elapsed >= time.Second {
		t.Errorf("LastUsed is %v ago, want < 1s", elapsed)
	}
}

func TestRecordUsageUnknownTool(t *testing.T) {
	c := NewCatalog()
	if err := c.RecordUsage("nope", true, 0, 0); err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestGetMetricsUnknownTool(t *testing.T) {
	c := NewCatalog()
	_, err := c.GetMetrics("nope")
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestAllMetrics(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("a", ScopeReadOnly, nil))
	_ = c.Register(makeTool("b", ScopeReadOnly, nil))
	_ = c.RecordUsage("a", true, time.Second, 1)

	all := c.AllMetrics()
	if len(all) != 2 {
		t.Errorf("expected 2 metrics entries, got %d", len(all))
	}
	if all["a"].Calls != 1 {
		t.Errorf("expected 1 call for a, got %d", all["a"].Calls)
	}
	if all["b"].Calls != 0 {
		t.Errorf("expected 0 calls for b, got %d", all["b"].Calls)
	}
}

func TestFailureRate(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("t", ScopeReadOnly, nil))

	rate, err := c.FailureRate("t")
	if err != nil {
		t.Fatalf("failure rate: %v", err)
	}
	if rate != 0 {
		t.Errorf("expected 0 for no calls, got %f", rate)
	}

	_ = c.RecordUsage("t", true, 0, 0)
	_ = c.RecordUsage("t", false, 0, 0)
	_ = c.RecordUsage("t", false, 0, 0)

	rate, err = c.FailureRate("t")
	if err != nil {
		t.Fatalf("failure rate: %v", err)
	}
	want := 2.0 / 3.0
	if diff := rate - want; diff > 0.001 || diff < -0.001 {
		t.Errorf("failure rate = %f, want ~%f", rate, want)
	}
}

func TestFailureRateUnknown(t *testing.T) {
	c := NewCatalog()
	_, err := c.FailureRate("nope")
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestAvgLatency(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("t", ScopeReadOnly, nil))

	avg, err := c.AvgLatency("t")
	if err != nil {
		t.Fatalf("avg latency: %v", err)
	}
	if avg != 0 {
		t.Errorf("expected 0 for no calls, got %v", avg)
	}

	_ = c.RecordUsage("t", true, 100*time.Millisecond, 0)
	_ = c.RecordUsage("t", true, 300*time.Millisecond, 0)

	avg, err = c.AvgLatency("t")
	if err != nil {
		t.Fatalf("avg latency: %v", err)
	}
	if avg != 200*time.Millisecond {
		t.Errorf("avg latency = %v, want 200ms", avg)
	}
}

func TestAvgLatencyUnknown(t *testing.T) {
	c := NewCatalog()
	_, err := c.AvgLatency("nope")
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestAvgTokenCost(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("t", ScopeReadOnly, nil))

	avg, err := c.AvgTokenCost("t")
	if err != nil {
		t.Fatalf("avg token cost: %v", err)
	}
	if avg != 0 {
		t.Errorf("expected 0 for no calls, got %f", avg)
	}

	_ = c.RecordUsage("t", true, 0, 2.0)
	_ = c.RecordUsage("t", true, 0, 4.0)

	avg, err = c.AvgTokenCost("t")
	if err != nil {
		t.Fatalf("avg token cost: %v", err)
	}
	if avg != 3.0 {
		t.Errorf("avg token cost = %f, want 3.0", avg)
	}
}

func TestAvgTokenCostUnknown(t *testing.T) {
	c := NewCatalog()
	_, err := c.AvgTokenCost("nope")
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

// --- ValidateTool tests ---

func TestValidateTool(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{"valid minimal", Tool{Name: "t", Description: "d", Scope: ScopeReadOnly}, false},
		{"valid with params", Tool{
			Name: "t", Description: "d", Scope: ScopeFullAutonomy,
			Parameters: []Param{{Name: "p", Type: ParamString}},
		}, false},
		{"missing name", Tool{Description: "d", Scope: ScopeReadOnly}, true},
		{"missing description", Tool{Name: "t", Scope: ScopeReadOnly}, true},
		{"scope zero", Tool{Name: "t", Description: "d", Scope: 0}, true},
		{"scope too high", Tool{Name: "t", Description: "d", Scope: 10}, true},
		{"param empty name", Tool{
			Name: "t", Description: "d", Scope: ScopeReadOnly,
			Parameters: []Param{{Name: "", Type: ParamString}},
		}, true},
		{"param bad type", Tool{
			Name: "t", Description: "d", Scope: ScopeReadOnly,
			Parameters: []Param{{Name: "p", Type: "float"}},
		}, true},
		{"duplicate param names", Tool{
			Name: "t", Description: "d", Scope: ScopeReadOnly,
			Parameters: []Param{
				{Name: "dup", Type: ParamString},
				{Name: "dup", Type: ParamInt},
			},
		}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTool(tc.tool)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateTool() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- ComputeSimilarity tests ---

func TestComputeSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b Tool
		low  float64 // expected similarity >= low
		high float64 // expected similarity <= high
	}{
		{
			name: "identical",
			a:    Tool{Name: "x", Description: "read file from disk", Tags: []string{"io", "file"}},
			b:    Tool{Name: "y", Description: "read file from disk", Tags: []string{"io", "file"}},
			low:  0.99, high: 1.01,
		},
		{
			name: "no overlap",
			a:    Tool{Name: "x", Description: "alpha", Tags: []string{"a"}},
			b:    Tool{Name: "y", Description: "beta", Tags: []string{"b"}},
			low:  -0.01, high: 0.01,
		},
		{
			name: "partial overlap",
			a:    Tool{Name: "x", Description: "read file from disk", Tags: []string{"io", "file"}},
			b:    Tool{Name: "y", Description: "write file to disk", Tags: []string{"io", "file"}},
			low:  0.4, high: 0.95,
		},
		{
			name: "both empty",
			a:    Tool{Name: "x", Description: "", Tags: nil},
			b:    Tool{Name: "y", Description: "", Tags: nil},
			low:  -0.01, high: 0.01,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sim := ComputeSimilarity(tc.a, tc.b)
			if sim < tc.low || sim > tc.high {
				t.Errorf("ComputeSimilarity() = %f, want [%f, %f]", sim, tc.low, tc.high)
			}
		})
	}
}

func TestComputeSimilaritySymmetric(t *testing.T) {
	a := Tool{Name: "x", Description: "read file from disk", Tags: []string{"io", "file"}}
	b := Tool{Name: "y", Description: "write file to disk", Tags: []string{"io", "disk"}}
	ab := ComputeSimilarity(a, b)
	ba := ComputeSimilarity(b, a)
	if ab != ba {
		t.Errorf("similarity not symmetric: %f != %f", ab, ba)
	}
}

// --- Remove cleans up metrics ---

func TestRemoveCleansMetrics(t *testing.T) {
	c := NewCatalog()
	_ = c.Register(makeTool("t", ScopeReadOnly, nil))
	_ = c.RecordUsage("t", true, time.Second, 1)
	_ = c.Remove("t")

	_, err := c.GetMetrics("t")
	if err == nil {
		t.Error("expected metrics to be removed after tool removal")
	}
}
