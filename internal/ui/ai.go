package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/ai"
	"github.com/bagaspra16/lean-mac/internal/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// aiState is the high-level state of the AI loop.
type aiState int

const (
	aiIdle          aiState = iota // waiting for user input
	aiThinking                     // request in flight
	aiAwaitApproval                // proposal shown, waiting on y/n/a/c
	aiExecuting                    // running an approved action
	aiDisabled                     // no keys
)

// chatLineKind controls how a line is rendered.
type chatLineKind int

const (
	kindUser      chatLineKind = iota
	kindAssistant              // AI text response
	kindThinking               // "AI is thinking" step indicator
	kindTool                   // tool call event
	kindScan                   // scan progress/done
	kindExec                   // execution result
	kindSystem                 // generic system notice
	kindDone                   // success result
	kindError                  // error
	kindProposal               // approval request
)

// chatLine is one entry in the scrollback.
type chatLine struct {
	kind chatLineKind
	role string // legacy "user"|"assistant"|"system"|"tool"
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

	scrollOffset int // manual scroll in approval view
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

func (v *aiView) Title() string { return "AI Cleanse" }

func (v *aiView) Subtitle() string {
	switch v.state {
	case aiDisabled:
		return "not configured · see setup below"
	case aiThinking:
		return "⠿ thinking…"
	case aiAwaitApproval:
		return "⚡ awaiting your approval"
	case aiExecuting:
		return "⚙ executing approved action"
	default:
		return "conversational cleanup · per-action approval"
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
			{"y", "approve & execute"},
			{"n", "reject & skip"},
			{"a", "auto-approve SAFE"},
			{"c", "cancel session"},
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
func (v *aiView) wantsKey(msg tea.KeyMsg) bool {
	if v.state == aiDisabled {
		return false
	}
	if v.state == aiAwaitApproval {
		return false
	}
	if v.state == aiThinking || v.state == aiExecuting {
		return false
	}
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
			v.appendKind(kindError, "error: "+m.err.Error())
			v.state = aiIdle
		}
		v.convo = m.msgs
		if v.state == aiThinking && v.pending == nil {
			v.state = aiIdle
		}
		return v, waitAIEvent(v.events)
	case aiExecDoneMsg:
		v.appendKind(kindDone, fmt.Sprintf("✓ Done — freed %s.", humanize.IBytes(uint64(m.freed))))
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
		v.appendKind(kindThinking, "AI is processing your request…")
	case ai.EvtAssistant:
		// Remove last thinking line if present
		if len(v.lines) > 0 && v.lines[len(v.lines)-1].kind == kindThinking {
			v.lines = v.lines[:len(v.lines)-1]
		}
		v.appendKind(kindAssistant, ev.Text)
	case ai.EvtScanStart:
		v.appendKind(kindScan, "🔍 Scanning disk — discovering reclaimable artifacts…")
	case ai.EvtScanDone:
		if ev.Report != nil {
			v.appendKind(kindScan, fmt.Sprintf(
				"✓ Scan complete — %d findings · %s reclaimable · %s free of %s",
				len(ev.Report.Findings),
				humanize.IBytes(uint64(ev.Report.TotalBytes)),
				humanize.IBytes(uint64(ev.Report.DiskFree)),
				humanize.IBytes(uint64(ev.Report.DiskTotal)),
			))
		}
	case ai.EvtProposal:
		v.pending = ev.Action
		v.state = aiAwaitApproval
		v.scrollOffset = 0
		// Don't add a chatLine for proposal — it renders as the approval panel
		if v.autoSafe && ev.Action.Risk == types.RiskSafe {
			return v, v.approve()
		}
	case ai.EvtExecutionStart:
		if ev.Action != nil {
			v.appendKind(kindExec, fmt.Sprintf("⚙ Executing: %s (%d items)…",
				ev.Action.Category, len(ev.Action.Findings)))
		}
	case ai.EvtExecutionResult:
		if ev.Result != nil {
			if ev.Result.Success {
				v.appendKind(kindDone, fmt.Sprintf("  ✓ %s  %s",
					ev.Result.Finding.Path,
					humanize.IBytes(uint64(ev.Result.BytesFreed))))
			} else {
				v.appendKind(kindError, fmt.Sprintf("  ✗ %s  %s",
					ev.Result.Finding.Path, ev.Result.Error))
			}
		}
	case ai.EvtError:
		v.appendKind(kindError, "✗ Error: "+ev.Err.Error())
		v.state = aiIdle
	}
	return v, waitAIEvent(v.events)
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
			v.appendKind(kindSystem, "Auto-approve SAFE proposals enabled.")
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
		v.appendKind(kindUser, text)
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
	v.appendKind(kindSystem, fmt.Sprintf("✔ Approved — executing %s…", action.Category))
	dry := v.dryRun
	events := v.events
	return func() tea.Msg {
		ai.Execute(context.Background(), action, dry, events)
		return aiExecDoneMsg{freed: action.Total}
	}
}

func (v *aiView) reject() tea.Cmd {
	action := v.pending
	v.pending = nil
	v.appendKind(kindSystem, fmt.Sprintf("✖ Rejected — skipping %s.", action.Category))
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
	v.appendKind(kindSystem, "Session cancelled.")
	return nil
}

func (v *aiView) runTurn() tea.Cmd {
	convo := v.convo
	agent := v.agent
	events := v.events
	ctx, cancel := context.WithCancel(context.Background())
	v.cancel = cancel
	return func() tea.Msg {
		defer cancel()
		for i := 0; i < 4; i++ {
			next, err := agent.Step(ctx, convo, events)
			if err != nil {
				return aiTurnDoneMsg{msgs: convo, err: err}
			}
			convo = next
			last := convo[len(convo)-1]
			if last.Role == "tool" {
				continue
			}
			break
		}
		return aiTurnDoneMsg{msgs: convo, err: nil}
	}
}

func (v *aiView) appendKind(kind chatLineKind, text string) {
	v.lines = append(v.lines, chatLine{kind: kind, text: text})
}

func (v *aiView) View(width, height int) string {
	v.width, v.height = width, height
	if v.state == aiDisabled {
		return v.viewDisabled(width, height)
	}
	// Reserve space: input box (3 rows) + approval panel if pending
	inputH := 3
	approvalH := 0
	if v.state == aiAwaitApproval && v.pending != nil {
		approvalH = v.approvalPanelHeight()
	}
	chatH := height - inputH - approvalH
	if chatH < 2 {
		chatH = 2
	}

	var sections []string
	if len(v.lines) == 0 {
		sections = append(sections, v.viewWelcome(width, chatH))
	} else {
		sections = append(sections, v.renderChat(width, chatH))
	}
	if v.state == aiAwaitApproval && v.pending != nil {
		sections = append(sections, v.renderApprovalPanel(width))
	}
	sections = append(sections, v.renderInput(width))
	return strings.Join(sections, "\n")
}

// viewWelcome renders a card-style splash when no conversation has started.
func (v *aiView) viewWelcome(width, height int) string {
	cardWidth := width - 4
	if cardWidth > 88 {
		cardWidth = 88
	}
	lines := []string{
		panelTitleStyle.Render("✦ Welcome to AI Cleanse"),
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
		sectionStyle.Render("How the AI process works"),
		"  " + thinkBadgeStyle.Render(" THINK ") + "  " + dimStyle.Render("AI analyses your request"),
		"  " + toolBadgeStyle.Render(" SCAN  ") + "  " + dimStyle.Render("Disk is scanned for artifacts"),
		"  " + scanBadgeStyle.Render(" PROP  ") + "  " + dimStyle.Render("AI proposes a cleanup category"),
		"  " + execBadgeStyle.Render(" EXEC  ") + "  " + dimStyle.Render("Files removed after your approval"),
		"  " + doneBadgeStyle.Render(" DONE  ") + "  " + dimStyle.Render("Result reported back to AI"),
		"",
		sectionStyle.Render("Risk levels"),
		"  " + riskChip("SAFE") + "  " + dimStyle.Render(riskBlurb("SAFE")),
		"  " + riskChip("MEDIUM") + "  " + dimStyle.Render(riskBlurb("MEDIUM")),
		"  " + riskChip("DANGEROUS") + "  " + dimStyle.Render(riskBlurb("DANGEROUS")),
		"",
		dimStyle.Render("Dry-run is ON by default. Press " + footerKeyStyle.Render("a") + " during a SAFE proposal to auto-approve all SAFE steps."),
	}
	card := cardStyle.Width(cardWidth).Render(strings.Join(lines, "\n"))
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
	switch ln.kind {
	case kindUser:
		badge = userBadgeStyle.Render(" you ")
	case kindAssistant:
		badge = aiBadgeStyle.Render(" ai  ")
	case kindThinking:
		spinner := thinkingSpinner()
		badge = thinkBadgeStyle.Render(" " + spinner + " ")
	case kindTool:
		badge = toolBadgeStyle.Render(" tool")
	case kindScan:
		badge = scanBadgeStyle.Render(" scan")
	case kindExec:
		badge = execBadgeStyle.Render(" exec")
	case kindDone:
		badge = doneBadgeStyle.Render(" done")
	case kindError:
		badge = errorBadgeStyle.Render(" err ")
	default:
		badge = systemBadgeStyle.Render("·")
	}
	prefix := badge + " "
	indent := strings.Repeat(" ", lipgloss.Width(prefix))
	maxWidth := width - lipgloss.Width(prefix) - 2
	if maxWidth < 20 {
		maxWidth = 20
	}
	wrapped := wrap(ln.text, maxWidth)
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

// approvalPanelHeight estimates the number of rows the approval panel will occupy.
func (v *aiView) approvalPanelHeight() int {
	if v.pending == nil {
		return 0
	}
	base := 10 // heading + risk + reason + items + action row + borders
	n := len(v.pending.Findings)
	if n > 5 {
		n = 5
	}
	return base + n
}

// renderApprovalPanel draws a rich, bordered approval dialog.
func (v *aiView) renderApprovalPanel(width int) string {
	a := v.pending
	if a == nil {
		return ""
	}
	risk := a.Risk.String()
	boxW := width - 6
	if boxW > 90 {
		boxW = 90
	}

	// Header line
	header := riskChip(risk) + "  " +
		lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("Cleanup Proposal") +
		"  " + dimStyle.Render(string(a.Category))

	// Size / count summary
	summary := fmt.Sprintf("%s reclaimable across %d item(s)",
		lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(humanize.IBytes(uint64(a.Total))),
		len(a.Findings))

	// Reason
	reasonLines := wrap(a.Reason, boxW-4)
	var reasonParts []string
	for _, rl := range reasonLines {
		reasonParts = append(reasonParts, "  "+dimStyle.Render(rl))
	}

	// File list (up to 5)
	var fileLines []string
	for i, f := range a.Findings {
		if i >= 5 {
			fileLines = append(fileLines, "  "+subtleStyle.Render(fmt.Sprintf("… and %d more", len(a.Findings)-5)))
			break
		}
		chip := riskChip(f.Risk.String())
		sz := mutedStyle.Render(fmt.Sprintf("%10s", humanize.IBytes(uint64(f.Size))))
		p := f.Path
		maxP := boxW - 28
		if maxP > 0 && len(p) > maxP {
			p = "…" + p[len(p)-maxP+1:]
		}
		fileLines = append(fileLines, fmt.Sprintf("  %s %s  %s", chip, sz, p))
	}

	// Action row
	modeTag := "DRY-RUN — no files will be deleted"
	if !v.dryRun {
		modeTag = "LIVE — files will be permanently deleted"
	}
	modeStr := dimStyle.Render("Mode: " + modeTag)

	actionRow := approveKeyStyle.Render(" y ") + " approve & execute   " +
		rejectKeyStyle.Render(" n ") + " reject & skip   " +
		autoKeyStyle.Render(" a ") + " auto-approve SAFE   " +
		cancelKeyStyle.Render(" c ") + " cancel"

	bodyLines := []string{
		header,
		"",
		summary,
		"",
	}
	bodyLines = append(bodyLines, reasonParts...)
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, fileLines...)
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, modeStr)
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, actionRow)

	box := approvalBoxForRisk(risk).Width(boxW).Render(strings.Join(bodyLines, "\n"))
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, box)
}

func (v *aiView) renderInput(width int) string {
	prompt := "› "
	cursor := "█"
	placeholder := ""
	if v.state == aiThinking {
		cursor = " "
		placeholder = dimStyle.Render("AI is thinking…")
	} else if v.state == aiExecuting {
		cursor = " "
		placeholder = dimStyle.Render("Executing…")
	} else if v.state == aiAwaitApproval {
		cursor = " "
		placeholder = dimStyle.Render("Waiting for your approval above ↑")
	}
	var body string
	if placeholder != "" && v.input == "" {
		body = prompt + placeholder
	} else {
		body = prompt + v.input + cursor
	}
	style := inputBoxStyle
	if v.state == aiIdle {
		style = inputActiveStyle
	}
	return style.Width(width - 2).Render(body)
}

// thinkingSpinner returns a single spinner frame based on current time.
func thinkingSpinner() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[(time.Now().UnixMilli()/120)%int64(len(frames))]
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
