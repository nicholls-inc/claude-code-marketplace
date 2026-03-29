package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InstructionLevel describes the scope at which an instruction applies.
type InstructionLevel int

const (
	// OrgLevel is an organization-wide instruction.
	OrgLevel InstructionLevel = iota
	// RepoLevel is a repository-wide instruction.
	RepoLevel
	// DirLevel is a directory-specific instruction.
	DirLevel
)

// String returns the human-readable name of the level.
func (l InstructionLevel) String() string {
	switch l {
	case OrgLevel:
		return "org"
	case RepoLevel:
		return "repo"
	case DirLevel:
		return "dir"
	}
	return "unknown"
}

// Instruction represents a single convention or standard detected at a
// particular level of the instruction hierarchy.
type Instruction struct {
	Level   InstructionLevel
	Path    string // Relative directory (e.g. ".", "src/frontend")
	Content string // Human-readable description of the convention
	Source  string // Config file that sourced this instruction (e.g. ".eslintrc.json")
}

// InstructionSet is an ordered collection of instructions resulting from
// merging org, repo, and directory levels.
type InstructionSet struct {
	Instructions []Instruction
}

// ciConfigGlobs lists CI configuration patterns that require directory walking.
var ciConfigGlobs = []string{".github/workflows/*.yml"}

// ciConfigFiles lists CI configuration files checked at fixed paths.
var ciConfigFiles = []string{".gitlab-ci.yml", "Jenkinsfile", ".circleci/config.yml"}

// DetectConventionFiles walks the directory tree rooted at path looking for
// convention config files (linters, formatters, CI) and returns a dir-level
// Instruction for each one found.
// INV: Returns nil when no convention files are found.
func DetectConventionFiles(path string) []Instruction {
	var instructions []Instruction

	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if p != path && isSkippableDir(filepath.Base(p)) {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(p)
		rel, err := filepath.Rel(path, filepath.Dir(p))
		if err != nil {
			return nil
		}

		category := conventionCategory(base)
		if category == "" {
			return nil
		}

		instructions = append(instructions, Instruction{
			Level:   DirLevel,
			Path:    rel,
			Content: fmt.Sprintf("%s config: %s", category, base),
			Source:  base,
		})

		return nil
	})

	// Check CI globs separately (e.g. .github/workflows/*.yml).
	for _, pattern := range ciConfigGlobs {
		matches, err := filepath.Glob(filepath.Join(path, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			rel, err := filepath.Rel(path, filepath.Dir(m))
			if err != nil {
				continue
			}
			base := filepath.Base(m)
			// Avoid duplicates — the walk above already picks up files whose
			// base name matches a linter/formatter config. CI workflow files
			// are only detected via glob, so no overlap in practice.
			instructions = append(instructions, Instruction{
				Level:   DirLevel,
				Path:    rel,
				Content: fmt.Sprintf("CI config: %s", base),
				Source:  base,
			})
		}
	}

	// Sort for determinism before returning.
	sortInstructions(instructions)
	return instructions
}

// conventionCategory returns the category label for a convention config file
// name, or "" if the file is not a known convention config.
func conventionCategory(name string) string {
	for _, lc := range linterConfigs {
		if name == lc {
			return "Linter"
		}
	}
	for _, fc := range formatterConfigs {
		if name == fc {
			return "Formatter"
		}
	}
	for _, ci := range ciConfigFiles {
		// CI files may contain path separators; match on base name only.
		if name == filepath.Base(ci) {
			return "CI"
		}
	}
	return ""
}

// MergeInstructions performs a deterministic cascade merge of instructions from
// three hierarchy levels. When instructions share the same Path and Source,
// dir-level overrides repo-level which overrides org-level.
// The output is sorted by Path then Source.
// INV: Output length <= len(org) + len(repo) + len(dir).
// INV: Output is sorted by (Path, Source).
func MergeInstructions(org, repo, dir []Instruction) InstructionSet {
	type key struct {
		Path   string
		Source string
	}

	merged := make(map[key]Instruction)

	// Apply in precedence order: org first, then repo, then dir.
	// Later levels overwrite earlier ones for the same key.
	for _, inst := range org {
		merged[key{inst.Path, inst.Source}] = inst
	}
	for _, inst := range repo {
		merged[key{inst.Path, inst.Source}] = inst
	}
	for _, inst := range dir {
		merged[key{inst.Path, inst.Source}] = inst
	}

	result := make([]Instruction, 0, len(merged))
	for _, inst := range merged {
		result = append(result, inst)
	}

	sortInstructions(result)
	return InstructionSet{Instructions: result}
}

// DetectInstructionHierarchy orchestrates convention detection and creates a
// merged instruction set from repo-level profile data and directory-level
// convention file detection.
// INV: Returned InstructionSet is sorted by (Path, Source).
func DetectInstructionHierarchy(repoRoot string, profile *RepoProfile) InstructionSet {
	// Org-level: none currently (placeholder for future org config).
	var org []Instruction

	// Repo-level: derive from profile.
	// Source includes the name to avoid key collisions when multiple
	// languages or frameworks are detected (merge keys on Path+Source).
	var repo []Instruction
	for _, lang := range profile.Languages {
		repo = append(repo, Instruction{
			Level:   RepoLevel,
			Path:    ".",
			Content: fmt.Sprintf("Primary language: %s (%d files)", lang.Name, lang.FileCount),
			Source:  fmt.Sprintf("language:%s", lang.Name),
		})
	}
	for _, fw := range profile.Frameworks {
		repo = append(repo, Instruction{
			Level:   RepoLevel,
			Path:    ".",
			Content: fmt.Sprintf("Framework: %s (%s)", fw.Name, fw.Language),
			Source:  fmt.Sprintf("framework:%s", fw.Name),
		})
	}

	// Dir-level: walk for convention files.
	dir := DetectConventionFiles(repoRoot)

	return MergeInstructions(org, repo, dir)
}

// WriteDirInstructions writes an AGENTS.md file for each unique directory that
// has dir-level instructions, unless an AGENTS.md already exists there.
// The repo-root directory is skipped since WriteAgentsMD handles it separately.
// INV: Never modifies an existing file.
func WriteDirInstructions(instructions InstructionSet, repoRoot string) error {
	// Group instructions by directory path.
	byDir := make(map[string][]Instruction)
	for _, inst := range instructions.Instructions {
		if inst.Level != DirLevel {
			continue
		}
		byDir[inst.Path] = append(byDir[inst.Path], inst)
	}

	// Sort directory keys for deterministic output.
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		// Skip repo root — handled by WriteAgentsMD.
		if dir == "." {
			continue
		}

		target := filepath.Join(repoRoot, dir, "AGENTS.md")
		if _, err := os.Stat(target); err == nil {
			// Already exists — never overwrite.
			continue
		}

		if err := os.MkdirAll(filepath.Join(repoRoot, dir), 0o755); err != nil {
			return fmt.Errorf("bootstrap: create dir %q: %w", dir, err)
		}

		content := generateDirAgentsMD(byDir[dir])
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return fmt.Errorf("bootstrap: write %s/AGENTS.md: %w", dir, err)
		}
	}

	return nil
}

// generateDirAgentsMD formats directory-level instructions into an AGENTS.md
// document.
func generateDirAgentsMD(instructions []Instruction) string {
	var sb strings.Builder
	sb.WriteString("# AGENTS.md\n\n## Directory Instructions\n\nDetected conventions for this directory:\n\n")
	for _, inst := range instructions {
		sb.WriteString(fmt.Sprintf("- %s\n", inst.Content))
	}
	return sb.String()
}

// sortInstructions sorts a slice of instructions by Path then Source.
func sortInstructions(instructions []Instruction) {
	sort.Slice(instructions, func(i, j int) bool {
		if instructions[i].Path != instructions[j].Path {
			return instructions[i].Path < instructions[j].Path
		}
		return instructions[i].Source < instructions[j].Source
	})
}
