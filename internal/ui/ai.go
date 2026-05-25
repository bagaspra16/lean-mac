package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaspra16/lean-mac/internal/ai"
	"github.com/bagaspra16/lean-mac/internal/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// aiState is the high-level state of the AI loop.
type aiState int

const (
	aiIdle           aiState = iota // waiting for user input
	aiThinking                       // request in flight
	aiAwaitApproval                  // proposal shown, waiting on y/n/a/c
	aiExecuting                      // running an approved action
	aiDisabled                       // no keys
)

// chatLine is one entry in the scrollback.
type chatLine struct {
	role string // "user" | "assistant" | "system" | "tool"
	text string
}

// aiEventMsg / aiTurnDoneMsg / aiExecResultMsg propagate async results.
type aiEventMsg struct{ ev ai.Event }
type aiTurnDoneMsg struct {
	msgs []ai.Message
	err  error
}
type aiExecDoneMsg struct{ freed int64 }

type aiView struct {
	client *ai.Client
	agent  *ai.Agent

	state    aiState
	dryRun   bool
	autoSafe bool

	input string
	lines []chatLine

	convo   []ai.Message
	pending *ai.Action

	events chan ai.Event
	cancel context.CancelFunc

	width, height int
}

func newAIView(client *ai.Client, dryRun bool) *aiView {
	v := &aiView{
		client: client,
		agent:  ai.NewAgent(client),
		state:  aiIdle,
		dryRun: dryRun,
		events: make(chan ai.Event, 32),
		convo: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
		},
	}
	// no seed lines — the welcome card is rendered when there is no chat yet.
	return v
}

func newAIDisabledView() *aiView {
	return &aiView{state: aiDisabled}
}

func (v *aiView) Init() tea.Cmd { return waitAIEvent(v.events) }

func waitAIEvent(ch chan ai.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return aiEventMsg{ev: ev}
	}
}

func (v *aiView) Title() string {
	switch v.state {
	case aiDisabled:
		return "AI Cleanse"
	case aiThinking:
		return "AI Cleanse"
	case aiAwaitApproval:
		return "AI Cleanse"
	case aiExecuting:
		return "AI Cleanse"
	default:
		return "AI Cleanse"
	}
}

func (v *aiView) Subtitle() string {
	switch v.state {
	case aiDisabled:
		return "not configured · see setup below"
	case aiThinking:
		return "thinking…"
	case aiAwaitApproval:
		return "awaiting your approval"
	case aiExecuting:
		return "executing approved action"
	default:
		return "conversational cleanup with per-action approval"
	}
}

func (v *aiView) Status() string {
	if v.state == aiDisabled {
		return "groq key required"
	}
	mode := "dry-run"
	if !v.dryRun {
		mode = "live"
	}
	auto := ""
	if v.autoSafe {
		auto = " · auto-safe ON"
	}
	return mode + auto
}

func (v *aiView) Footer() []hint {
	switch v.state {
	case aiDisabled:
		return []hint{{"3", "open Help"}}
	case aiAwaitApproval:
		return []hint{
			{"y", "approve"},
			{"n", "reject"},
			{"a", "auto-approve SAFE"},
			{"c", "cancel"},
		}
	case aiThinking, aiExecuting:
		return []hint{{"ctrl+c", "abort"}}
	default:
		return []hint{
			{"type", "your question"},
			{"enter", "send"},
			{"ctrl+u", "clear input"},
		}
	}
}

// wantsKey lets the App route keystrokes here while the input is focused.
// Globals (tab, q) should still work when we're idle and the input is empty,
// or when we're in approval/exec state.
func (v *aiView) wantsKey(msg tea.KeyMsg) bool {
	if v.state == aiDisabled {
		return false
	}
	if v.state == aiAwaitApproval {
		return false // let global keys (tab) work; approval uses single letters
	}
	if v.state == aiThinking || v.state == aiExecuting {
		return false
	}
	// idle: capture printable runes + edit keys so typing doesn't hit globals.
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyEnter, tea.KeyCtrlU:
		return true
	}
	return false
}

func (v *aiView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = m.Width, m.Height
		return v, nil
	case aiEventMsg:
		return v.onEvent(m.ev)
	case aiTurnDoneMsg:
		if m.err != nil {
			v.append("system", "error: "+m.err.Error())
			v.state = aiIdle
		}
		v.convo = m.msgs
		// State transitions are driven by events (EvtProposal → awaiting,
		// EvtError → idle). If after the turn we're still in aiThinking with no
		// pending proposal, the model just chatted — return to idle.
		if v.state == aiThinking && v.pending == nil {
			v.state = aiIdle
		}
		return v, waitAIEvent(v.events)
	case aiExecDoneMsg:
		v.append("system", fmt.Sprintf("✓ done. freed %s.", humanize.IBytes(uint64(m.freed))))
		// feed result back to model
		v.convo = append(v.convo, ai.FeedbackForModel(v.pending, true, m.freed))
		approved := v.pending
		v.pending = nil
		v.state = aiThinking
		return v, tea.Batch(v.runTurn(), waitAIEvent(v.events), func() tea.Msg {
			_ = approved
			return nil
		})
	case tea.KeyMsg:
		return v.onKey(m)
	}
	return v, nil
}

func (v *aiView) onEvent(ev ai.Event) (view, tea.Cmd) {
	switch ev.Kind {
	case ai.EvtThinking:
		v.state = aiThinking
	case ai.EvtAssistant:
		v.append("assistant", ev.Text)
	case ai.EvtScanStart:
		v.append("system", "scanning disk…")
	case ai.EvtScanDone:
		if ev.Report != nil {
			v.append("system", fmt.Sprintf("scan done · %d findings · %s reclaimable",
				len(ev.Report.Findings),
				humanize.IBytes(uint64(ev.Report.TotalBytes))))
		}
	case ai.EvtProposal:
		v.pending = ev.Action
		v.state = aiAwaitApproval
		v.append("system", v.renderProposal(ev.Action))
		if v.autoSafe && ev.Action.Risk == types.RiskSafe {
			return v, v.approve()
		}
	case ai.EvtError:
		v.append("system", "error: "+ev.Err.Error())
		v.state = aiIdle
	}
	return v, waitAIEvent(v.events)
}

func (v *aiView) renderProposal(a *ai.Action) string {
	risk := riskStyle(a.Risk.String()).Render("[" + a.Risk.String() + "]")
	header := fmt.Sprintf("%s proposal: remove %s (%d items, %s)",
		risk, a.Category, len(a.Findings), humanize.IBytes(uint64(a.Total)))
	preview := []string{header, "  " + a.Reason}
	for i, f := range a.Findings {
		if i >= 3 {
			preview = append(preview, fmt.Sprintf("  · and %d more…", len(a.Findings)-3))
			break
		}
		preview = append(preview, fmt.Sprintf("  · %s (%s)", f.Path, humanize.IBytes(uint64(f.Size))))
	}
	return strings.Join(preview, "\n")
}

func (v *aiView) onKey(msg tea.KeyMsg) (view, tea.Cmd) {
	if v.state == aiAwaitApproval {
		switch msg.String() {
		case "y":
			return v, v.approve()
		case "n":
			return v, v.reject()
		case "a":
			v.autoSafe = true
			v.append("system", "auto-approve SAFE proposals enabled.")
			if v.pending != nil && v.pending.Risk == types.RiskSafe {
				return v, v.approve()
			}
		case "c":
			return v, v.cancelAgent()
		}
		return v, nil
	}
	if v.state == aiThinking || v.state == aiExecuting {
		if msg.String() == "c" || msg.Type == tea.KeyCtrlC {
			return v, v.cancelAgent()
		}
		return v, nil
	}
	if v.state == aiDisabled {
		return v, nil
	}
	// idle — typing
	switch msg.Type {
	case tea.KeyEnter:
		text := strings.TrimSpace(v.input)
		if text == "" {
			return v, nil
		}
		v.input = ""
		v.append("user", text)
		v.convo = append(v.convo, ai.Message{Role: "user", Content: text})
		v.state = aiThinking
		return v, tea.Batch(v.runTurn(), waitAIEvent(v.events))
	case tea.KeyBackspace:
		if n := len(v.input); n > 0 {
			v.input = v.input[:n-1]
		}
	case tea.KeyCtrlU:
		v.input = ""
	case tea.KeySpace:
		v.input += " "
	case tea.KeyRunes:
		v.input += string(msg.Runes)
	}
	return v, nil
}

func (v *aiView) approve() tea.Cmd {
	action := v.pending
	if action == nil {
		return nil
	}
	v.state = aiExecuting
	v.append("system", fmt.Sprintf("approved · executing %s…", action.Category))
	dry := v.dryRun
	events := v.events
	return func() tea.Msg {
		// stream execution events into v.events for live feedback
		ai.Execute(context.Background(), action, dry, events)
		// Execute doesn't emit a freed-total; sum from action since cleaner returns
		// per-result bytes via events. For simplicity we trust action.Total here
		// (in dry-run that's exact; live runs may have partial successes — those
		// show as individual ✗ events to the user).
		return aiExecDoneMsg{freed: action.Total}
	}
}

func (v *aiView) reject() tea.Cmd {
	action := v.pending
	v.pending = nil
	v.append("system", fmt.Sprintf("declined · skipping %s.", action.Category))
	v.convo = append(v.convo, ai.FeedbackForModel(action, false, 0))
	v.state = aiThinking
	return tea.Batch(v.runTurn(), waitAIEvent(v.events))
}

func (v *aiView) cancelAgent() tea.Cmd {
	if v.cancel != nil {
		v.cancel()
	}
	v.pending = nil
	v.state = aiIdle
	v.append("system", "cancelled.")
	return nil
}

// runTurn fires one agent step. It loops internally if the model returns tool
// calls but no proposal (e.g. scan_disk then immediate follow-up), so the user
// always sees either an assistant text or a proposal.
func (v *aiView) runTurn() tea.Cmd {
	convo := v.convo
	agent := v.agent
	events := v.events
	ctx, cancel := context.WithCancel(context.Background())
	v.cancel = cancel
	return func() tea.Msg {
		defer cancel()
		// up to 4 chained tool turns per user message
		for i := 0; i < 4; i++ {
			next, err := agent.Step(ctx, convo, events)
			if err != nil {
				return aiTurnDoneMsg{msgs: convo, err: err}
			}
			convo = next
			last := convo[len(convo)-1]
			if last.Role == "tool" {
				continue // model called a tool; loop to let it reply
			}
			break
		}
		return aiTurnDoneMsg{msgs: convo, err: nil}
	}
}

func (v *aiView) append(role, text string) {
	v.lines = append(v.lines, chatLine{role: role, text: text})
}

func (v *aiView) View(width, height int) string {
	v.width, v.height = width, height
	if v.state == aiDisabled {
		return v.viewDisabled(width, height)
	}
	chatH := height - 3 // input box is 3 rows tall
	if chatH < 3 {
		chatH = 3
	}
	var body string
	if len(v.lines) == 0 {
		body = v.viewWelcome(width, chatH)
	} else {
		body = v.renderChat(width, chatH)
	}
	return body + "\n" + v.renderInput(width)
}

// viewWelcome renders a card-style splash when no conversation has started.
func (v *aiView) viewWelcome(width, height int) string {
	cardWidth := width - 4
	if cardWidth > 88 {
		cardWidth = 88
	}
	lines := []string{
		panelTitleStyle.Render("Welcome to AI Cleanse"),
		"",
		panelDescStyle.Render("Ask in plain English. I'll scan your disk, explain what's"),
		panelDescStyle.Render("eating space, and propose deletions one category at a time."),
		panelDescStyle.Render("You approve each step — nothing is removed without your y."),
		"",
		sectionStyle.Render("Try one of these"),
		"  " + chipStyle.Render("What's eating my disk?"),
		"  " + chipStyle.Render("Free up 10 GB safely"),
		"  " + chipStyle.Render("Show me only Docker cleanup"),
		"  " + chipStyle.Render("Be aggressive — I haven't touched these in months"),
		"",
		sectionStyle.Render("How approval works"),
		"  " + riskChip("SAFE") + "  " + dimStyle.Render(riskBlurb("SAFE")),
		"  " + riskChip("MEDIUM") + "  " + dimStyle.Render(riskBlurb("MEDIUM")),
		"  " + riskChip("DANGEROUS") + "  " + dimStyle.Render(riskBlurb("DANGEROUS")),
		"",
		dimStyle.Render("Dry-run is on by default. Press " + footerKeyStyle.Render("a") + " during a SAFE proposal to auto-approve the rest."),
	}
	card := cardStyle.Width(cardWidth).Render(strings.Join(lines, "\n"))
	// pad above to vertically center
	cardH := strings.Count(card, "\n") + 1
	pad := (height - cardH) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat("\n", pad) + lipgloss.PlaceHorizontal(width, lipgloss.Center, card)
}

func (v *aiView) viewDisabled(width, height int) string {
	cardWidth := width - 4
	if cardWidth > 88 {
		cardWidth = 88
	}
	lines := []string{
		panelTitleStyle.Render("AI Cleanse — set up in two steps"),
		"",
		panelDescStyle.Render("AI Cleanse uses Groq's free LLM tier. You bring your own key,"),
		panelDescStyle.Render("which stays on your machine. Nothing is sent except the key"),
		panelDescStyle.Render("and the scan summary you explicitly ask it to analyse."),
		"",
		sectionStyle.Render("1. Get a free Groq API key"),
		"  " + dimStyle.Render("Open ") + headerStyle.Render("https://console.groq.com/keys"),
		"  " + dimStyle.Render("Sign in, click ") + headerStyle.Render("Create API Key") + dimStyle.Render(", copy the value (starts with gsk_)."),
		"",
		sectionStyle.Render("2. Save it where lm can find it"),
		"  " + dimStyle.Render("Run in your shell:"),
		"",
		"    " + footerKeyStyle.Render("mkdir -p ~/.config/lean-mac"),
		"    " + footerKeyStyle.Render("cat > ~/.config/lean-mac/config.toml <<EOF"),
		"    " + footerKeyStyle.Render("groq_api_key = \"gsk_paste_yours_here\""),
		"    " + footerKeyStyle.Render("EOF"),
		"    " + footerKeyStyle.Render("chmod 600 ~/.config/lean-mac/config.toml"),
		"",
		dimStyle.Render("Then quit lm (q) and start again. The rest of lm (Scan, monitor,"),
		dimStyle.Render("doctor) works without any key."),
	}
	card := cardStyle.Width(cardWidth).Render(strings.Join(lines, "\n"))
	cardH := strings.Count(card, "\n") + 1
	pad := (height - cardH) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat("\n", pad) + lipgloss.PlaceHorizontal(width, lipgloss.Center, card)
}

func (v *aiView) renderChat(width, height int) string {
	// flatten lines into wrapped strings, then keep the last `height` rows.
	var rendered []string
	for _, ln := range v.lines {
		rendered = append(rendered, v.formatLine(ln, width)...)
	}
	if len(rendered) > height {
		rendered = rendered[len(rendered)-height:]
	}
	for len(rendered) < height {
		rendered = append(rendered, "")
	}
	return strings.Join(rendered, "\n")
}

func (v *aiView) formatLine(ln chatLine, width int) []string {
	var badge string
	var prefix string
	switch ln.role {
	case "user":
		badge = userBadgeStyle.Render(" you ")
	case "assistant":
		badge = aiBadgeStyle.Render(" ai ")
	default:
		badge = systemBadgeStyle.Render("·")
	}
	prefix = badge + " "
	indent := strings.Repeat(" ", lipgloss.Width(prefix))
	body := ln.text
	maxWidth := width - lipgloss.Width(prefix) - 2
	if maxWidth < 20 {
		maxWidth = 20
	}
	wrapped := wrap(body, maxWidth)
	out := make([]string, 0, len(wrapped))
	for i, w := range wrapped {
		if i == 0 {
			out = append(out, prefix+w)
		} else {
			out = append(out, indent+w)
		}
	}
	return out
}

func (v *aiView) renderInput(width int) string {
	prompt := "› "
	cursor := "█"
	if v.state == aiThinking || v.state == aiExecuting {
		cursor = " "
	}
	body := prompt + v.input + cursor
	style := inputBoxStyle
	if v.state == aiIdle {
		style = inputActiveStyle
	}
	return style.Width(width - 2).Render(body)
}

// wrap splits s into lines no longer than width, preserving existing newlines.
func wrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		if paragraph == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, paragraph)
			continue
		}
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if len(line)+1+len(w) > width {
				out = append(out, line)
				line = w
			} else {
				line += " " + w
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
