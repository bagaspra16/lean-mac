package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bagaspra16/lean-mac/internal/cleaner"
	"github.com/bagaspra16/lean-mac/internal/detectors"
	"github.com/bagaspra16/lean-mac/internal/scanner"
	"github.com/bagaspra16/lean-mac/internal/types"
	"github.com/dustin/go-humanize"
)

// allowedCategories is the closed set of values the AI may name in tool calls.
// Anything outside this list is rejected before the cleaner is invoked.
var allowedCategories = map[types.Category]bool{
	types.CatNode:        true,
	types.CatNpmCache:    true,
	types.CatPnpmStore:   true,
	types.CatYarnCache:   true,
	types.CatPipCache:    true,
	types.CatRustTarget:  true,
	types.CatRustCargo:   true,
	types.CatGoCache:     true,
	types.CatGoModCache:  true,
	types.CatXcodeDD:     true,
	types.CatXcodeArch:   true,
	types.CatXcodeSim:    true,
	types.CatDockerImg:   true,
	types.CatDockerVol:   true,
	types.CatDockerBuild: true,
	types.CatBrewCache:   true,
	types.CatGradle:      true,
	types.CatMaven:       true,
}

// Action is a single deletion proposal emitted by the agent. Each carries its
// risk so the UI can render colors and gate auto-execute.
type Action struct {
	Category types.Category
	Findings []types.Finding
	Total    int64
	Reason   string
	Risk     types.Risk
}

// EventKind distinguishes UI updates streamed from the agent.
type EventKind int

const (
	EvtThinking EventKind = iota
	EvtAssistant
	EvtScanStart
	EvtScanDone
	EvtProposal
	EvtExecutionStart
	EvtExecutionResult
	EvtError
	EvtDone
)

type Event struct {
	Kind     EventKind
	Text     string
	Action   *Action
	Report   *types.ScanReport
	Result   *types.CleanResult
	Err      error
}

// Agent orchestrates the AI conversation, scan, proposals, and (with the UI's
// per-action approval) cleanup.
type Agent struct {
	client *Client
	scan   *types.ScanReport
}

func NewAgent(client *Client) *Agent { return &Agent{client: client} }

// SystemPrompt is what we send the model. It's intentionally narrow — the
// model is here to analyze and explain, not to ad-lib filesystem operations.
const SystemPrompt = `You are the AI assistant inside lean-mac, a developer storage tool for macOS.

Your job is to help the user understand WHY their disk is full and propose
cleanup steps. You have two tools available:

1. scan_disk()           — discovers reclaimable artifacts. Always run this
                           first before proposing any cleanup.
2. propose_cleanup(...)  — propose deleting a CATEGORY of artifacts. The user
                           reviews and approves each proposal individually in
                           the UI. You CANNOT delete arbitrary paths; only
                           named categories from the scan result.

When proposing cleanup:
- Explain in plain English what the category is and why it's safe (or not).
- Start with SAFE categories (caches that regenerate).
- Only propose MEDIUM risk if the user has explicitly asked to be aggressive.
- NEVER propose DANGEROUS categories unless the user explicitly names them.
- Be brief. The user is reading this in a terminal.

You can never see or touch arbitrary paths. Don't ask the user for paths.
Don't suggest 'rm' commands. Work through propose_cleanup only.`

// Tools advertises the function-calling schema.
func Tools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        "scan_disk",
				Description: "Scan disk for reclaimable developer artifacts and return a summary by category.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDef{
				Name: "propose_cleanup",
				Description: "Propose removing one category of artifacts. The user will see your proposal and approve or reject it. Use after scan_disk.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"category": map[string]any{
							"type": "string",
							"enum": categoryEnum(),
							"description": "Category name from the scan result. Exactly one of the allowed values.",
						},
						"reason": map[string]any{
							"type":        "string",
							"description": "Short explanation for the user about why this is safe and what it does.",
						},
					},
					"required": []string{"category", "reason"},
				},
			},
		},
	}
}

func categoryEnum() []string {
	var out []string
	for c := range allowedCategories {
		out = append(out, string(c))
	}
	return out
}

// Step runs one turn of the agent loop. It sends the conversation to Groq,
// then dispatches any tool calls, emitting Events. Returns the updated
// conversation (with new assistant + tool messages appended).
func (a *Agent) Step(ctx context.Context, msgs []Message, out chan<- Event) ([]Message, error) {
	out <- Event{Kind: EvtThinking}
	reply, err := a.client.Chat(ctx, msgs, Tools())
	if err != nil {
		out <- Event{Kind: EvtError, Err: err}
		return msgs, err
	}
	msgs = append(msgs, reply)
	if reply.Content != "" {
		out <- Event{Kind: EvtAssistant, Text: reply.Content}
	}
	for _, tc := range reply.ToolCalls {
		toolMsg := a.dispatchTool(ctx, tc, out)
		msgs = append(msgs, toolMsg)
	}
	return msgs, nil
}

func (a *Agent) dispatchTool(ctx context.Context, tc ToolCall, out chan<- Event) Message {
	switch tc.Function.Name {
	case "scan_disk":
		return a.toolScan(ctx, tc, out)
	case "propose_cleanup":
		return a.toolPropose(tc, out)
	default:
		return Message{
			Role:       "tool",
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    "error: unknown tool",
		}
	}
}

func (a *Agent) toolScan(ctx context.Context, tc ToolCall, out chan<- Event) Message {
	out <- Event{Kind: EvtScanStart}
	s := scanner.New(detectors.Default()...)
	ch := make(chan scanner.Progress, 64)
	done := make(chan *types.ScanReport, 1)
	go func() { done <- s.Run(ctx, ch) }()
	for range ch {
	}
	rpt := <-done
	a.scan = rpt
	out <- Event{Kind: EvtScanDone, Report: rpt}
	return Message{
		Role:       "tool",
		ToolCallID: tc.ID,
		Name:       "scan_disk",
		Content:    summarizeScan(rpt),
	}
}

func (a *Agent) toolPropose(tc ToolCall, out chan<- Event) Message {
	var args struct {
		Category string `json:"category"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return toolErr(tc, "invalid arguments: "+err.Error())
	}
	cat := types.Category(args.Category)
	if !allowedCategories[cat] {
		return toolErr(tc, fmt.Sprintf("rejected: %q is not an allowed category", args.Category))
	}
	if a.scan == nil {
		return toolErr(tc, "no scan has been run yet; call scan_disk first")
	}
	action := &Action{Category: cat, Reason: args.Reason}
	for _, f := range a.scan.Findings {
		if f.Category == cat {
			action.Findings = append(action.Findings, f)
			action.Total += f.Size
			if f.Risk > action.Risk {
				action.Risk = f.Risk
			}
		}
	}
	if len(action.Findings) == 0 {
		return toolErr(tc, fmt.Sprintf("no findings for category %q in current scan", cat))
	}
	out <- Event{Kind: EvtProposal, Action: action}
	return Message{
		Role:       "tool",
		ToolCallID: tc.ID,
		Name:       "propose_cleanup",
		Content:    fmt.Sprintf("proposal sent to user: %s (%d items, %s)", cat, len(action.Findings), humanize.IBytes(uint64(action.Total))),
	}
}

func toolErr(tc ToolCall, msg string) Message {
	return Message{
		Role:       "tool",
		ToolCallID: tc.ID,
		Name:       tc.Function.Name,
		Content:    "error: " + msg,
	}
}

// Execute runs a single approved action through the cleaner. The path-level
// safety check inside the cleaner is the final line of defense.
func Execute(ctx context.Context, action *Action, dryRun bool, out chan<- Event) {
	if action == nil {
		out <- Event{Kind: EvtError, Err: errors.New("nil action")}
		return
	}
	out <- Event{Kind: EvtExecutionStart, Action: action}
	c := cleaner.New(cleaner.Options{
		DryRun:           dryRun,
		Aggressive:       action.Risk >= types.RiskMedium,
		IncludeDangerous: action.Risk == types.RiskDangerous,
	})
	rpt := c.Clean(ctx, action.Findings)
	for i := range rpt.Results {
		out <- Event{Kind: EvtExecutionResult, Result: &rpt.Results[i]}
	}
}

// FeedbackForModel turns user-approval results into a tool message the model
// can use to decide its next step.
func FeedbackForModel(action *Action, approved bool, freed int64) Message {
	if !approved {
		return Message{
			Role:    "user",
			Content: fmt.Sprintf("I declined the proposal to remove %s.", action.Category),
		}
	}
	return Message{
		Role:    "user",
		Content: fmt.Sprintf("I approved %s. Freed %s.", action.Category, humanize.IBytes(uint64(freed))),
	}
}

func summarizeScan(r *types.ScanReport) string {
	totals := map[types.Category]int64{}
	counts := map[types.Category]int{}
	risks := map[types.Category]types.Risk{}
	for _, f := range r.Findings {
		totals[f.Category] += f.Size
		counts[f.Category]++
		if f.Risk > risks[f.Category] {
			risks[f.Category] = f.Risk
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Scan complete. %d findings, %s total reclaimable. Disk free: %s of %s.\n\n",
		len(r.Findings),
		humanize.IBytes(uint64(r.TotalBytes)),
		humanize.IBytes(uint64(r.DiskFree)),
		humanize.IBytes(uint64(r.DiskTotal)),
	)
	fmt.Fprintf(&b, "By category:\n")
	for c, t := range totals {
		fmt.Fprintf(&b, "- %s: %d items, %s, risk=%s\n",
			c, counts[c], humanize.IBytes(uint64(t)), risks[c].String())
	}
	return b.String()
}
