package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/config"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/gate"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/phase"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/queue"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/reporter"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/workflow"
	"github.com/nicholls-inc/claude-code-marketplace/xylem/cli/internal/source"
)

// CommandRunner abstracts subprocess execution for testing.
type CommandRunner interface {
	RunOutput(ctx context.Context, name string, args ...string) ([]byte, error)
	RunProcess(ctx context.Context, dir string, name string, args ...string) error
	RunPhase(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error)
}

// WorktreeManager abstracts worktree lifecycle for testing.
type WorktreeManager interface {
	Create(ctx context.Context, branchName string) (string, error)
}

// DrainResult summarises a drain run.
type DrainResult struct {
	Completed int
	Failed    int
	Skipped   int
	Waiting   int
}

// Runner launches Claude sessions for queued vessels with concurrency control.
type Runner struct {
	Config   *config.Config
	Queue    *queue.Queue
	Worktree WorktreeManager
	Runner   CommandRunner
	Sources  map[string]source.Source
	Reporter *reporter.Reporter // may be nil for non-github vessels
}

// New creates a Runner.
func New(cfg *config.Config, q *queue.Queue, wt WorktreeManager, r CommandRunner) *Runner {
	return &Runner{Config: cfg, Queue: q, Worktree: wt, Runner: r}
}

// Drain dequeues pending vessels and launches sessions up to Config.Concurrency concurrently.
// On context cancellation, no new vessels are launched; running vessels complete normally.
func (r *Runner) Drain(ctx context.Context) (DrainResult, error) {
	timeout, err := time.ParseDuration(r.Config.Timeout)
	if err != nil {
		return DrainResult{}, fmt.Errorf("parse timeout: %w", err)
	}

	sem := make(chan struct{}, r.Config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var result DrainResult

	for {
		select {
		case <-ctx.Done():
			goto wait
		default:
		}

		vessel, err := r.Queue.Dequeue()
		if err != nil || vessel == nil {
			break
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(j queue.Vessel) {
			defer wg.Done()
			defer func() { <-sem }()

			vesselCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			outcome := r.runVessel(vesselCtx, j)

			mu.Lock()
			switch outcome {
			case "completed":
				result.Completed++
			case "failed":
				result.Failed++
			case "waiting":
				result.Waiting++
			default:
				result.Skipped++
			}
			mu.Unlock()
		}(*vessel)
	}

wait:
	wg.Wait()
	return result, nil
}

// CheckWaitingVessels checks all waiting vessels for label gate resolution.
func (r *Runner) CheckWaitingVessels(ctx context.Context) {
	waiting, err := r.Queue.ListByState(queue.StateWaiting)
	if err != nil {
		log.Printf("warn: list waiting vessels: %v", err)
		return
	}

	for _, vessel := range waiting {
		// Check timeout
		if vessel.WaitingSince != nil {
			timeoutDur := 24 * time.Hour // default
			if vessel.Workflow != "" {
				if wf, loadErr := r.loadWorkflow(vessel.Workflow); loadErr == nil {
					if int(vessel.CurrentPhase) > 0 && int(vessel.CurrentPhase) <= len(wf.Phases) {
						prevPhase := wf.Phases[vessel.CurrentPhase-1]
						if prevPhase.Gate != nil && prevPhase.Gate.Timeout != "" {
							if parsed, pErr := time.ParseDuration(prevPhase.Gate.Timeout); pErr == nil {
								timeoutDur = parsed
							}
						}
					}
				}
			}
			if time.Since(*vessel.WaitingSince) > timeoutDur {
				if updateErr := r.Queue.Update(vessel.ID, queue.StateTimedOut, "label gate timed out"); updateErr != nil {
					log.Printf("warn: failed to update vessel %s to timed_out: %v", vessel.ID, updateErr)
				}
				// Post timeout comment
				issueNum := r.parseIssueNum(vessel)
				if issueNum > 0 && r.Reporter != nil {
					r.Reporter.LabelTimeout(ctx, issueNum, vessel.WaitingFor, vessel.FailedPhase, time.Since(*vessel.WaitingSince))
				}
				continue
			}
		}

		// Check label
		issueNum := r.parseIssueNum(vessel)
		repo := r.resolveRepo(vessel)
		if issueNum == 0 || repo == "" {
			continue
		}

		found, err := gate.CheckLabel(ctx, r.Runner, repo, issueNum, vessel.WaitingFor)
		if err != nil {
			log.Printf("warn: check label for vessel %s: %v", vessel.ID, err)
			continue
		}
		if found {
			// Advance past the gated phase — CurrentPhase was already incremented
			// when the vessel entered waiting state.
			if err := r.Queue.Update(vessel.ID, queue.StateRunning, ""); err != nil {
				log.Printf("warn: failed to resume vessel %s: %v", vessel.ID, err)
			}
		}
	}
}

func (r *Runner) runVessel(ctx context.Context, vessel queue.Vessel) string {
	// Look up source for this vessel
	src := r.resolveSource(vessel.Source)

	// Source-specific start hook (e.g., add in-progress label)
	if err := src.OnStart(ctx, vessel); err != nil {
		log.Printf("warn: source OnStart for %s: %v", vessel.ID, err)
	}

	// Worktree: reuse if set (resuming from waiting), otherwise create
	worktreePath := vessel.WorktreePath
	if worktreePath == "" {
		branchName := src.BranchName(vessel)
		var err error
		worktreePath, err = r.Worktree.Create(ctx, branchName)
		if err != nil {
			r.failVessel(vessel.ID, err.Error())
			return "failed"
		}
		vessel.WorktreePath = worktreePath
		if updateErr := r.Queue.UpdateVessel(vessel); updateErr != nil {
			log.Printf("warn: failed to persist worktree path for %s: %v", vessel.ID, updateErr)
		}
	}

	// Prompt-only vessel (no workflow): single claude -p invocation
	if vessel.Workflow == "" && vessel.Prompt != "" {
		return r.runPromptOnly(ctx, vessel, worktreePath)
	}

	// Load workflow definition
	if vessel.Workflow == "" {
		r.failVessel(vessel.ID, "vessel has neither workflow nor prompt")
		return "failed"
	}

	wf, err := r.loadWorkflow(vessel.Workflow)
	if err != nil {
		r.failVessel(vessel.ID, fmt.Sprintf("load workflow: %v", err))
		return "failed"
	}

	// Fetch issue data (GitHub source only)
	issueData := r.fetchIssueData(ctx, &vessel)

	// Read harness file
	harnessContent := r.readHarness()

	// Rebuild previousOutputs from .xylem/phases/<id>/*.output (for resume)
	previousOutputs := r.rebuildPreviousOutputs(vessel.ID, wf)

	// Execute phases
	var phaseResults []reporter.PhaseResult
	for i := vessel.CurrentPhase; i < len(wf.Phases); i++ {
		p := wf.Phases[i]
		gateResult := ""

		// Initialize gate retries for this phase (once, before retry loop)
		if p.Gate != nil && p.Gate.Type == "command" && p.Gate.Retries > 0 && vessel.GateRetries == 0 {
			vessel.GateRetries = p.Gate.Retries
		}

		// Gate retry loop: may re-run the same phase with gate output appended
		for {
			phaseStart := time.Now()

			// Build template data
			td := phase.TemplateData{
				Issue: issueData,
				Phase: phase.PhaseData{
					Name:  p.Name,
					Index: i,
				},
				PreviousOutputs: previousOutputs,
				GateResult:      gateResult,
				Vessel: phase.VesselData{
					ID:     vessel.ID,
					Source: vessel.Source,
				},
			}

			// Read prompt template
			promptContent, err := os.ReadFile(p.PromptFile)
			if err != nil {
				r.failVessel(vessel.ID, fmt.Sprintf("read prompt file %s: %v", p.PromptFile, err))
				return "failed"
			}

			// Render prompt
			rendered, err := phase.RenderPrompt(string(promptContent), td)
			if err != nil {
				r.failVessel(vessel.ID, fmt.Sprintf("render prompt for phase %s: %v", p.Name, err))
				return "failed"
			}

			// Write prompt to file for debugging
			phasesDir := filepath.Join(r.Config.StateDir, "phases", vessel.ID)
			os.MkdirAll(phasesDir, 0o755)
			promptPath := filepath.Join(phasesDir, p.Name+".prompt")
			if wErr := os.WriteFile(promptPath, []byte(rendered), 0o644); wErr != nil {
				log.Printf("warn: write prompt file %s: %v", promptPath, wErr)
			}

			// Construct claude args
			args := buildPhaseArgs(r.Config, &p, harnessContent)

			// Run phase via stdin
			output, runErr := r.Runner.RunPhase(ctx, worktreePath, strings.NewReader(rendered), r.Config.Claude.Command, args...)

			// Write phase output
			outputPath := filepath.Join(phasesDir, p.Name+".output")
			if wErr := os.WriteFile(outputPath, output, 0o644); wErr != nil {
				log.Printf("warn: write output file %s: %v", outputPath, wErr)
			}

			phaseDuration := time.Since(phaseStart)

			if runErr != nil {
				vessel.FailedPhase = p.Name
				r.failVessel(vessel.ID, fmt.Sprintf("phase %s: %v", p.Name, runErr))
				issueNum := r.parseIssueNum(vessel)
				if issueNum > 0 && r.Reporter != nil {
					r.Reporter.VesselFailed(ctx, issueNum, p.Name, runErr.Error(), "")
				}
				return "failed"
			}

			// Store output for subsequent phases
			previousOutputs[p.Name] = string(output)

			// Update current phase, persist
			vessel.CurrentPhase = i + 1
			if vessel.PhaseOutputs == nil {
				vessel.PhaseOutputs = make(map[string]string)
			}
			vessel.PhaseOutputs[p.Name] = outputPath
			if updateErr := r.Queue.UpdateVessel(vessel); updateErr != nil {
				log.Printf("warn: persist phase progress for %s: %v", vessel.ID, updateErr)
			}

			// Report phase completion (non-fatal)
			phaseResults = append(phaseResults, reporter.PhaseResult{Name: p.Name, Duration: phaseDuration})
			issueNum := r.parseIssueNum(vessel)
			if issueNum > 0 && r.Reporter != nil {
				r.Reporter.PhaseComplete(ctx, issueNum, p.Name, phaseDuration, string(output))
			}

			// Handle gate
			if p.Gate == nil {
				break // no gate, proceed to next phase
			}

			switch p.Gate.Type {
			case "command":
				gateOut, passed, gateErr := gate.RunCommandGate(ctx, r.Runner, worktreePath, p.Gate.Run)
				if gateErr != nil {
					r.failVessel(vessel.ID, fmt.Sprintf("phase %s gate error: %v", p.Name, gateErr))
					return "failed"
				}
				if passed {
					break // gate passed, proceed to next phase
				}

				// Gate failed
				retryDelay := 10 * time.Second
				if p.Gate.RetryDelay != "" {
					if parsed, pErr := time.ParseDuration(p.Gate.RetryDelay); pErr == nil {
						retryDelay = parsed
					}
				}

				if vessel.GateRetries <= 0 {
					vessel.FailedPhase = p.Name
					vessel.GateOutput = gateOut
					r.failVessel(vessel.ID, fmt.Sprintf("phase %s: gate failed, retries exhausted", p.Name))
					if issueNum > 0 && r.Reporter != nil {
						r.Reporter.VesselFailed(ctx, issueNum, p.Name, "gate failed, retries exhausted", gateOut)
					}
					return "failed"
				}

				vessel.GateRetries--
				if updateErr := r.Queue.UpdateVessel(vessel); updateErr != nil {
					log.Printf("warn: persist gate retries for %s: %v", vessel.ID, updateErr)
				}

				// Re-render prompt with gate output context
				gateResult = fmt.Sprintf("The following gate check failed after the previous phase. Fix the issues and try again:\n\n%s", gateOut)

				time.Sleep(retryDelay)
				continue // re-run same phase

			case "label":
				// Set vessel to waiting state
				vessel.WaitingFor = p.Gate.WaitFor
				now := time.Now().UTC()
				vessel.WaitingSince = &now
				vessel.State = queue.StateWaiting
				if updateErr := r.Queue.UpdateVessel(vessel); updateErr != nil {
					log.Printf("warn: persist waiting state for %s: %v", vessel.ID, updateErr)
				}
				// Update queue state
				if updateErr := r.Queue.Update(vessel.ID, queue.StateWaiting, ""); updateErr != nil {
					log.Printf("warn: failed to set vessel %s to waiting: %v", vessel.ID, updateErr)
				}
				return "waiting"
			}

			break // gate passed or unknown gate type
		}
	}

	// All phases complete
	if updateErr := r.Queue.Update(vessel.ID, queue.StateCompleted, ""); updateErr != nil {
		log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
	}

	// Report completion
	issueNum := r.parseIssueNum(vessel)
	if issueNum > 0 && r.Reporter != nil {
		r.Reporter.VesselCompleted(ctx, issueNum, phaseResults)
	}

	return "completed"
}

// runPromptOnly handles vessels with a prompt but no workflow.
func (r *Runner) runPromptOnly(ctx context.Context, vessel queue.Vessel, worktreePath string) string {
	prompt := vessel.Prompt
	if vessel.Ref != "" {
		prompt = fmt.Sprintf("Ref: %s\n\n%s", vessel.Ref, vessel.Prompt)
	}

	args := []string{"-p", "--max-turns", fmt.Sprintf("%d", r.Config.MaxTurns)}
	if r.Config.Claude.Flags != "" {
		args = append(args, strings.Fields(r.Config.Claude.Flags)...)
	}
	for _, tool := range r.Config.Claude.AllowedTools {
		args = append(args, "--allowedTools", tool)
	}

	_, runErr := r.Runner.RunPhase(ctx, worktreePath, strings.NewReader(prompt), r.Config.Claude.Command, args...)

	if runErr != nil {
		r.failVessel(vessel.ID, runErr.Error())
		return "failed"
	}

	if updateErr := r.Queue.Update(vessel.ID, queue.StateCompleted, ""); updateErr != nil {
		log.Printf("warn: failed to update vessel %s state: %v", vessel.ID, updateErr)
	}
	return "completed"
}

func (r *Runner) resolveSource(name string) source.Source {
	if r.Sources != nil {
		if src, ok := r.Sources[name]; ok {
			return src
		}
	}
	return &source.Manual{}
}

// buildCommand constructs the claude command and args from config and vessel.
func buildCommand(cfg *config.Config, vessel *queue.Vessel) (string, []string, error) {
	// Direct prompt mode
	if vessel.Prompt != "" {
		prompt := vessel.Prompt
		if vessel.Ref != "" {
			prompt = fmt.Sprintf("Ref: %s\n\n%s", vessel.Ref, vessel.Prompt)
		}
		args := []string{"-p", prompt, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns)}
		if cfg.Claude.Flags != "" {
			args = append(args, strings.Fields(cfg.Claude.Flags)...)
		}
		return cfg.Claude.Command, args, nil
	}

	// Workflow-based mode: build command from flags (v2 phase-based execution will replace this)
	prompt := fmt.Sprintf("/%s %s", vessel.Workflow, vessel.Ref)
	args := []string{"-p", prompt, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns)}
	if cfg.Claude.Flags != "" {
		args = append(args, strings.Fields(cfg.Claude.Flags)...)
	}
	return cfg.Claude.Command, args, nil
}

func (r *Runner) failVessel(id string, errMsg string) {
	if updateErr := r.Queue.Update(id, queue.StateFailed, errMsg); updateErr != nil {
		log.Printf("warn: failed to update vessel %s state: %v", id, updateErr)
	}
}

func (r *Runner) loadWorkflow(name string) (*workflow.Workflow, error) {
	path := filepath.Join(".xylem", "workflows", name+".yaml")
	return workflow.Load(path)
}

func (r *Runner) readHarness() string {
	data, err := os.ReadFile(filepath.Join(".xylem", "HARNESS.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func (r *Runner) fetchIssueData(ctx context.Context, vessel *queue.Vessel) phase.IssueData {
	data := phase.IssueData{}

	if vessel.Source != "github-issue" {
		return data
	}

	issueNum := r.parseIssueNum(*vessel)
	if issueNum == 0 {
		return data
	}

	// Check if already cached in Meta
	if vessel.Meta != nil && vessel.Meta["issue_title"] != "" {
		data.Number = issueNum
		data.Title = vessel.Meta["issue_title"]
		data.Body = vessel.Meta["issue_body"]
		data.URL = vessel.Ref
		if labelsStr, ok := vessel.Meta["issue_labels"]; ok {
			data.Labels = strings.Split(labelsStr, ",")
		}
		return data
	}

	repo := r.resolveRepo(*vessel)
	if repo == "" {
		return data
	}

	out, err := r.Runner.RunOutput(ctx, "gh", "issue", "view",
		fmt.Sprintf("%d", issueNum),
		"--repo", repo,
		"--json", "title,body,labels,url",
	)
	if err != nil {
		log.Printf("warn: fetch issue data for vessel %s: %v", vessel.ID, err)
		return data
	}

	var resp struct {
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"url"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		log.Printf("warn: parse issue data for vessel %s: %v", vessel.ID, err)
		return data
	}

	data.Number = issueNum
	data.Title = resp.Title
	data.Body = resp.Body
	data.URL = resp.URL
	for _, l := range resp.Labels {
		data.Labels = append(data.Labels, l.Name)
	}

	// Cache in Meta
	if vessel.Meta == nil {
		vessel.Meta = make(map[string]string)
	}
	vessel.Meta["issue_title"] = resp.Title
	vessel.Meta["issue_body"] = resp.Body
	vessel.Meta["issue_labels"] = strings.Join(data.Labels, ",")

	return data
}

func (r *Runner) rebuildPreviousOutputs(vesselID string, wf *workflow.Workflow) map[string]string {
	outputs := make(map[string]string)
	phasesDir := filepath.Join(r.Config.StateDir, "phases", vesselID)
	for _, p := range wf.Phases {
		outputPath := filepath.Join(phasesDir, p.Name+".output")
		data, err := os.ReadFile(outputPath)
		if err == nil {
			outputs[p.Name] = string(data)
		}
	}
	return outputs
}

func (r *Runner) parseIssueNum(vessel queue.Vessel) int {
	if vessel.Meta == nil {
		return 0
	}
	numStr, ok := vessel.Meta["issue_num"]
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}
	return n
}

func (r *Runner) resolveRepo(vessel queue.Vessel) string {
	if r.Sources == nil {
		return ""
	}
	src, ok := r.Sources[vessel.Source]
	if !ok {
		return ""
	}
	gh, ok := src.(*source.GitHub)
	if !ok {
		return ""
	}
	return gh.Repo
}

// buildPhaseArgs constructs the claude CLI arguments for a phase invocation.
func buildPhaseArgs(cfg *config.Config, p *workflow.Phase, harnessContent string) []string {
	args := []string{"-p"}
	args = append(args, "--max-turns", fmt.Sprintf("%d", p.MaxTurns))

	if cfg.Claude.Flags != "" {
		args = append(args, strings.Fields(cfg.Claude.Flags)...)
	}

	if p.AllowedTools != nil && *p.AllowedTools != "" {
		args = append(args, "--allowedTools", *p.AllowedTools)
	}

	for _, tool := range cfg.Claude.AllowedTools {
		args = append(args, "--allowedTools", tool)
	}

	if harnessContent != "" {
		args = append(args, "--append-system-prompt", harnessContent)
	}

	return args
}
