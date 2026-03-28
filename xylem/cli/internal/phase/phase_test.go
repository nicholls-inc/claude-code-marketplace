package phase

import (
	"fmt"
	"strings"
	"testing"
)

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string under limit",
			input:  "hello",
			maxLen: 100,
			want:   "hello",
		},
		{
			name:   "exact length string",
			input:  "abcde",
			maxLen: 5,
			want:   "abcde",
		},
		{
			name:   "string exceeding limit",
			input:  "abcdefghij",
			maxLen: 5,
			want:   "abcde" + fmt.Sprintf(TruncationSuffix, 5),
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 100,
			want:   "",
		},
		{
			name:   "string of exactly limit plus one",
			input:  "abcdef",
			maxLen: 5,
			want:   "abcde" + fmt.Sprintf(TruncationSuffix, 5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateOutput(tt.input, tt.maxLen)
			if got != tt.want {
				t.Fatalf("TruncateOutput(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestRenderPrompt(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     TemplateData
		want     string
		wantErr  bool
	}{
		{
			name:     "renders issue title",
			template: "Title: {{.Issue.Title}}",
			data: TemplateData{
				Issue: IssueData{Title: "Fix the bug"},
			},
			want: "Title: Fix the bug",
		},
		{
			name:     "renders issue body",
			template: "Body: {{.Issue.Body}}",
			data: TemplateData{
				Issue: IssueData{Body: "Detailed description here"},
			},
			want: "Body: Detailed description here",
		},
		{
			name:     "renders issue labels",
			template: "Labels: {{.Issue.Labels}}",
			data: TemplateData{
				Issue: IssueData{Labels: []string{"bug", "urgent"}},
			},
			want: "Labels: [bug urgent]",
		},
		{
			name:     "renders phase name and index",
			template: "Phase: {{.Phase.Name}} ({{.Phase.Index}})",
			data: TemplateData{
				Phase: PhaseData{Name: "analyze", Index: 0},
			},
			want: "Phase: analyze (0)",
		},
		{
			name:     "renders previous phase output",
			template: "Previous: {{.PreviousOutputs.analyze}}",
			data: TemplateData{
				PreviousOutputs: map[string]string{
					"analyze": "analysis result",
				},
			},
			want: "Previous: analysis result",
		},
		{
			name:     "nonexistent previous output renders empty",
			template: "Previous: [{{.PreviousOutputs.nonexistent}}]",
			data: TemplateData{
				PreviousOutputs: map[string]string{
					"analyze": "analysis result",
				},
			},
			want: "Previous: []",
		},
		{
			name:     "renders gate result",
			template: "Gate: {{.GateResult}}",
			data: TemplateData{
				GateResult: "all checks passed",
			},
			want: "Gate: all checks passed",
		},
		{
			name:     "renders vessel ID and source",
			template: "Vessel: {{.Vessel.ID}} from {{.Vessel.Source}}",
			data: TemplateData{
				Vessel: VesselData{ID: "issue-42", Source: "github"},
			},
			want: "Vessel: issue-42 from github",
		},
		{
			name:     "template syntax error",
			template: "{{.BadSyntax",
			data:     TemplateData{},
			wantErr:  true,
		},
		{
			name:     "undefined function",
			template: "{{undefined}}",
			data:     TemplateData{},
			wantErr:  true,
		},
		{
			name:     "empty template",
			template: "",
			data:     TemplateData{},
			want:     "",
		},
		{
			name:     "complex template combining multiple fields",
			template: "Issue #{{.Issue.Number}}: {{.Issue.Title}}\nURL: {{.Issue.URL}}\nPhase: {{.Phase.Name}} ({{.Phase.Index}})\nVessel: {{.Vessel.ID}}\nPrevious: {{.PreviousOutputs.plan}}\nGate: {{.GateResult}}",
			data: TemplateData{
				Issue: IssueData{
					URL:    "https://github.com/org/repo/issues/7",
					Title:  "Add caching layer",
					Number: 7,
				},
				Phase: PhaseData{Name: "implement", Index: 2},
				PreviousOutputs: map[string]string{
					"plan": "Step 1: Add Redis. Step 2: Wire it up.",
				},
				GateResult: "tests pass",
				Vessel:     VesselData{ID: "issue-7", Source: "github"},
			},
			want: "Issue #7: Add caching layer\nURL: https://github.com/org/repo/issues/7\nPhase: implement (2)\nVessel: issue-7\nPrevious: Step 1: Add Redis. Step 2: Wire it up.\nGate: tests pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderPrompt(tt.template, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("RenderPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderPromptTruncation(t *testing.T) {
	longString := func(n int) string {
		return strings.Repeat("x", n)
	}

	t.Run("PreviousOutputs value exceeding limit is truncated", func(t *testing.T) {
		data := TemplateData{
			PreviousOutputs: map[string]string{
				"analyze": longString(MaxPreviousOutputLen + 500),
			},
		}
		got, err := RenderPrompt("{{.PreviousOutputs.analyze}}", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		suffix := fmt.Sprintf(TruncationSuffix, MaxPreviousOutputLen)
		if !strings.HasSuffix(got, suffix) {
			t.Fatalf("expected truncation suffix, got ending: %q", got[len(got)-80:])
		}
		// The content before the suffix should be exactly MaxPreviousOutputLen chars.
		contentLen := len(got) - len(suffix)
		if contentLen != MaxPreviousOutputLen {
			t.Fatalf("content length = %d, want %d", contentLen, MaxPreviousOutputLen)
		}
	})

	t.Run("GateResult exceeding limit is truncated", func(t *testing.T) {
		data := TemplateData{
			GateResult: longString(MaxGateResultLen + 100),
		}
		got, err := RenderPrompt("{{.GateResult}}", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		suffix := fmt.Sprintf(TruncationSuffix, MaxGateResultLen)
		if !strings.HasSuffix(got, suffix) {
			t.Fatalf("expected truncation suffix")
		}
	})

	t.Run("Issue.Body exceeding limit is truncated", func(t *testing.T) {
		data := TemplateData{
			Issue: IssueData{Body: longString(MaxIssueBodyLen + 200)},
		}
		got, err := RenderPrompt("{{.Issue.Body}}", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		suffix := fmt.Sprintf(TruncationSuffix, MaxIssueBodyLen)
		if !strings.HasSuffix(got, suffix) {
			t.Fatalf("expected truncation suffix")
		}
	})

	t.Run("only large PreviousOutputs value is truncated", func(t *testing.T) {
		shortVal := "short output"
		longVal := longString(MaxPreviousOutputLen + 100)
		data := TemplateData{
			PreviousOutputs: map[string]string{
				"small": shortVal,
				"large": longVal,
			},
		}

		// Check that small value is not truncated.
		got, err := RenderPrompt("{{.PreviousOutputs.small}}", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != shortVal {
			t.Fatalf("small value was modified: got %q, want %q", got, shortVal)
		}

		// Check that large value is truncated.
		got, err = RenderPrompt("{{.PreviousOutputs.large}}", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		suffix := fmt.Sprintf(TruncationSuffix, MaxPreviousOutputLen)
		if !strings.HasSuffix(got, suffix) {
			t.Fatalf("expected truncation suffix for large value")
		}
	})
}

func TestPrepareDataDoesNotMutateOriginal(t *testing.T) {
	original := TemplateData{
		PreviousOutputs: map[string]string{
			"analyze": strings.Repeat("x", MaxPreviousOutputLen+100),
		},
		GateResult: strings.Repeat("y", MaxGateResultLen+100),
		Issue:      IssueData{Body: strings.Repeat("z", MaxIssueBodyLen+100)},
	}

	originalAnalyzeLen := len(original.PreviousOutputs["analyze"])
	originalGateLen := len(original.GateResult)
	originalBodyLen := len(original.Issue.Body)

	_ = prepareData(original)

	if len(original.PreviousOutputs["analyze"]) != originalAnalyzeLen {
		t.Fatal("prepareData mutated original PreviousOutputs")
	}
	if len(original.GateResult) != originalGateLen {
		t.Fatal("prepareData mutated original GateResult")
	}
	if len(original.Issue.Body) != originalBodyLen {
		t.Fatal("prepareData mutated original Issue.Body")
	}
}

func TestRenderPromptNilPreviousOutputs(t *testing.T) {
	data := TemplateData{
		PreviousOutputs: nil,
	}
	got, err := RenderPrompt("done", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "done" {
		t.Fatalf("got %q, want %q", got, "done")
	}
}
