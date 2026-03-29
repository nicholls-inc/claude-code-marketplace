package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupRepo creates a temp directory with the given files and dirs.
func setupRepo(t *testing.T, files []string, dirs []string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		p := filepath.Join(root, d)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", p, err)
		}
	}
	for _, f := range files {
		p := filepath.Join(root, f)
		dir := filepath.Dir(p)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
		if err := os.WriteFile(p, []byte("// placeholder"), 0o644); err != nil {
			t.Fatalf("write %q: %v", p, err)
		}
	}
	return root
}

func TestDetectLanguages(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantLang string
		wantMin  int
	}{
		{
			name:     "go files",
			files:    []string{"main.go", "lib.go", "util.go"},
			wantLang: "Go",
			wantMin:  3,
		},
		{
			name:     "python files",
			files:    []string{"app.py", "test.py"},
			wantLang: "Python",
			wantMin:  2,
		},
		{
			name:     "typescript files",
			files:    []string{"index.ts", "app.tsx", "util.ts"},
			wantLang: "TypeScript",
			wantMin:  3,
		},
		{
			name:     "mixed languages",
			files:    []string{"main.go", "script.py"},
			wantLang: "Go",
			wantMin:  1,
		},
		{
			name:     "empty repo",
			files:    nil,
			wantLang: "",
			wantMin:  0,
		},
		{
			name:     "unknown extensions only",
			files:    []string{"readme.txt", "data.csv"},
			wantLang: "",
			wantMin:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, nil)
			langs := DetectLanguages(root)

			if tt.wantLang == "" {
				if len(langs) != 0 {
					t.Fatalf("expected no languages, got %d", len(langs))
				}
				return
			}

			if len(langs) == 0 {
				t.Fatalf("expected at least one language, got none")
			}

			found := false
			for _, l := range langs {
				if l.Name == tt.wantLang && l.FileCount >= tt.wantMin {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected language %q with at least %d files, got %v", tt.wantLang, tt.wantMin, langs)
			}
		})
	}
}

func TestDetectLanguagesSkipsHiddenAndVendor(t *testing.T) {
	root := setupRepo(t, nil, nil)
	// Create files in hidden and vendor directories.
	for _, d := range []string{".git", "node_modules", "vendor", "__pycache__"} {
		dir := filepath.Join(root, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("package x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	langs := DetectLanguages(root)
	if len(langs) != 0 {
		t.Fatalf("expected no languages from hidden/vendor dirs, got %v", langs)
	}
}

func TestDetectLanguagesConfidence(t *testing.T) {
	files := []string{"a.go", "b.go", "c.go", "d.py"}
	root := setupRepo(t, files, nil)
	langs := DetectLanguages(root)

	if len(langs) < 2 {
		t.Fatalf("expected at least 2 languages, got %d", len(langs))
	}

	// Go should be first (3 files vs 1).
	if langs[0].Name != "Go" {
		t.Fatalf("expected Go first, got %s", langs[0].Name)
	}

	// Confidence should be 3/4 = 0.75.
	if langs[0].Confidence < 0.7 || langs[0].Confidence > 0.8 {
		t.Fatalf("expected Go confidence ~0.75, got %f", langs[0].Confidence)
	}
}

func TestDetectFrameworks(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantNames []string
	}{
		{
			name:      "go module",
			files:     []string{"go.mod"},
			wantNames: []string{"Go Modules"},
		},
		{
			name:      "node project",
			files:     []string{"package.json"},
			wantNames: []string{"Node.js"},
		},
		{
			name:      "python project",
			files:     []string{"requirements.txt", "pyproject.toml"},
			wantNames: []string{"pip", "Python Project"},
		},
		{
			name:      "rust project",
			files:     []string{"Cargo.toml"},
			wantNames: []string{"Cargo"},
		},
		{
			name:      "no frameworks",
			files:     []string{"random.txt"},
			wantNames: nil,
		},
		{
			name:      "multiple frameworks",
			files:     []string{"go.mod", "package.json"},
			wantNames: []string{"Go Modules", "Node.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, nil)
			fws := DetectFrameworks(root)

			if tt.wantNames == nil {
				if len(fws) != 0 {
					t.Fatalf("expected no frameworks, got %d", len(fws))
				}
				return
			}

			names := make(map[string]bool)
			for _, fw := range fws {
				names[fw.Name] = true
			}
			for _, want := range tt.wantNames {
				if !names[want] {
					t.Fatalf("expected framework %q, not found in %v", want, fws)
				}
			}
		})
	}
}

func TestDetectBuildTools(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantNames []string
	}{
		{
			name:      "makefile",
			files:     []string{"Makefile"},
			wantNames: []string{"Make"},
		},
		{
			name:      "docker",
			files:     []string{"Dockerfile"},
			wantNames: []string{"Docker"},
		},
		{
			name:      "docker compose yaml",
			files:     []string{"docker-compose.yml"},
			wantNames: []string{"Docker Compose"},
		},
		{
			name:      "docker compose both variants",
			files:     []string{"docker-compose.yml", "docker-compose.yaml"},
			wantNames: []string{"Docker Compose"},
		},
		{
			name:      "multiple tools",
			files:     []string{"Makefile", "Dockerfile"},
			wantNames: []string{"Make", "Docker"},
		},
		{
			name:      "no build tools",
			files:     []string{"readme.md"},
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, nil)
			tools := DetectBuildTools(root)

			if tt.wantNames == nil {
				if len(tools) != 0 {
					t.Fatalf("expected no build tools, got %d", len(tools))
				}
				return
			}

			names := make(map[string]bool)
			for _, bt := range tools {
				names[bt.Name] = true
			}
			for _, want := range tt.wantNames {
				if !names[want] {
					t.Fatalf("expected build tool %q, not found in %v", want, tools)
				}
			}
		})
	}
}

func TestDetectBuildToolsNoDuplicates(t *testing.T) {
	root := setupRepo(t, []string{"docker-compose.yml", "docker-compose.yaml"}, nil)
	tools := DetectBuildTools(root)

	count := 0
	for _, bt := range tools {
		if bt.Name == "Docker Compose" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 Docker Compose entry, got %d", count)
	}
}

func TestDiscoverEntryPoints(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantNames []string
	}{
		{
			name:      "makefile entry point",
			files:     []string{"Makefile"},
			wantNames: []string{"make"},
		},
		{
			name:      "go main",
			files:     []string{"main.go"},
			wantNames: []string{"go run"},
		},
		{
			name:      "node project",
			files:     []string{"package.json"},
			wantNames: []string{"npm"},
		},
		{
			name:      "python app",
			files:     []string{"app.py"},
			wantNames: []string{"python app"},
		},
		{
			name:      "no entry points",
			files:     []string{"data.csv"},
			wantNames: nil,
		},
		{
			name:      "multiple entry points",
			files:     []string{"Makefile", "main.go", "package.json"},
			wantNames: []string{"make", "go run", "npm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, nil)
			eps := DiscoverEntryPoints(root)

			if tt.wantNames == nil {
				if len(eps) != 0 {
					t.Fatalf("expected no entry points, got %d", len(eps))
				}
				return
			}

			names := make(map[string]bool)
			for _, ep := range eps {
				names[ep.Name] = true
			}
			for _, want := range tt.wantNames {
				if !names[want] {
					t.Fatalf("expected entry point %q, not found in %v", want, eps)
				}
			}
		})
	}
}

func TestDiscoverEntryPointsExists(t *testing.T) {
	root := setupRepo(t, []string{"Makefile"}, nil)
	eps := DiscoverEntryPoints(root)

	for _, ep := range eps {
		if !ep.Exists {
			t.Fatalf("expected entry point %q to have Exists=true", ep.Name)
		}
		if ep.Error != "" {
			t.Fatalf("expected no error for entry point %q, got %q", ep.Name, ep.Error)
		}
	}
}

func TestDetectTechStack(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		dirs         []string
		wantTech     []string
		wantWarnings []string
	}{
		{
			name:     "docker project",
			files:    []string{"Dockerfile"},
			wantTech: []string{"Docker"},
			wantWarnings: []string{"Docker"},
		},
		{
			name:     "clean project",
			files:    []string{"main.go", "go.mod"},
			wantTech: []string{"Go Modules"},
		},
		{
			name:     "terraform project",
			files:    []string{"main.tf"},
			wantTech: []string{"Terraform"},
			wantWarnings: []string{"Terraform"},
		},
		{
			name:     "github actions",
			files:    []string{".github/workflows/ci.yml"},
			dirs:     []string{".github/workflows"},
			wantTech: []string{"GitHub Actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, tt.dirs)
			profile := &RepoProfile{Path: root}
			profile.Frameworks = DetectFrameworks(root)
			profile.BuildTools = DetectBuildTools(root)
			stack := DetectTechStack(profile)

			techNames := make(map[string]bool)
			for _, tech := range stack.Detected {
				techNames[tech.Name] = true
			}
			for _, want := range tt.wantTech {
				if !techNames[want] {
					t.Fatalf("expected technology %q, got %v", want, stack.Detected)
				}
			}

			warnTech := make(map[string]bool)
			for _, w := range stack.Warnings {
				warnTech[w.Technology] = true
			}
			for _, want := range tt.wantWarnings {
				if !warnTech[want] {
					t.Fatalf("expected warning for %q, got %v", want, stack.Warnings)
				}
			}
		})
	}
}

func TestDetectTechStackWarningLevels(t *testing.T) {
	root := setupRepo(t, []string{"Dockerfile", "main.tf"}, nil)
	profile := &RepoProfile{Path: root}
	profile.BuildTools = DetectBuildTools(root)
	profile.Frameworks = DetectFrameworks(root)
	stack := DetectTechStack(profile)

	warnMap := make(map[string]Compatibility)
	for _, w := range stack.Warnings {
		warnMap[w.Technology] = w
	}

	if w, ok := warnMap["Docker"]; ok {
		if w.Level != "moderate" {
			t.Fatalf("Docker level = %q, want moderate", w.Level)
		}
	} else {
		t.Fatal("expected Docker warning")
	}

	if w, ok := warnMap["Terraform"]; ok {
		if w.Level != "poor" {
			t.Fatalf("Terraform level = %q, want poor", w.Level)
		}
	} else {
		t.Fatal("expected Terraform warning")
	}
}

func TestAnalyzeRepo(t *testing.T) {
	root := setupRepo(t, []string{"main.go", "go.mod", "Makefile"}, nil)

	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	if profile.Path != root {
		t.Fatalf("Path = %q, want %q", profile.Path, root)
	}

	if profile.AnalyzedAt.IsZero() {
		t.Fatal("AnalyzedAt should not be zero")
	}

	if len(profile.Languages) == 0 {
		t.Fatal("expected at least one language")
	}
	if profile.Languages[0].Name != "Go" {
		t.Fatalf("expected first language to be Go, got %q", profile.Languages[0].Name)
	}

	if len(profile.Frameworks) == 0 {
		t.Fatal("expected at least one framework")
	}
	fwNames := make(map[string]bool)
	for _, fw := range profile.Frameworks {
		fwNames[fw.Name] = true
	}
	if !fwNames["Go Modules"] {
		t.Fatalf("expected Go Modules framework, got %v", profile.Frameworks)
	}

	if len(profile.BuildTools) == 0 {
		t.Fatal("expected at least one build tool")
	}
	btNames := make(map[string]bool)
	for _, bt := range profile.BuildTools {
		btNames[bt.Name] = true
	}
	if !btNames["Make"] {
		t.Fatalf("expected Make build tool, got %v", profile.BuildTools)
	}

	if len(profile.EntryPoints) == 0 {
		t.Fatal("expected at least one entry point")
	}
	epNames := make(map[string]bool)
	for _, ep := range profile.EntryPoints {
		epNames[ep.Name] = true
	}
	if !epNames["make"] {
		t.Fatalf("expected 'make' entry point, got %v", profile.EntryPoints)
	}
	if !epNames["go run"] {
		t.Fatalf("expected 'go run' entry point, got %v", profile.EntryPoints)
	}
}

func TestAnalyzeRepoNotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := AnalyzeRepo(f)
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected 'not a directory' error, got: %v", err)
	}
}

func TestAnalyzeRepoMissing(t *testing.T) {
	_, err := AnalyzeRepo(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestAnalyzeRepoEmpty(t *testing.T) {
	root := t.TempDir()
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	if len(profile.Languages) != 0 {
		t.Fatalf("expected no languages for empty repo, got %d", len(profile.Languages))
	}
	if len(profile.Frameworks) != 0 {
		t.Fatalf("expected no frameworks for empty repo, got %d", len(profile.Frameworks))
	}
	if len(profile.BuildTools) != 0 {
		t.Fatalf("expected no build tools for empty repo, got %d", len(profile.BuildTools))
	}
	if len(profile.EntryPoints) != 0 {
		t.Fatalf("expected no entry points for empty repo, got %d", len(profile.EntryPoints))
	}
}

func TestAuditLegibility(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		dirs      []string
		wantAbove float64
		wantBelow float64
	}{
		{
			name:      "empty repo",
			wantAbove: -0.01,
			wantBelow: 0.01,
		},
		{
			name:      "well-structured repo",
			files:     []string{"README.md", "go.mod", "Makefile", "AGENTS.md", "main.go", "main_test.go", ".golangci.yml", ".editorconfig", "ARCHITECTURE.md", "CONTRIBUTING.md", "CHANGELOG.md"},
			dirs:      []string{"scripts", "internal", "docs", "docs/adr", ".github/workflows"},
			wantAbove: 0.93,
			wantBelow: 0.95,
		},
		{
			name:      "minimal repo",
			files:     []string{"README.md", "main.go"},
			dirs:      []string{"src"},
			wantAbove: 0.24,
			wantBelow: 0.26,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupRepo(t, tt.files, tt.dirs)
			profile, err := AnalyzeRepo(root)
			if err != nil {
				t.Fatalf("AnalyzeRepo: %v", err)
			}

			report, err := AuditLegibility(root, profile)
			if err != nil {
				t.Fatalf("AuditLegibility: %v", err)
			}

			if report.Overall < tt.wantAbove || report.Overall > tt.wantBelow {
				t.Fatalf("Overall = %f, want in [%f, %f]", report.Overall, tt.wantAbove, tt.wantBelow)
			}

			if len(report.Dimensions) != 7 {
				t.Fatalf("expected 7 dimensions, got %d", len(report.Dimensions))
			}

			if report.RepoPath != root {
				t.Fatalf("RepoPath = %q, want %q", report.RepoPath, root)
			}

			if report.AuditedAt.IsZero() {
				t.Fatal("AuditedAt should not be zero")
			}
		})
	}
}

func TestAuditLegibilityAllDimensionsPresent(t *testing.T) {
	root := t.TempDir()
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	report, err := AuditLegibility(root, profile)
	if err != nil {
		t.Fatalf("AuditLegibility: %v", err)
	}

	wantDims := map[string]bool{
		"Bootstrap Self-Sufficiency": false,
		"Task Entry Points":          false,
		"Validation Harness":         false,
		"Linting/Formatting":         false,
		"Codebase Map":               false,
		"Doc Structure":              false,
		"Decision Records":           false,
	}

	for _, ds := range report.Dimensions {
		if _, ok := wantDims[ds.Dimension.Name]; !ok {
			t.Fatalf("unexpected dimension %q", ds.Dimension.Name)
		}
		wantDims[ds.Dimension.Name] = true
	}

	for name, found := range wantDims {
		if !found {
			t.Fatalf("missing dimension %q", name)
		}
	}
}

func TestAuditLegibilityScoreBounds(t *testing.T) {
	root := setupRepo(t, []string{"README.md", "go.mod", "main.go"}, nil)
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	report, err := AuditLegibility(root, profile)
	if err != nil {
		t.Fatalf("AuditLegibility: %v", err)
	}

	for _, ds := range report.Dimensions {
		if ds.Score < 0 || ds.Score > 1 {
			t.Fatalf("dimension %q score = %f, want in [0, 1]", ds.Dimension.Name, ds.Score)
		}
	}

	if report.Overall < 0 || report.Overall > 1 {
		t.Fatalf("Overall = %f, want in [0, 1]", report.Overall)
	}
}

func TestAuditLegibilityNotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := AuditLegibility(f, &RepoProfile{Path: f})
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestAuditLegibilityEmptyRepoZeroScores(t *testing.T) {
	root := t.TempDir()
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	report, err := AuditLegibility(root, profile)
	if err != nil {
		t.Fatalf("AuditLegibility: %v", err)
	}

	for _, ds := range report.Dimensions {
		if ds.Score != 0 {
			t.Fatalf("empty repo: dimension %q score = %f, want 0", ds.Dimension.Name, ds.Score)
		}
	}

	if report.Overall != 0 {
		t.Fatalf("empty repo: Overall = %f, want 0", report.Overall)
	}
}

func TestWeightedOverall(t *testing.T) {
	tests := []struct {
		name   string
		scores []DimensionScore
		want   float64
	}{
		{
			name:   "no dimensions",
			scores: nil,
			want:   0,
		},
		{
			name: "all perfect",
			scores: func() []DimensionScore {
				dims := DefaultDimensions()
				scores := make([]DimensionScore, len(dims))
				for i, d := range dims {
					scores[i] = DimensionScore{Dimension: d, Score: 1.0}
				}
				return scores
			}(),
			want: 1.0,
		},
		{
			name: "all zero",
			scores: func() []DimensionScore {
				dims := DefaultDimensions()
				scores := make([]DimensionScore, len(dims))
				for i, d := range dims {
					scores[i] = DimensionScore{Dimension: d, Score: 0.0}
				}
				return scores
			}(),
			want: 0.0,
		},
		{
			name: "uniform half",
			scores: func() []DimensionScore {
				dims := DefaultDimensions()
				scores := make([]DimensionScore, len(dims))
				for i, d := range dims {
					scores[i] = DimensionScore{Dimension: d, Score: 0.5}
				}
				return scores
			}(),
			want: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &LegibilityReport{Dimensions: tt.scores}
			got := report.WeightedOverall()
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				t.Fatalf("WeightedOverall() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestDefaultDimensions(t *testing.T) {
	dims := DefaultDimensions()

	if len(dims) != 7 {
		t.Fatalf("expected 7 dimensions, got %d", len(dims))
	}

	totalWeight := 0.0
	for _, d := range dims {
		if d.Name == "" {
			t.Fatal("dimension has empty name")
		}
		if d.Description == "" {
			t.Fatalf("dimension %q has empty description", d.Name)
		}
		if d.Weight <= 0 || d.Weight > 1 {
			t.Fatalf("dimension %q weight = %f, want in (0, 1]", d.Name, d.Weight)
		}
		totalWeight += d.Weight
	}

	diff := totalWeight - 1.0
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.001 {
		t.Fatalf("total weight = %f, want 1.0", totalWeight)
	}
}

func TestGenerateAgentsMD(t *testing.T) {
	tests := []struct {
		name         string
		profile      *RepoProfile
		wantContains []string
	}{
		{
			name: "full profile",
			profile: &RepoProfile{
				Languages: []Language{
					{Name: "Go", FileCount: 10, Confidence: 0.8},
				},
				Frameworks: []Framework{
					{Name: "Go Modules", Language: "Go"},
				},
				BuildTools: []BuildTool{
					{Name: "Make", ConfigFile: "Makefile"},
				},
				EntryPoints: []EntryPoint{
					{Name: "make", Command: "make"},
				},
				TechStack: TechStack{
					Warnings: []Compatibility{
						{Technology: "Docker", Level: "moderate", Reason: "needs socket", Alternative: "use host"},
					},
				},
			},
			wantContains: []string{
				"# AGENTS.md",
				"Go",
				"Go Modules",
				"Make",
				"make",
				"Docker",
				"Entry Points",
			},
		},
		{
			name:    "empty profile",
			profile: &RepoProfile{},
			wantContains: []string{
				"# AGENTS.md",
			},
		},
		{
			name: "entry points present",
			profile: &RepoProfile{
				EntryPoints: []EntryPoint{
					{Name: "npm", Command: "npm start"},
					{Name: "go run", Command: "go run ."},
				},
			},
			wantContains: []string{
				"## Entry Points",
				"npm start",
				"go run .",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := GenerateAgentsMD(tt.profile)

			for _, want := range tt.wantContains {
				if !strings.Contains(md, want) {
					t.Fatalf("AGENTS.md missing %q.\nGot:\n%s", want, md)
				}
			}
		})
	}
}

func TestGenerateAgentsMDHeading(t *testing.T) {
	profile := &RepoProfile{}
	md := GenerateAgentsMD(profile)
	if !strings.HasPrefix(md, "# AGENTS.md") {
		t.Fatalf("expected AGENTS.md to start with heading, got: %s", md[:min(50, len(md))])
	}
}

func TestGenerateAgentsMDContainsEntryPoints(t *testing.T) {
	profile := &RepoProfile{
		EntryPoints: []EntryPoint{
			{Name: "make", Command: "make build"},
			{Name: "go test", Command: "go test ./..."},
		},
	}

	md := GenerateAgentsMD(profile)

	for _, ep := range profile.EntryPoints {
		if !strings.Contains(md, ep.Command) {
			t.Fatalf("AGENTS.md missing entry point command %q", ep.Command)
		}
	}
}

func TestClampScore(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}
	for _, tt := range tests {
		got := clampScore(tt.input)
		if got != tt.want {
			t.Fatalf("clampScore(%f) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestLegibilityDimensionGaps(t *testing.T) {
	root := t.TempDir()
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	report, err := AuditLegibility(root, profile)
	if err != nil {
		t.Fatalf("AuditLegibility: %v", err)
	}

	// Every dimension on an empty repo should have gaps.
	for _, ds := range report.Dimensions {
		if len(ds.Gaps) == 0 {
			t.Fatalf("empty repo: dimension %q should have gaps", ds.Dimension.Name)
		}
	}
}

func TestScoreValidationHarnessSkipsVendor(t *testing.T) {
	root := t.TempDir()

	// Create a test file inside node_modules/ only — no test files at root.
	nmDir := filepath.Join(root, "node_modules", "some-pkg")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "index.test.js"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	score, gaps, _ := scoreValidationHarness(root)

	if score != 0 {
		t.Fatalf("expected score 0 (vendored tests should be skipped), got %f", score)
	}
	foundTestGap := false
	for _, g := range gaps {
		if g == "no test files found" {
			foundTestGap = true
			break
		}
	}
	if !foundTestGap {
		t.Fatalf("expected 'no test files found' gap, got %v", gaps)
	}
}

func TestLegibilityAutoFixable(t *testing.T) {
	// Set up a repo where "Linting/Formatting" has a perfect score (no gaps,
	// not auto-fixable) and "Decision Records" has gaps (auto-fixable).
	root := setupRepo(t,
		[]string{".golangci.yml", ".editorconfig"},
		nil,
	)
	profile, err := AnalyzeRepo(root)
	if err != nil {
		t.Fatalf("AnalyzeRepo: %v", err)
	}

	report, err := AuditLegibility(root, profile)
	if err != nil {
		t.Fatalf("AuditLegibility: %v", err)
	}

	for _, ds := range report.Dimensions {
		switch ds.Dimension.Name {
		case "Linting/Formatting":
			// Full score — no gaps, not auto-fixable.
			if ds.AutoFixable {
				t.Fatalf("Linting/Formatting has no gaps but AutoFixable is true")
			}
			if len(ds.Gaps) != 0 {
				t.Fatalf("Linting/Formatting should have no gaps, got %v", ds.Gaps)
			}
		case "Decision Records":
			// Missing ADR dir and CHANGELOG — has gaps, auto-fixable.
			if !ds.AutoFixable {
				t.Fatalf("Decision Records has gaps but AutoFixable is false")
			}
			if len(ds.Gaps) == 0 {
				t.Fatal("Decision Records should have gaps on this repo")
			}
		}
	}
}

func TestWriteAgentsMD(t *testing.T) {
	dir := t.TempDir()
	profile := &RepoProfile{
		Languages: []Language{{Name: "Go", FileCount: 3, Confidence: 0.9}},
	}

	if err := WriteAgentsMD(profile, dir); err != nil {
		t.Fatalf("WriteAgentsMD: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}

	if !strings.HasPrefix(string(data), "# AGENTS.md") {
		t.Fatalf("AGENTS.md missing heading, got: %s", string(data[:min(50, len(data))]))
	}
}

func TestWriteAgentsMDSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "AGENTS.md")
	original := "# existing content\n"
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	profile := &RepoProfile{
		Languages: []Language{{Name: "Go", FileCount: 5, Confidence: 1.0}},
	}

	if err := WriteAgentsMD(profile, dir); err != nil {
		t.Fatalf("WriteAgentsMD: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != original {
		t.Fatalf("existing AGENTS.md was overwritten: got %q, want %q", string(data), original)
	}
}

func TestGenerateDocsStructure(t *testing.T) {
	dir := t.TempDir()
	profile := &RepoProfile{
		Languages:  []Language{{Name: "Go", FileCount: 2, Confidence: 0.8}},
		Frameworks: []Framework{{Name: "Go Modules", Language: "Go"}},
		EntryPoints: []EntryPoint{
			{Name: "make", Command: "make"},
		},
	}

	if err := GenerateDocsStructure(profile, dir); err != nil {
		t.Fatalf("GenerateDocsStructure: %v", err)
	}

	// Check architecture.md exists and has expected content.
	archData, err := os.ReadFile(filepath.Join(dir, "docs", "architecture.md"))
	if err != nil {
		t.Fatalf("read architecture.md: %v", err)
	}
	if !strings.Contains(string(archData), "# Architecture") {
		t.Fatal("architecture.md missing heading")
	}
	if !strings.Contains(string(archData), "Go") {
		t.Fatal("architecture.md missing language info")
	}

	// Check getting-started.md exists.
	gsData, err := os.ReadFile(filepath.Join(dir, "docs", "getting-started.md"))
	if err != nil {
		t.Fatalf("read getting-started.md: %v", err)
	}
	if !strings.Contains(string(gsData), "# Getting Started") {
		t.Fatal("getting-started.md missing heading")
	}

	// Check ADR directory and template.
	adrData, err := os.ReadFile(filepath.Join(dir, "docs", "adr", "0001-initial-setup.md"))
	if err != nil {
		t.Fatalf("read ADR template: %v", err)
	}
	if !strings.Contains(string(adrData), "ADR 0001") {
		t.Fatal("ADR template missing title")
	}
}

func TestGenerateDocsStructureSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	original := "# My custom architecture\n"
	archPath := filepath.Join(docsDir, "architecture.md")
	if err := os.WriteFile(archPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	profile := &RepoProfile{
		Languages: []Language{{Name: "Go", FileCount: 1, Confidence: 1.0}},
	}

	if err := GenerateDocsStructure(profile, dir); err != nil {
		t.Fatalf("GenerateDocsStructure: %v", err)
	}

	data, err := os.ReadFile(archPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != original {
		t.Fatalf("architecture.md was overwritten: got %q, want %q", string(data), original)
	}
}

func TestGenerateProgressFile(t *testing.T) {
	dir := t.TempDir()

	if err := GenerateProgressFile("mission-42", dir); err != nil {
		t.Fatalf("GenerateProgressFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "progress.json"))
	if err != nil {
		t.Fatalf("read progress.json: %v", err)
	}

	var result progressData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal progress.json: %v", err)
	}

	if result.MissionID != "mission-42" {
		t.Fatalf("mission_id = %q, want %q", result.MissionID, "mission-42")
	}
	if result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("items should be empty array, got %v", result.Items)
	}
	if result.UpdatedAt == "" {
		t.Fatal("updated_at should not be empty")
	}
}

func TestGenerateFeatureList(t *testing.T) {
	dir := t.TempDir()
	profile := &RepoProfile{
		EntryPoints: []EntryPoint{
			{Name: "make", Command: "make"},
			{Name: "go run", Command: "go run ."},
		},
		Frameworks: []Framework{
			{Name: "Go Modules", Language: "Go"},
		},
	}

	if err := GenerateFeatureList(profile, dir); err != nil {
		t.Fatalf("GenerateFeatureList: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "feature-list.json"))
	if err != nil {
		t.Fatalf("read feature-list.json: %v", err)
	}

	var features []feature
	if err := json.Unmarshal(data, &features); err != nil {
		t.Fatalf("unmarshal feature-list.json: %v", err)
	}

	if len(features) != 3 {
		t.Fatalf("expected 3 features (2 entry points + 1 framework), got %d", len(features))
	}

	// Check that entry points are present.
	foundMake := false
	for _, f := range features {
		if f.Name == "make" && f.Source == "entry-point" && f.Status == "pending" {
			foundMake = true
		}
	}
	if !foundMake {
		t.Fatal("missing 'make' entry-point feature")
	}

	// Check that framework is present.
	foundFramework := false
	for _, f := range features {
		if f.Name == "Go Modules" && f.Source == "framework" && f.Status == "pending" {
			foundFramework = true
		}
	}
	if !foundFramework {
		t.Fatal("missing 'Go Modules' framework feature")
	}
}

func TestBootstrapIntegration(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal repo structure.
	for _, f := range []string{"main.go", "go.mod", "Makefile"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("// placeholder"), 0o644); err != nil {
			t.Fatalf("write %q: %v", f, err)
		}
	}

	profile, err := Bootstrap(dir)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// AGENTS.md should exist.
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}

	// docs/ structure should exist.
	if _, err := os.Stat(filepath.Join(dir, "docs", "architecture.md")); err != nil {
		t.Fatalf("docs/architecture.md not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "getting-started.md")); err != nil {
		t.Fatalf("docs/getting-started.md not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "adr", "0001-initial-setup.md")); err != nil {
		t.Fatalf("docs/adr/0001-initial-setup.md not created: %v", err)
	}

	// feature-list.json should exist.
	if _, err := os.Stat(filepath.Join(dir, "feature-list.json")); err != nil {
		t.Fatalf("feature-list.json not created: %v", err)
	}
}

func TestDetectConventionFiles(t *testing.T) {
	root := t.TempDir()

	// Create a subdirectory with an eslint config.
	subDir := filepath.Join(root, "frontend")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, ".eslintrc.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	instructions := DetectConventionFiles(root)

	if len(instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d: %+v", len(instructions), instructions)
	}

	inst := instructions[0]
	if inst.Level != DirLevel {
		t.Fatalf("Level = %v, want DirLevel", inst.Level)
	}
	if inst.Path != "frontend" {
		t.Fatalf("Path = %q, want %q", inst.Path, "frontend")
	}
	if inst.Source != ".eslintrc.json" {
		t.Fatalf("Source = %q, want %q", inst.Source, ".eslintrc.json")
	}
	if !strings.Contains(inst.Content, "Linter") {
		t.Fatalf("Content = %q, want it to contain %q", inst.Content, "Linter")
	}
}

func TestDetectConventionFilesEmpty(t *testing.T) {
	root := t.TempDir()

	instructions := DetectConventionFiles(root)

	if instructions != nil {
		t.Fatalf("expected nil slice for empty repo, got %d instructions", len(instructions))
	}
}

func TestMergeInstructionsOverride(t *testing.T) {
	repo := []Instruction{
		{Level: RepoLevel, Path: "src", Source: ".eslintrc", Content: "repo-level linter"},
	}
	dir := []Instruction{
		{Level: DirLevel, Path: "src", Source: ".eslintrc", Content: "dir-level linter"},
	}

	result := MergeInstructions(nil, repo, dir)

	if len(result.Instructions) != 1 {
		t.Fatalf("expected 1 instruction after merge, got %d", len(result.Instructions))
	}
	if result.Instructions[0].Level != DirLevel {
		t.Fatalf("expected DirLevel to win, got %v", result.Instructions[0].Level)
	}
	if result.Instructions[0].Content != "dir-level linter" {
		t.Fatalf("Content = %q, want %q", result.Instructions[0].Content, "dir-level linter")
	}
}

func TestMergeInstructionsOrdering(t *testing.T) {
	org := []Instruction{
		{Level: OrgLevel, Path: "z-dir", Source: "alpha", Content: "org z"},
	}
	repo := []Instruction{
		{Level: RepoLevel, Path: "a-dir", Source: "beta", Content: "repo a"},
	}
	dir := []Instruction{
		{Level: DirLevel, Path: "a-dir", Source: "alpha", Content: "dir a-alpha"},
		{Level: DirLevel, Path: "m-dir", Source: "gamma", Content: "dir m"},
	}

	result := MergeInstructions(org, repo, dir)

	if len(result.Instructions) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(result.Instructions))
	}

	// Verify sorted by Path then Source.
	for i := 1; i < len(result.Instructions); i++ {
		prev := result.Instructions[i-1]
		curr := result.Instructions[i]
		if prev.Path > curr.Path || (prev.Path == curr.Path && prev.Source > curr.Source) {
			t.Fatalf("instructions not sorted: [%d]={Path:%q,Source:%q} before [%d]={Path:%q,Source:%q}",
				i-1, prev.Path, prev.Source, i, curr.Path, curr.Source)
		}
	}
}

func TestWriteDirInstructionsNeverOverwrites(t *testing.T) {
	root := t.TempDir()

	// Create pre-existing AGENTS.md in a subdirectory.
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := "# My custom AGENTS.md\n"
	if err := os.WriteFile(filepath.Join(subDir, "AGENTS.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	instrSet := InstructionSet{
		Instructions: []Instruction{
			{Level: DirLevel, Path: "sub", Source: ".eslintrc", Content: "Linter config: .eslintrc"},
		},
	}

	if err := WriteDirInstructions(instrSet, root); err != nil {
		t.Fatalf("WriteDirInstructions: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(subDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != original {
		t.Fatalf("AGENTS.md was overwritten: got %q, want %q", string(data), original)
	}
}

func TestWriteDirInstructionsCreatesNew(t *testing.T) {
	root := t.TempDir()

	instrSet := InstructionSet{
		Instructions: []Instruction{
			{Level: DirLevel, Path: "lib", Source: ".prettierrc", Content: "Formatter config: .prettierrc"},
		},
	}

	if err := WriteDirInstructions(instrSet, root); err != nil {
		t.Fatalf("WriteDirInstructions: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "lib", "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "## Directory Instructions") {
		t.Fatalf("missing heading in generated AGENTS.md: %s", content)
	}
	if !strings.Contains(content, "Formatter config: .prettierrc") {
		t.Fatalf("missing convention entry in generated AGENTS.md: %s", content)
	}
}

func TestWriteDirInstructionsSkipsRoot(t *testing.T) {
	root := t.TempDir()

	instrSet := InstructionSet{
		Instructions: []Instruction{
			{Level: DirLevel, Path: ".", Source: ".eslintrc", Content: "Linter config: .eslintrc"},
		},
	}

	if err := WriteDirInstructions(instrSet, root); err != nil {
		t.Fatalf("WriteDirInstructions: %v", err)
	}

	// Root AGENTS.md should NOT be created by WriteDirInstructions.
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Fatal("WriteDirInstructions should not create AGENTS.md at repo root")
	}
}

func TestDetectInstructionHierarchy(t *testing.T) {
	root := t.TempDir()

	// Create a convention file.
	subDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, ".golangci.yml"), []byte("# config"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	profile := &RepoProfile{
		Languages: []Language{
			{Name: "Go", FileCount: 5},
		},
		Frameworks: []Framework{
			{Name: "Go Modules", Language: "Go"},
		},
	}

	instrSet := DetectInstructionHierarchy(root, profile)

	// Should have repo-level from profile + dir-level from convention file.
	if len(instrSet.Instructions) < 2 {
		t.Fatalf("expected at least 2 instructions, got %d: %+v", len(instrSet.Instructions), instrSet.Instructions)
	}

	// Check ordering: sorted by Path then Source.
	for i := 1; i < len(instrSet.Instructions); i++ {
		prev := instrSet.Instructions[i-1]
		curr := instrSet.Instructions[i]
		if prev.Path > curr.Path || (prev.Path == curr.Path && prev.Source > curr.Source) {
			t.Fatalf("instructions not sorted at index %d", i)
		}
	}

	// Verify we have both repo-level and dir-level instructions.
	hasRepo := false
	hasDir := false
	for _, inst := range instrSet.Instructions {
		if inst.Level == RepoLevel {
			hasRepo = true
		}
		if inst.Level == DirLevel {
			hasDir = true
		}
	}
	if !hasRepo {
		t.Fatal("expected repo-level instructions from profile")
	}
	if !hasDir {
		t.Fatal("expected dir-level instructions from convention files")
	}
}

func TestInstructionLevelString(t *testing.T) {
	tests := []struct {
		level InstructionLevel
		want  string
	}{
		{OrgLevel, "org"},
		{RepoLevel, "repo"},
		{DirLevel, "dir"},
		{InstructionLevel(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Fatalf("InstructionLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}
