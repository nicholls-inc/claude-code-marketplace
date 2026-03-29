package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Language describes a programming language detected in the repository.
type Language struct {
	Name           string
	FileExtensions []string
	FileCount      int
	Confidence     float64
}

// Framework describes a framework detected in the repository.
type Framework struct {
	Name       string
	Language   string
	Indicators []string
	Confidence float64
}

// BuildTool describes a build tool detected via its config file.
type BuildTool struct {
	Name       string
	ConfigFile string
	Commands   map[string]string
}

// Technology describes a technology detected in the repository.
type Technology struct {
	Name       string
	Category   string
	Confidence float64
}

// Compatibility describes agent compatibility for a detected technology.
type Compatibility struct {
	Technology  string
	Level       string // "good", "moderate", "poor"
	Reason      string
	Alternative string
}

// TechStack combines detected technologies with agent-compatibility warnings.
type TechStack struct {
	Detected []Technology
	Warnings []Compatibility
}

// EntryPoint describes a runnable entry point discovered in the repo.
type EntryPoint struct {
	Name    string
	Command string
	// Exists indicates the entry point file was found on disk.
	Exists bool   `json:"exists"`
	Error  string
}

// RepoProfile aggregates analysis results for a repository.
type RepoProfile struct {
	Path        string
	Languages   []Language
	Frameworks  []Framework
	BuildTools  []BuildTool
	TechStack   TechStack
	EntryPoints []EntryPoint
	AnalyzedAt  time.Time
}

// Dimension describes a legibility audit dimension.
type Dimension struct {
	Name        string
	Description string
	Weight      float64
}

// DimensionScore records the audit result for a single dimension.
type DimensionScore struct {
	Dimension   Dimension
	Score       float64
	Gaps        []string
	AutoFixable bool
}

// LegibilityReport contains the results of a legibility audit.
type LegibilityReport struct {
	Dimensions []DimensionScore
	Overall    float64
	RepoPath   string
	AuditedAt  time.Time
}

// knownLanguages maps file extensions to language names.
var knownLanguages = map[string]string{
	".go":   "Go",
	".py":   "Python",
	".js":   "JavaScript",
	".ts":   "TypeScript",
	".tsx":  "TypeScript",
	".jsx":  "JavaScript",
	".rs":   "Rust",
	".java": "Java",
	".rb":   "Ruby",
	".c":    "C",
	".cpp":  "C++",
	".cs":   "C#",
	".php":  "PHP",
	".swift": "Swift",
	".kt":   "Kotlin",
}

// frameworkIndicators maps config/marker files to framework definitions.
var frameworkIndicators = []struct {
	File      string
	Name      string
	Language  string
	Indicator string
}{
	{"go.mod", "Go Modules", "Go", "go.mod"},
	{"package.json", "Node.js", "JavaScript", "package.json"},
	{"requirements.txt", "pip", "Python", "requirements.txt"},
	{"setup.py", "setuptools", "Python", "setup.py"},
	{"pyproject.toml", "Python Project", "Python", "pyproject.toml"},
	{"Cargo.toml", "Cargo", "Rust", "Cargo.toml"},
	{"pom.xml", "Maven", "Java", "pom.xml"},
	{"build.gradle", "Gradle", "Java", "build.gradle"},
	{"Gemfile", "Bundler", "Ruby", "Gemfile"},
	{"composer.json", "Composer", "PHP", "composer.json"},
	{"Package.swift", "Swift PM", "Swift", "Package.swift"},
}

// buildToolDefs maps config files to build tool definitions.
var buildToolDefs = []struct {
	ConfigFile string
	Name       string
	Commands   map[string]string
}{
	{"Makefile", "Make", map[string]string{"build": "make", "test": "make test"}},
	{"Dockerfile", "Docker", map[string]string{"build": "docker build ."}},
	{"docker-compose.yml", "Docker Compose", map[string]string{"up": "docker-compose up"}},
	{"docker-compose.yaml", "Docker Compose", map[string]string{"up": "docker-compose up"}},
	{"Justfile", "Just", map[string]string{"default": "just"}},
	{"Taskfile.yml", "Task", map[string]string{"default": "task"}},
	{"CMakeLists.txt", "CMake", map[string]string{"build": "cmake --build ."}},
	{".goreleaser.yml", "GoReleaser", map[string]string{"release": "goreleaser release"}},
	{".goreleaser.yaml", "GoReleaser", map[string]string{"release": "goreleaser release"}},
}

// entryPointDefs maps files to entry point definitions.
var entryPointDefs = []struct {
	File    string
	Name    string
	Command string
}{
	{"Makefile", "make", "make"},
	{"package.json", "npm", "npm start"},
	{"main.go", "go run", "go run ."},
	{"cmd/main.go", "go run cmd", "go run ./cmd"},
	{"manage.py", "django", "python manage.py runserver"},
	{"app.py", "python app", "python app.py"},
	{"Cargo.toml", "cargo run", "cargo run"},
}

// techWarnings maps technology names to agent-compatibility assessments.
var techWarnings = map[string]Compatibility{
	"Docker": {
		Technology:  "Docker",
		Level:       "moderate",
		Reason:      "Container operations require Docker socket access",
		Alternative: "Consider using host-native build commands",
	},
	"Kubernetes": {
		Technology:  "Kubernetes",
		Level:       "poor",
		Reason:      "Cluster operations require credentials and network access",
		Alternative: "Use local development with kind or minikube",
	},
	"Terraform": {
		Technology:  "Terraform",
		Level:       "poor",
		Reason:      "Infrastructure provisioning requires cloud credentials",
		Alternative: "Use plan-only mode for validation",
	},
}

// AnalyzeRepo performs a full analysis of the repository at path.
func AnalyzeRepo(path string) (*RepoProfile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat repo path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", path)
	}

	profile := &RepoProfile{
		Path:       path,
		AnalyzedAt: time.Now(),
	}

	profile.Languages = DetectLanguages(path)
	profile.Frameworks = DetectFrameworks(path)
	profile.BuildTools = DetectBuildTools(path)
	profile.EntryPoints = DiscoverEntryPoints(path)
	profile.TechStack = DetectTechStack(profile)

	return profile, nil
}

// DetectLanguages scans the directory tree and counts files by extension.
func DetectLanguages(path string) []Language {
	extCounts := make(map[string]int)
	total := 0

	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isSkippableDir(filepath.Base(p)) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if _, ok := knownLanguages[ext]; ok {
			extCounts[ext]++
			total++
		}
		return nil
	})

	// Group by language name.
	langFiles := make(map[string][]string)
	langCounts := make(map[string]int)
	for ext, count := range extCounts {
		name := knownLanguages[ext]
		langFiles[name] = append(langFiles[name], ext)
		langCounts[name] += count
	}

	var langs []Language
	for name, exts := range langFiles {
		count := langCounts[name]
		confidence := 0.0
		if total > 0 {
			confidence = float64(count) / float64(total)
		}
		sort.Strings(exts)
		langs = append(langs, Language{
			Name:           name,
			FileExtensions: exts,
			FileCount:      count,
			Confidence:     confidence,
		})
	}

	sort.Slice(langs, func(i, j int) bool {
		return langs[i].FileCount > langs[j].FileCount
	})

	return langs
}

// DetectFrameworks checks for framework indicator files at the repo root.
func DetectFrameworks(path string) []Framework {
	var frameworks []Framework
	for _, fi := range frameworkIndicators {
		target := filepath.Join(path, fi.File)
		if _, err := os.Stat(target); err == nil {
			frameworks = append(frameworks, Framework{
				Name:       fi.Name,
				Language:   fi.Language,
				Indicators: []string{fi.Indicator},
				Confidence: 0.9,
			})
		}
	}
	return frameworks
}

// DetectBuildTools checks for build tool config files at the repo root.
func DetectBuildTools(path string) []BuildTool {
	var tools []BuildTool
	seen := make(map[string]bool)
	for _, bt := range buildToolDefs {
		target := filepath.Join(path, bt.ConfigFile)
		if _, err := os.Stat(target); err == nil {
			if seen[bt.Name] {
				continue
			}
			seen[bt.Name] = true
			cmds := make(map[string]string, len(bt.Commands))
			for k, v := range bt.Commands {
				cmds[k] = v
			}
			tools = append(tools, BuildTool{
				Name:       bt.Name,
				ConfigFile: bt.ConfigFile,
				Commands:   cmds,
			})
		}
	}
	return tools
}

// DiscoverEntryPoints checks for known entry point files.
func DiscoverEntryPoints(path string) []EntryPoint {
	var eps []EntryPoint
	for _, ep := range entryPointDefs {
		target := filepath.Join(path, ep.File)
		if _, err := os.Stat(target); err == nil {
			eps = append(eps, EntryPoint{
				Name:    ep.Name,
				Command: ep.Command,
				Exists:  true,
			})
		}
	}
	return eps
}

// DetectTechStack determines technologies and their agent-compatibility.
func DetectTechStack(profile *RepoProfile) TechStack {
	var stack TechStack

	for _, bt := range profile.BuildTools {
		tech := Technology{
			Name:       bt.Name,
			Category:   "build",
			Confidence: 0.9,
		}
		stack.Detected = append(stack.Detected, tech)
		if w, ok := techWarnings[bt.Name]; ok {
			stack.Warnings = append(stack.Warnings, w)
		}
	}

	for _, fw := range profile.Frameworks {
		tech := Technology{
			Name:       fw.Name,
			Category:   "framework",
			Confidence: fw.Confidence,
		}
		stack.Detected = append(stack.Detected, tech)
		if w, ok := techWarnings[fw.Name]; ok {
			stack.Warnings = append(stack.Warnings, w)
		}
	}

	// Check for infrastructure-as-code files.
	infraFiles := map[string]Technology{
		"main.tf":        {Name: "Terraform", Category: "infrastructure", Confidence: 0.95},
		"k8s":            {Name: "Kubernetes", Category: "infrastructure", Confidence: 0.8},
		"kubernetes":     {Name: "Kubernetes", Category: "infrastructure", Confidence: 0.8},
		".github/workflows": {Name: "GitHub Actions", Category: "ci", Confidence: 0.95},
	}

	for file, tech := range infraFiles {
		target := filepath.Join(profile.Path, file)
		if _, err := os.Stat(target); err == nil {
			stack.Detected = append(stack.Detected, tech)
			if w, ok := techWarnings[tech.Name]; ok {
				stack.Warnings = append(stack.Warnings, w)
			}
		}
	}

	return stack
}

// DefaultDimensions returns the 7 legibility audit dimensions.
func DefaultDimensions() []Dimension {
	return []Dimension{
		{
			Name:        "Bootstrap Self-Sufficiency",
			Description: "Can the agent bootstrap the project without human help?",
			Weight:      0.20,
		},
		{
			Name:        "Task Entry Points",
			Description: "Are there clear, documented entry points for common tasks?",
			Weight:      0.15,
		},
		{
			Name:        "Validation Harness",
			Description: "Can the agent validate its own changes (tests, linters)?",
			Weight:      0.20,
		},
		{
			Name:        "Linting/Formatting",
			Description: "Are linting and formatting tools configured and documented?",
			Weight:      0.10,
		},
		{
			Name:        "Codebase Map",
			Description: "Is there a high-level map of the codebase structure?",
			Weight:      0.15,
		},
		{
			Name:        "Doc Structure",
			Description: "Is documentation well-organized and up to date?",
			Weight:      0.10,
		},
		{
			Name:        "Decision Records",
			Description: "Are architectural decisions recorded (ADRs or similar)?",
			Weight:      0.10,
		},
	}
}

// AuditLegibility evaluates the repository across the 7 legibility dimensions.
func AuditLegibility(path string, profile *RepoProfile) (*LegibilityReport, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat repo path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", path)
	}

	dims := DefaultDimensions()
	scores := make([]DimensionScore, len(dims))

	for i, dim := range dims {
		score, gaps, autoFixable := scoreDimension(path, profile, dim.Name)
		scores[i] = DimensionScore{
			Dimension:   dim,
			Score:       score,
			Gaps:        gaps,
			AutoFixable: autoFixable,
		}
	}

	report := &LegibilityReport{
		Dimensions: scores,
		RepoPath:   path,
		AuditedAt:  time.Now(),
	}
	report.Overall = report.WeightedOverall()

	return report, nil
}

// WeightedOverall computes the weighted average of dimension scores.
func (r *LegibilityReport) WeightedOverall() float64 {
	totalWeight := 0.0
	weightedSum := 0.0
	for _, ds := range r.Dimensions {
		weightedSum += ds.Score * ds.Dimension.Weight
		totalWeight += ds.Dimension.Weight
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// scoreDimension evaluates a single dimension and returns (score, gaps, autoFixable).
func scoreDimension(path string, profile *RepoProfile, dimName string) (float64, []string, bool) {
	switch dimName {
	case "Bootstrap Self-Sufficiency":
		return scoreBootstrapSelfSufficiency(path, profile)
	case "Task Entry Points":
		return scoreTaskEntryPoints(path, profile)
	case "Validation Harness":
		return scoreValidationHarness(path)
	case "Linting/Formatting":
		return scoreLintingFormatting(path)
	case "Codebase Map":
		return scoreCodebaseMap(path)
	case "Doc Structure":
		return scoreDocStructure(path)
	case "Decision Records":
		return scoreDecisionRecords(path)
	default:
		return 0, []string{"unknown dimension"}, false
	}
}

func scoreBootstrapSelfSufficiency(path string, profile *RepoProfile) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if anyFileExists(path, []string{"README.md", "readme.md", "README"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no README found")
	}

	if anyFileExists(path, []string{"go.mod", "package.json", "requirements.txt", "Cargo.toml", "pyproject.toml", "Gemfile", "pom.xml"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no dependency manifest found")
	}

	if len(profile.BuildTools) > 0 {
		score += 0.2
	} else {
		gaps = append(gaps, "no build tool detected")
	}

	if anyFileExists(path, []string{"AGENTS.md", "CLAUDE.md", ".claude/agents.md"}) {
		score += 0.2
	} else {
		gaps = append(gaps, "no AGENTS.md or CLAUDE.md found")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreTaskEntryPoints(path string, profile *RepoProfile) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if len(profile.EntryPoints) > 0 {
		score += 0.5
	} else {
		gaps = append(gaps, "no entry points discovered")
	}

	if anyDirExists(path, []string{"scripts", "bin"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no scripts/ or bin/ directory")
	}

	if fileExists(path, "Makefile") {
		score += 0.2
	} else {
		gaps = append(gaps, "no Makefile with targets")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreValidationHarness(path string) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	hasTests := false
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isSkippableDir(filepath.Base(p)) {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(p)
		if strings.HasSuffix(base, "_test.go") ||
			strings.HasPrefix(base, "test_") ||
			strings.HasSuffix(base, ".test.js") ||
			strings.HasSuffix(base, ".test.ts") ||
			strings.HasSuffix(base, "_test.py") ||
			strings.HasSuffix(base, ".spec.js") ||
			strings.HasSuffix(base, ".spec.ts") {
			hasTests = true
			return filepath.SkipAll
		}
		return nil
	})

	if hasTests {
		score += 0.4
	} else {
		gaps = append(gaps, "no test files found")
	}

	if anyExists(path, []string{".github/workflows", ".gitlab-ci.yml", "Jenkinsfile", ".circleci", ".travis.yml"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no CI configuration found")
	}

	if anyExists(path, []string{".pre-commit-config.yaml", ".husky", ".git/hooks/pre-commit"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no pre-commit hooks configured")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreLintingFormatting(path string) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if anyFileExists(path, []string{
		".eslintrc", ".eslintrc.js", ".eslintrc.json", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs",
		".golangci.yml", ".golangci.yaml",
		".flake8", ".pylintrc", "pyproject.toml",
		".rubocop.yml",
		"rustfmt.toml", ".rustfmt.toml",
	}) {
		score += 0.5
	} else {
		gaps = append(gaps, "no linter configuration found")
	}

	if anyFileExists(path, []string{
		".prettierrc", ".prettierrc.json", ".prettierrc.yml", ".prettierrc.js",
		".editorconfig",
		"rustfmt.toml", ".rustfmt.toml",
		".clang-format",
	}) {
		score += 0.5
	} else {
		gaps = append(gaps, "no formatter configuration found")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreCodebaseMap(path string) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if anyFileExists(path, []string{
		"ARCHITECTURE.md", "architecture.md",
		"docs/architecture.md", "docs/ARCHITECTURE.md",
		"AGENTS.md", "CLAUDE.md",
	}) {
		score += 0.5
	} else {
		gaps = append(gaps, "no architecture documentation found")
	}

	if anyDirExists(path, []string{"src", "lib", "internal", "pkg", "cmd"}) {
		score += 0.5
	} else {
		gaps = append(gaps, "no organized directory structure (src/, lib/, internal/, pkg/, cmd/)")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreDocStructure(path string) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if anyFileExists(path, []string{"README.md", "readme.md", "README"}) {
		score += 0.4
	} else {
		gaps = append(gaps, "no README found")
	}

	if anyDirExists(path, []string{"docs", "doc"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no docs/ directory")
	}

	if anyFileExists(path, []string{"CONTRIBUTING.md", "contributing.md"}) {
		score += 0.3
	} else {
		gaps = append(gaps, "no CONTRIBUTING.md found")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

func scoreDecisionRecords(path string) (float64, []string, bool) {
	score := 0.0
	var gaps []string

	if anyDirExists(path, []string{"docs/adr", "docs/adrs", "adr", "adrs", "docs/decisions", "decisions"}) {
		score += 0.6
	} else {
		gaps = append(gaps, "no ADR directory found")
	}

	if anyFileExists(path, []string{"CHANGELOG.md", "changelog.md", "CHANGES.md"}) {
		score += 0.4
	} else {
		gaps = append(gaps, "no CHANGELOG found")
	}

	return clampScore(score), gaps, len(gaps) > 0
}

// GenerateAgentsMD produces an AGENTS.md string based on the repo profile.
func GenerateAgentsMD(profile *RepoProfile) string {
	var sb strings.Builder

	sb.WriteString("# AGENTS.md\n\n")
	sb.WriteString("Auto-generated agent harness configuration.\n\n")

	// Languages section.
	if len(profile.Languages) > 0 {
		sb.WriteString("## Languages\n\n")
		for _, lang := range profile.Languages {
			sb.WriteString(fmt.Sprintf("- **%s** (%d files, %.0f%% confidence)\n",
				lang.Name, lang.FileCount, lang.Confidence*100))
		}
		sb.WriteString("\n")
	}

	// Frameworks section.
	if len(profile.Frameworks) > 0 {
		sb.WriteString("## Frameworks\n\n")
		for _, fw := range profile.Frameworks {
			sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", fw.Name, fw.Language))
		}
		sb.WriteString("\n")
	}

	// Build Tools section.
	if len(profile.BuildTools) > 0 {
		sb.WriteString("## Build Tools\n\n")
		for _, bt := range profile.BuildTools {
			sb.WriteString(fmt.Sprintf("- **%s** (config: `%s`)\n", bt.Name, bt.ConfigFile))
		}
		sb.WriteString("\n")
	}

	// Entry Points section.
	if len(profile.EntryPoints) > 0 {
		sb.WriteString("## Entry Points\n\n")
		for _, ep := range profile.EntryPoints {
			sb.WriteString(fmt.Sprintf("- **%s**: `%s`\n", ep.Name, ep.Command))
		}
		sb.WriteString("\n")
	}

	// Tech Stack Warnings section.
	if len(profile.TechStack.Warnings) > 0 {
		sb.WriteString("## Agent Compatibility Warnings\n\n")
		for _, w := range profile.TechStack.Warnings {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s. Alternative: %s\n",
				w.Technology, w.Level, w.Reason, w.Alternative))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// WriteAgentsMD generates AGENTS.md from the given profile and writes it
// to the specified directory. It does not overwrite an existing AGENTS.md.
// INV: Never modifies an existing file.
func WriteAgentsMD(profile *RepoProfile, dir string) error {
	target := filepath.Join(dir, "AGENTS.md")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	content := GenerateAgentsMD(profile)
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return fmt.Errorf("bootstrap: write AGENTS.md: %w", err)
	}
	return nil
}

// GenerateDocsStructure creates a docs/ directory skeleton with placeholder
// files inferred from the repository profile. Existing files are not overwritten.
// INV: Never modifies existing files.
func GenerateDocsStructure(profile *RepoProfile, dir string) error {
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return fmt.Errorf("bootstrap: create docs dir: %w", err)
	}

	adrDir := filepath.Join(docsDir, "adr")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		return fmt.Errorf("bootstrap: create docs/adr dir: %w", err)
	}

	// architecture.md
	archPath := filepath.Join(docsDir, "architecture.md")
	if _, err := os.Stat(archPath); err != nil {
		var sb strings.Builder
		sb.WriteString("# Architecture\n\n## Overview\n\n")
		if len(profile.Languages) > 0 {
			sb.WriteString("Languages: ")
			names := make([]string, 0, len(profile.Languages))
			for _, l := range profile.Languages {
				names = append(names, l.Name)
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n\n")
		}
		if len(profile.Frameworks) > 0 {
			sb.WriteString("Frameworks: ")
			names := make([]string, 0, len(profile.Frameworks))
			for _, f := range profile.Frameworks {
				names = append(names, f.Name)
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n")
		}
		if err := os.WriteFile(archPath, []byte(sb.String()), 0o644); err != nil {
			return fmt.Errorf("bootstrap: write architecture.md: %w", err)
		}
	}

	// getting-started.md
	gsPath := filepath.Join(docsDir, "getting-started.md")
	if _, err := os.Stat(gsPath); err != nil {
		var sb strings.Builder
		sb.WriteString("# Getting Started\n\n## Prerequisites\n\n")
		if len(profile.EntryPoints) > 0 {
			sb.WriteString("## Entry Points\n\n")
			for _, ep := range profile.EntryPoints {
				sb.WriteString(fmt.Sprintf("- **%s**: `%s`\n", ep.Name, ep.Command))
			}
		}
		if err := os.WriteFile(gsPath, []byte(sb.String()), 0o644); err != nil {
			return fmt.Errorf("bootstrap: write getting-started.md: %w", err)
		}
	}

	// adr/0001-initial-setup.md
	adrPath := filepath.Join(adrDir, "0001-initial-setup.md")
	if _, err := os.Stat(adrPath); err != nil {
		content := fmt.Sprintf("# ADR 0001: Initial project setup\n\nDate: %s\n\n## Status\n\nAccepted\n\n## Context\n\nInitial project setup.\n\n## Decision\n\nTBD\n\n## Consequences\n\nTBD\n",
			time.Now().Format("2006-01-02"))
		if err := os.WriteFile(adrPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("bootstrap: write ADR template: %w", err)
		}
	}

	return nil
}

// progressData is the JSON structure for progress tracking files.
type progressData struct {
	MissionID string        `json:"mission_id"`
	Items     []interface{} `json:"items"`
	UpdatedAt string        `json:"updated_at"`
}

// GenerateProgressFile creates an initial JSON progress tracking file for
// a mission. Does not overwrite an existing file.
// INV: Never modifies an existing file.
func GenerateProgressFile(missionID string, dir string) error {
	target := filepath.Join(dir, "progress.json")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	data := progressData{
		MissionID: missionID,
		Items:     []interface{}{},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("bootstrap: marshal progress: %w", err)
	}
	if err := os.WriteFile(target, b, 0o644); err != nil {
		return fmt.Errorf("bootstrap: write progress.json: %w", err)
	}
	return nil
}

// feature is the JSON structure for entries in feature-list.json.
type feature struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Status string `json:"status"`
}

// GenerateFeatureList creates a feature-list.json skeleton from the profile's
// detected entry points and frameworks. Does not overwrite an existing file.
// INV: Never modifies an existing file.
func GenerateFeatureList(profile *RepoProfile, dir string) error {
	target := filepath.Join(dir, "feature-list.json")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	var features []feature
	for _, ep := range profile.EntryPoints {
		features = append(features, feature{
			Name:   ep.Name,
			Source: "entry-point",
			Status: "pending",
		})
	}
	for _, fw := range profile.Frameworks {
		features = append(features, feature{
			Name:   fw.Name,
			Source: "framework",
			Status: "pending",
		})
	}
	if features == nil {
		features = []feature{}
	}
	b, err := json.MarshalIndent(features, "", "  ")
	if err != nil {
		return fmt.Errorf("bootstrap: marshal feature list: %w", err)
	}
	if err := os.WriteFile(target, b, 0o644); err != nil {
		return fmt.Errorf("bootstrap: write feature-list.json: %w", err)
	}
	return nil
}

// Bootstrap runs the full analysis and generation pipeline for a repository:
// analyze the repo, write AGENTS.md, generate docs structure, and create
// feature list. Existing files are never overwritten.
// INV: Never modifies existing files — only creates new ones.
func Bootstrap(path string) (*RepoProfile, error) {
	profile, err := AnalyzeRepo(path)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: analyze repo: %w", err)
	}

	var errs []string

	if err := WriteAgentsMD(profile, path); err != nil {
		errs = append(errs, err.Error())
	}
	if err := GenerateDocsStructure(profile, path); err != nil {
		errs = append(errs, err.Error())
	}
	if err := GenerateFeatureList(profile, path); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return profile, fmt.Errorf("bootstrap: generation errors: %s", strings.Join(errs, "; "))
	}
	return profile, nil
}

// clampScore ensures a score is within [0.0, 1.0].
func clampScore(s float64) float64 {
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// fileExists checks whether a file exists at the given repo-relative path.
func fileExists(base, rel string) bool {
	info, err := os.Stat(filepath.Join(base, rel))
	return err == nil && !info.IsDir()
}

// dirExists checks whether a directory exists at the given repo-relative path.
func dirExists(base, rel string) bool {
	info, err := os.Stat(filepath.Join(base, rel))
	return err == nil && info.IsDir()
}

// anyFileExists returns true if any of the given repo-relative paths exist as files.
func anyFileExists(base string, files []string) bool {
	for _, f := range files {
		if fileExists(base, f) {
			return true
		}
	}
	return false
}

// anyDirExists returns true if any of the given repo-relative paths exist as directories.
func anyDirExists(base string, dirs []string) bool {
	for _, d := range dirs {
		if dirExists(base, d) {
			return true
		}
	}
	return false
}

// anyExists returns true if any of the given repo-relative paths exist as files or directories.
func anyExists(base string, paths []string) bool {
	for _, p := range paths {
		if fileExists(base, p) || dirExists(base, p) {
			return true
		}
	}
	return false
}

// isSkippableDir reports whether a directory should be skipped during tree walks
// (hidden dirs, vendored dependencies, caches).
func isSkippableDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__"
}
