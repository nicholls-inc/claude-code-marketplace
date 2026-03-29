package bootstrap

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// propTempDir creates a temporary directory for use inside rapid.Check
// and registers cleanup via t.Cleanup.
func propTempDir(t *rapid.T) string {
	dir, err := os.MkdirTemp("", "bootstrap-prop-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestPropDimensionScoresInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random file sets.
		fileChoices := []string{
			"README.md", "go.mod", "Makefile", "AGENTS.md", "main.go",
			"main_test.go", ".golangci.yml", ".editorconfig", "ARCHITECTURE.md",
			"CONTRIBUTING.md", "CHANGELOG.md", "package.json", "Cargo.toml",
			"Dockerfile", "requirements.txt", "app.py", "index.ts",
			".pre-commit-config.yaml",
		}
		dirChoices := []string{
			"scripts", "internal", "docs", "docs/adr", ".github/workflows",
			"src", "bin", "cmd", "pkg",
		}

		numFiles := rapid.IntRange(0, len(fileChoices)).Draw(t, "numFiles")
		numDirs := rapid.IntRange(0, len(dirChoices)).Draw(t, "numDirs")

		var files []string
		used := make(map[int]bool)
		for i := 0; i < numFiles; i++ {
			idx := rapid.IntRange(0, len(fileChoices)-1).Draw(t, "fileIdx")
			if !used[idx] {
				files = append(files, fileChoices[idx])
				used[idx] = true
			}
		}

		var dirs []string
		usedDirs := make(map[int]bool)
		for i := 0; i < numDirs; i++ {
			idx := rapid.IntRange(0, len(dirChoices)-1).Draw(t, "dirIdx")
			if !usedDirs[idx] {
				dirs = append(dirs, dirChoices[idx])
				usedDirs[idx] = true
			}
		}

		root := propTempDir(t)
		for _, d := range dirs {
			p := filepath.Join(root, d)
			_ = os.MkdirAll(p, 0o755)
		}
		for _, f := range files {
			p := filepath.Join(root, f)
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
			_ = os.WriteFile(p, []byte("// gen"), 0o644)
		}

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
				t.Fatalf("dimension %q score = %f, out of [0, 1]", ds.Dimension.Name, ds.Score)
			}
		}
	})
}

func TestPropWeightedOverallInBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dims := DefaultDimensions()
		scores := make([]DimensionScore, len(dims))
		for i, d := range dims {
			s := rapid.Float64Range(0, 1).Draw(t, "score")
			scores[i] = DimensionScore{Dimension: d, Score: s}
		}

		report := &LegibilityReport{Dimensions: scores}
		overall := report.WeightedOverall()

		if overall < 0 || overall > 1 {
			t.Fatalf("WeightedOverall = %f, out of [0, 1]", overall)
		}
	})
}

func TestPropWeightedOverallIsWeightedAverage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dims := DefaultDimensions()
		scores := make([]DimensionScore, len(dims))
		for i, d := range dims {
			s := rapid.Float64Range(0, 1).Draw(t, "score")
			scores[i] = DimensionScore{Dimension: d, Score: s}
		}

		report := &LegibilityReport{Dimensions: scores}
		overall := report.WeightedOverall()

		// Compute expected weighted average.
		totalWeight := 0.0
		weightedSum := 0.0
		for _, ds := range scores {
			weightedSum += ds.Score * ds.Dimension.Weight
			totalWeight += ds.Dimension.Weight
		}
		expected := 0.0
		if totalWeight > 0 {
			expected = weightedSum / totalWeight
		}

		if math.Abs(overall-expected) > 1e-10 {
			t.Fatalf("WeightedOverall = %f, expected %f", overall, expected)
		}
	})
}

func TestPropEmptyRepoSafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root := propTempDir(t)

		profile, err := AnalyzeRepo(root)
		if err != nil {
			t.Fatalf("AnalyzeRepo: %v", err)
		}

		// Empty repo should not error.
		if len(profile.Languages) != 0 {
			t.Fatalf("expected no languages, got %d", len(profile.Languages))
		}

		report, err := AuditLegibility(root, profile)
		if err != nil {
			t.Fatalf("AuditLegibility: %v", err)
		}

		// All scores should be zero.
		for _, ds := range report.Dimensions {
			if ds.Score != 0 {
				t.Fatalf("empty repo: dimension %q score = %f, want 0", ds.Dimension.Name, ds.Score)
			}
		}

		if report.Overall != 0 {
			t.Fatalf("empty repo: Overall = %f, want 0", report.Overall)
		}

		// Should have exactly 7 dimensions.
		if len(report.Dimensions) != 7 {
			t.Fatalf("expected 7 dimensions, got %d", len(report.Dimensions))
		}
	})
}

func TestPropAgentsMDContainsHeading(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numLangs := rapid.IntRange(0, 5).Draw(t, "numLangs")
		numEPs := rapid.IntRange(0, 5).Draw(t, "numEPs")

		var langs []Language
		for i := 0; i < numLangs; i++ {
			langs = append(langs, Language{
				Name:       rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "langName"),
				FileCount:  rapid.IntRange(1, 100).Draw(t, "fileCount"),
				Confidence: rapid.Float64Range(0, 1).Draw(t, "confidence"),
			})
		}

		var eps []EntryPoint
		for i := 0; i < numEPs; i++ {
			eps = append(eps, EntryPoint{
				Name:    rapid.StringMatching(`[a-z]+`).Draw(t, "epName"),
				Command: rapid.StringMatching(`[a-z]+ [a-z./]+`).Draw(t, "epCmd"),
			})
		}

		profile := &RepoProfile{
			Languages:   langs,
			EntryPoints: eps,
		}

		md := GenerateAgentsMD(profile)

		if !strings.HasPrefix(md, "# AGENTS.md") {
			t.Fatalf("AGENTS.md missing heading, starts with: %q", md[:min(50, len(md))])
		}

		// All entry point commands should appear.
		for _, ep := range eps {
			if !strings.Contains(md, ep.Command) {
				t.Fatalf("AGENTS.md missing entry point command %q", ep.Command)
			}
		}
	})
}

func TestPropClampScoreIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Float64Range(-10, 10).Draw(t, "value")
		clamped := clampScore(v)
		doubleClamped := clampScore(clamped)

		if clamped != doubleClamped {
			t.Fatalf("clampScore not idempotent: clampScore(%f) = %f, clampScore(%f) = %f", v, clamped, clamped, doubleClamped)
		}

		if clamped < 0 || clamped > 1 {
			t.Fatalf("clampScore(%f) = %f, out of [0, 1]", v, clamped)
		}
	})
}

func TestPropDetectedLanguagesMatchExtensions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		extOptions := []string{".go", ".py", ".js", ".ts", ".rs", ".java", ".rb"}
		numFiles := rapid.IntRange(0, 20).Draw(t, "numFiles")

		root := propTempDir(t)
		extPresent := make(map[string]bool)
		for i := 0; i < numFiles; i++ {
			ext := extOptions[rapid.IntRange(0, len(extOptions)-1).Draw(t, "extIdx")]
			fname := filepath.Join(root, rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "fname")+ext)
			_ = os.WriteFile(fname, []byte("// gen"), 0o644)
			extPresent[ext] = true
		}

		langs := DetectLanguages(root)

		for _, lang := range langs {
			for _, ext := range lang.FileExtensions {
				if !extPresent[ext] {
					t.Fatalf("language %q claims extension %q but no such file exists", lang.Name, ext)
				}
			}
		}
	})
}

func TestPropAllSevenDimensionsAlwaysPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fileChoices := []string{
			"README.md", "go.mod", "Makefile", "main.go", "Dockerfile",
			"package.json", "CHANGELOG.md", "AGENTS.md",
		}

		numFiles := rapid.IntRange(0, len(fileChoices)).Draw(t, "numFiles")
		var files []string
		used := make(map[int]bool)
		for i := 0; i < numFiles; i++ {
			idx := rapid.IntRange(0, len(fileChoices)-1).Draw(t, "idx")
			if !used[idx] {
				files = append(files, fileChoices[idx])
				used[idx] = true
			}
		}

		root := propTempDir(t)
		for _, f := range files {
			p := filepath.Join(root, f)
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
			_ = os.WriteFile(p, []byte("// gen"), 0o644)
		}

		profile, err := AnalyzeRepo(root)
		if err != nil {
			t.Fatalf("AnalyzeRepo: %v", err)
		}

		report, err := AuditLegibility(root, profile)
		if err != nil {
			t.Fatalf("AuditLegibility: %v", err)
		}

		if len(report.Dimensions) != 7 {
			t.Fatalf("expected 7 dimensions, got %d", len(report.Dimensions))
		}
	})
}

func TestPropWriteAgentsMDAlwaysContainsHeading(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numLangs := rapid.IntRange(0, 5).Draw(t, "numLangs")
		numEPs := rapid.IntRange(0, 5).Draw(t, "numEPs")
		numFWs := rapid.IntRange(0, 3).Draw(t, "numFWs")

		var langs []Language
		for i := 0; i < numLangs; i++ {
			langs = append(langs, Language{
				Name:       rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "langName"),
				FileCount:  rapid.IntRange(1, 100).Draw(t, "fileCount"),
				Confidence: rapid.Float64Range(0, 1).Draw(t, "confidence"),
			})
		}

		var eps []EntryPoint
		for i := 0; i < numEPs; i++ {
			eps = append(eps, EntryPoint{
				Name:    rapid.StringMatching(`[a-z]+`).Draw(t, "epName"),
				Command: rapid.StringMatching(`[a-z]+ [a-z./]+`).Draw(t, "epCmd"),
			})
		}

		var fws []Framework
		for i := 0; i < numFWs; i++ {
			fws = append(fws, Framework{
				Name:     rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "fwName"),
				Language: rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "fwLang"),
			})
		}

		profile := &RepoProfile{
			Languages:   langs,
			EntryPoints: eps,
			Frameworks:  fws,
		}

		dir := propTempDir(t)
		if err := WriteAgentsMD(profile, dir); err != nil {
			t.Fatalf("WriteAgentsMD: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
		if err != nil {
			t.Fatalf("read AGENTS.md: %v", err)
		}

		if !strings.HasPrefix(string(data), "# AGENTS.md") {
			t.Fatalf("AGENTS.md missing heading, starts with: %q", string(data[:min(50, len(data))]))
		}
	})
}

func TestPropBootstrapNeverOverwrites(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fileChoices := []string{
			"main.go", "go.mod", "Makefile", "package.json", "app.py",
		}

		numFiles := rapid.IntRange(1, len(fileChoices)).Draw(t, "numFiles")
		var files []string
		used := make(map[int]bool)
		for i := 0; i < numFiles; i++ {
			idx := rapid.IntRange(0, len(fileChoices)-1).Draw(t, "idx")
			if !used[idx] {
				files = append(files, fileChoices[idx])
				used[idx] = true
			}
		}

		root := propTempDir(t)
		for _, f := range files {
			p := filepath.Join(root, f)
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
			_ = os.WriteFile(p, []byte("// gen"), 0o644)
		}

		// First run.
		_, err := Bootstrap(root)
		if err != nil {
			t.Fatalf("first Bootstrap: %v", err)
		}

		// Snapshot generated files.
		snapshots := make(map[string][]byte)
		generatedFiles := []string{
			"AGENTS.md",
			"feature-list.json",
			"docs/architecture.md",
			"docs/getting-started.md",
			"docs/adr/0001-initial-setup.md",
		}
		for _, f := range generatedFiles {
			p := filepath.Join(root, f)
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			snapshots[f] = data
		}

		// Second run.
		_, err = Bootstrap(root)
		if err != nil {
			t.Fatalf("second Bootstrap: %v", err)
		}

		// Verify no files changed.
		for f, original := range snapshots {
			p := filepath.Join(root, f)
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("file %q disappeared after second run: %v", f, err)
			}
			if string(data) != string(original) {
				t.Fatalf("file %q was overwritten by second Bootstrap run", f)
			}
		}
	})
}
