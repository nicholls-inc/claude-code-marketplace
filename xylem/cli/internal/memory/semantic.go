package memory

import (
	"strings"
	"unicode/utf8"
)

// SemanticCheck describes one semantic validation finding.
type SemanticCheck struct {
	Check    string `json:"check"`    // "contradiction", "hallucination", "duplication"
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// SemanticValidationResult aggregates all semantic checks for an entry.
type SemanticValidationResult struct {
	Valid  bool            `json:"valid"`
	Checks []SemanticCheck `json:"checks,omitempty"`
}

// SemanticValidator runs configurable semantic checks against memory entries.
type SemanticValidator struct {
	// MinValueLength is the minimum number of runes a value must have to avoid
	// being flagged as a potential hallucination.
	MinValueLength int
	// MaxKeyReuse is the maximum number of existing entries that may share the
	// same key before new writes are rejected as errors.
	MaxKeyReuse int
}

// DefaultSemanticValidator returns a validator with sensible defaults.
func DefaultSemanticValidator() *SemanticValidator {
	return &SemanticValidator{
		MinValueLength: 3,
		MaxKeyReuse:    50,
	}
}

// Validate runs all semantic checks against entry given the set of existing
// entries. Valid is true only when no check has Severity "error".
func (v *SemanticValidator) Validate(entry Entry, existing []Entry) SemanticValidationResult {
	var checks []SemanticCheck

	if c := v.detectContradiction(entry, existing); c != nil {
		checks = append(checks, *c)
	}
	if c := v.detectHallucination(entry); c != nil {
		checks = append(checks, *c)
	}
	if c := v.detectDuplication(entry, existing); c != nil {
		checks = append(checks, *c)
	}
	if c := v.detectKeyReuse(entry, existing); c != nil {
		checks = append(checks, *c)
	}

	valid := true
	for _, c := range checks {
		if c.Severity == "error" {
			valid = false
			break
		}
	}

	return SemanticValidationResult{
		Valid:  valid,
		Checks: checks,
	}
}

// detectContradiction checks whether an existing entry with the same key has a
// different value. Only applies to Semantic and Procedural types because
// updates to these types may indicate contradictory information. Returns a
// warning since legitimate updates are common.
func (v *SemanticValidator) detectContradiction(entry Entry, existing []Entry) *SemanticCheck {
	if entry.Type != Semantic && entry.Type != Procedural {
		return nil
	}
	for _, e := range existing {
		if e.Key == entry.Key && e.Type == entry.Type && e.Value != entry.Value {
			return &SemanticCheck{
				Check:    "contradiction",
				Message:  "existing entry with key " + entry.Key + " has a different value",
				Severity: "warning",
			}
		}
	}
	return nil
}

// placeholders are values that consist entirely of a placeholder token and
// carry no real information.
var placeholders = map[string]bool{
	"tbd":  true,
	"todo": true,
	"n/a":  true,
}

// detectHallucination flags values that are likely not meaningful content:
//   - value consisting entirely of a single repeated character
//   - value that is only a placeholder token (TBD, TODO, N/A)
//   - value shorter than MinValueLength (in runes)
func (v *SemanticValidator) detectHallucination(entry Entry) *SemanticCheck {
	val := strings.TrimSpace(entry.Value)

	// Check placeholder tokens (case-insensitive, exact match after trim).
	if placeholders[strings.ToLower(val)] {
		return &SemanticCheck{
			Check:    "hallucination",
			Message:  "value is a placeholder: " + val,
			Severity: "warning",
		}
	}

	// Check single repeated character (isRepeatedChar returns false for empty).
	if isRepeatedChar(val) {
		return &SemanticCheck{
			Check:    "hallucination",
			Message:  "value consists entirely of repeated characters",
			Severity: "warning",
		}
	}

	// Check minimum length.
	if utf8.RuneCountInString(val) < v.MinValueLength {
		return &SemanticCheck{
			Check:    "hallucination",
			Message:  "value is shorter than minimum length",
			Severity: "warning",
		}
	}

	return nil
}

// isRepeatedChar returns true if s consists entirely of the same rune.
func isRepeatedChar(s string) bool {
	if len(s) == 0 {
		return false
	}
	first, firstSize := utf8.DecodeRuneInString(s)
	if first == utf8.RuneError && firstSize <= 1 {
		return false
	}
	for i := firstSize; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r != first {
			return false
		}
		i += size
	}
	return true
}

// detectDuplication checks whether a different key already stores the exact
// same value. Exact string match only.
func (v *SemanticValidator) detectDuplication(entry Entry, existing []Entry) *SemanticCheck {
	for _, e := range existing {
		if e.Key != entry.Key && e.Value == entry.Value {
			return &SemanticCheck{
				Check:    "duplication",
				Message:  "existing entry with key " + e.Key + " has identical value",
				Severity: "warning",
			}
		}
	}
	return nil
}

// detectKeyReuse flags when a key has been reused more than MaxKeyReuse times,
// which may indicate a runaway write loop or context poisoning attempt.
func (v *SemanticValidator) detectKeyReuse(entry Entry, existing []Entry) *SemanticCheck {
	count := 0
	for _, e := range existing {
		if e.Key == entry.Key {
			count++
		}
	}
	if count >= v.MaxKeyReuse {
		return &SemanticCheck{
			Check:    "duplication",
			Message:  "key has been reused too many times",
			Severity: "error",
		}
	}
	return nil
}
