package catalog

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// genScope generates a valid PermissionScope.
func genScope(t *rapid.T) PermissionScope {
	return PermissionScope(rapid.IntRange(1, 3).Draw(t, "scope"))
}

// genParamType generates a valid ParamType.
func genParamType(t *rapid.T) ParamType {
	types := []ParamType{ParamString, ParamInt, ParamBool, ParamArray, ParamObject}
	return types[rapid.IntRange(0, len(types)-1).Draw(t, "paramTypeIdx")]
}

// genTag generates a short alphabetic tag.
func genTag(t *rapid.T) string {
	return rapid.StringMatching(`[a-z]{2,8}`).Draw(t, "tag")
}

// genToolName generates a unique-ish tool name.
func genToolName(t *rapid.T) string {
	return rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(t, "toolName")
}

// genTool generates a valid Tool with random fields.
func genTool(t *rapid.T) Tool {
	nParams := rapid.IntRange(0, 3).Draw(t, "nParams")
	params := make([]Param, nParams)
	usedNames := make(map[string]struct{}, nParams)
	for i := range params {
		// Generate unique parameter names to satisfy validation.
		var name string
		for {
			name = rapid.StringMatching(`[a-z]{2,8}`).Draw(t, "paramName")
			if _, exists := usedNames[name]; !exists {
				break
			}
		}
		usedNames[name] = struct{}{}
		params[i] = Param{
			Name:     name,
			Type:     genParamType(t),
			Required: rapid.Bool().Draw(t, "required"),
		}
	}
	nTags := rapid.IntRange(0, 4).Draw(t, "nTags")
	tags := make([]string, nTags)
	for i := range tags {
		tags[i] = genTag(t)
	}
	return Tool{
		Name:        genToolName(t),
		Description: rapid.StringMatching(`[a-z]{1,5}[a-z ]{4,25}`).Draw(t, "desc"),
		Parameters:  params,
		Scope:       genScope(t),
		Tags:        tags,
	}
}

// --- Property: duplicate registration always rejected ---

func TestPropDuplicateRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		tool := genTool(t)
		if err := c.Register(tool); err != nil {
			t.Fatalf("first register failed: %v", err)
		}
		if err := c.Register(tool); err == nil {
			t.Fatal("duplicate register should fail")
		}
	})
}

// --- Property: register then get returns same tool ---

func TestPropRegisterGet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		tool := genTool(t)
		if err := c.Register(tool); err != nil {
			t.Fatalf("register: %v", err)
		}
		got, err := c.Get(tool.Name)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Name != tool.Name {
			t.Errorf("name mismatch: got %q, want %q", got.Name, tool.Name)
		}
		if got.Scope != tool.Scope {
			t.Errorf("scope mismatch: got %d, want %d", got.Scope, tool.Scope)
		}
	})
}

// --- Property: register then remove then get fails ---

func TestPropRegisterRemoveGet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		tool := genTool(t)
		if err := c.Register(tool); err != nil {
			t.Fatalf("register: %v", err)
		}
		if err := c.Remove(tool.Name); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if _, err := c.Get(tool.Name); err == nil {
			t.Fatal("get after remove should fail")
		}
	})
}

// --- Property: similarity is symmetric ---

func TestPropSimilaritySymmetric(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genTool(t)
		b := genTool(t)
		ab := ComputeSimilarity(a, b)
		ba := ComputeSimilarity(b, a)
		if ab != ba {
			t.Errorf("similarity not symmetric: %f != %f", ab, ba)
		}
	})
}

// --- Property: similarity in [0,1] ---

func TestPropSimilarityBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genTool(t)
		b := genTool(t)
		sim := ComputeSimilarity(a, b)
		if sim < 0 || sim > 1 {
			t.Errorf("similarity %f out of [0,1]", sim)
		}
	})
}

// --- Property: failure rate in [0,1] ---

func TestPropFailureRateBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		tool := genTool(t)
		if err := c.Register(tool); err != nil {
			t.Fatalf("register: %v", err)
		}
		n := rapid.IntRange(1, 50).Draw(t, "nCalls")
		for i := 0; i < n; i++ {
			success := rapid.Bool().Draw(t, "success")
			_ = c.RecordUsage(tool.Name, success, time.Millisecond, 0.1)
		}
		rate, err := c.FailureRate(tool.Name)
		if err != nil {
			t.Fatalf("failure rate: %v", err)
		}
		if rate < 0 || rate > 1 {
			t.Errorf("failure rate %f out of [0,1]", rate)
		}
	})
}

// --- Property: metrics are non-negative ---

func TestPropMetricsNonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		tool := genTool(t)
		if err := c.Register(tool); err != nil {
			t.Fatalf("register: %v", err)
		}
		n := rapid.IntRange(0, 20).Draw(t, "nCalls")
		for i := 0; i < n; i++ {
			success := rapid.Bool().Draw(t, "success")
			lat := time.Duration(rapid.IntRange(0, 5000).Draw(t, "latMs")) * time.Millisecond
			cost := float64(rapid.IntRange(0, 100).Draw(t, "cost")) / 10.0
			_ = c.RecordUsage(tool.Name, success, lat, cost)
		}
		m, err := c.GetMetrics(tool.Name)
		if err != nil {
			t.Fatalf("get metrics: %v", err)
		}
		if m.Calls < 0 {
			t.Errorf("calls negative: %d", m.Calls)
		}
		if m.Failures < 0 {
			t.Errorf("failures negative: %d", m.Failures)
		}
		if m.TotalLatency < 0 {
			t.Errorf("latency negative: %v", m.TotalLatency)
		}
		if m.TotalTokenCost < 0 {
			t.Errorf("token cost negative: %f", m.TotalTokenCost)
		}
		if m.Failures > m.Calls {
			t.Errorf("failures (%d) > calls (%d)", m.Failures, m.Calls)
		}
	})
}

// --- Property: authorization respects scope hierarchy ---

func TestPropAuthorizationHierarchy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		toolScope := genScope(t)
		tool := Tool{
			Name:        genToolName(t),
			Description: "test tool",
			Scope:       toolScope,
		}
		if err := c.Register(tool); err != nil {
			t.Fatalf("register: %v", err)
		}
		roleScope := genScope(t)
		role := "role-" + rapid.StringMatching(`[a-z]{3}`).Draw(t, "roleSuffix")
		_ = c.SetRolePermissions(RolePermissions{
			Role:         role,
			MaxScope:     roleScope,
			AllowedTools: []string{tool.Name},
		})
		allowed, err := c.Authorize(role, tool.Name)
		if err != nil {
			t.Fatalf("authorize: %v", err)
		}
		if toolScope > roleScope && allowed {
			t.Errorf("tool scope %d > role scope %d but was authorized", toolScope, roleScope)
		}
		if toolScope <= roleScope && !allowed {
			t.Errorf("tool scope %d <= role scope %d but was denied", toolScope, roleScope)
		}
	})
}

// --- Property: unregistered tool operations return error ---

func TestPropUnregisteredToolErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := NewCatalog()
		name := genToolName(t)
		if _, err := c.Get(name); err == nil {
			t.Error("get on empty catalog should fail")
		}
		if err := c.Remove(name); err == nil {
			t.Error("remove on empty catalog should fail")
		}
		if err := c.RecordUsage(name, true, 0, 0); err == nil {
			t.Error("record usage on empty catalog should fail")
		}
		if _, err := c.GetMetrics(name); err == nil {
			t.Error("get metrics on empty catalog should fail")
		}
		if _, err := c.FailureRate(name); err == nil {
			t.Error("failure rate on empty catalog should fail")
		}
		if _, err := c.AvgLatency(name); err == nil {
			t.Error("avg latency on empty catalog should fail")
		}
		if _, err := c.AvgTokenCost(name); err == nil {
			t.Error("avg token cost on empty catalog should fail")
		}
	})
}
