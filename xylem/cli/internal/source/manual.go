package source

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
)

// Manual is the default source for ad-hoc tasks enqueued via the CLI.
type Manual struct{}

func (m *Manual) Name() string { return "manual" }

func (m *Manual) Scan(_ context.Context) ([]queue.Vessel, error) {
	return nil, nil // manual source doesn't auto-discover
}

func (m *Manual) OnStart(_ context.Context, _ queue.Vessel) error {
	return nil // no side effects
}

func (m *Manual) BranchName(vessel queue.Vessel) string {
	slug := slugify(vessel.Ref)
	if slug == "" {
		slug = "task"
	}
	return fmt.Sprintf("task/%s-%s", vessel.ID, slug)
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	parts := strings.Split(strings.ToLower(s), "/")
	src := parts[len(parts)-1]
	if src == "" && len(parts) > 1 {
		src = parts[len(parts)-2]
	}
	clean := nonAlphaNum.ReplaceAllString(src, "-")
	clean = strings.Trim(clean, "-")
	if len(clean) > 20 {
		clean = clean[:20]
		clean = strings.TrimRight(clean, "-")
	}
	return clean
}
