package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Gitlawb/zero/internal/tools"
)

const (
	// planPanelOuterWidth is the column width the docked plan panel occupies,
	// including its border. The chat area gives up exactly this many columns.
	planPanelOuterWidth = 34
	// planPanelMinChat is the narrowest the chat may get; below it the panel is
	// hidden so the transcript never becomes unreadable on small terminals.
	planPanelMinChat = 52
)

// planPanelActive reports whether the right-docked plan/progress panel shows: an
// alt-screen chat (not setup), a wide-enough terminal, and either a live plan or
// a run in flight. When false, chatAreaWidth == chatWidth, so every existing
// render and hit-test path behaves exactly as before.
func (m model) planPanelActive() bool {
	if !m.altScreen || m.height <= 0 || m.setup.visible || m.transcriptDetailed {
		return false
	}
	if chatWidth(m.width)-planPanelOuterWidth < planPanelMinChat {
		return false
	}
	return m.pending || len(m.currentPlanItems()) > 0
}

// reservedPanelWidth is the columns the plan panel takes from the chat area.
func (m model) reservedPanelWidth() int {
	if m.planPanelActive() {
		return planPanelOuterWidth
	}
	return 0
}

// chatAreaWidth is the width available to the transcript, composer, and overlays
// once the docked plan panel (if any) has taken its column. It is the single
// source of truth both the renderers and the mouse hit-tests use, so the panel
// can never desync the click coordinates from what's drawn.
func (m model) chatAreaWidth() int {
	return chatWidth(m.width) - m.reservedPanelWidth()
}

// currentPlanItems returns the live update_plan steps, or nil.
func (m model) currentPlanItems() []tools.PlanItem {
	if m.registry == nil {
		return nil
	}
	tool, ok := m.registry.Get("update_plan")
	if !ok {
		return nil
	}
	reader, ok := tool.(currentPlanReader)
	if !ok {
		return nil
	}
	return reader.CurrentPlan()
}

// runningToolName returns the name of the most recent tool call that has not yet
// produced a result (the one currently executing), or "".
func (m model) runningToolName() string {
	rc := buildRowContext(m.transcript)
	for index := len(m.transcript) - 1; index >= 0; index-- {
		row := m.transcript[index]
		if row.kind == rowToolCall && row.id != "" && !rc.resolved[rcKey(row.runID, row.id)] {
			return row.tool
		}
	}
	return ""
}

// planActivityLabel is the human word for what the agent is doing right now —
// the "is it stuck?" signal. Empty when no run is in flight.
func (m model) planActivityLabel() string {
	if !m.pending {
		return ""
	}
	if strings.TrimSpace(m.streamingText) != "" {
		return "Responding"
	}
	switch toolActivityKind(m.runningToolName()) {
	case "plan":
		return "Planning"
	case "build":
		return "Building"
	case "scan":
		return "Scanning"
	case "shell":
		return "Running"
	case "search":
		return "Searching"
	default:
		return "Thinking"
	}
}

// toolActivityKind buckets a tool name into a coarse activity for the label.
func toolActivityKind(tool string) string {
	switch tool {
	case "update_plan":
		return "plan"
	case "write_file", "edit_file", "apply_patch", "create_file", "str_replace", "multi_edit":
		return "build"
	case "read_file", "list_dir", "grep", "glob", "ripgrep", "search_files", "list_files":
		return "scan"
	case "bash", "shell", "run", "exec":
		return "shell"
	case "web_search", "web_fetch", "fetch", "search":
		return "search"
	default:
		return ""
	}
}

// planStatusGlyph maps a plan step's status to a glyph + style.
func planStatusGlyph(status string) (string, lipgloss.Style) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "in_progress", "in-progress", "active", "doing":
		return "◐", zeroTheme.accent
	case "done", "completed", "complete":
		return "✓", zeroTheme.accent
	case "blocked", "failed", "error":
		return "✗", zeroTheme.red
	default: // pending / todo / ""
		return "○", zeroTheme.faint
	}
}

func formatPanelElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

// renderPlanPanel draws the docked panel as exactly `height` lines of
// planPanelOuterWidth columns: a header with the live activity (spinner + word +
// elapsed) and the plan steps with status glyphs, framed in a left-separated box.
func (m model) renderPlanPanel(height int) string {
	if height < 2 {
		height = 2
	}
	inner := planPanelOuterWidth - 4 // "│ " prefix + " │" suffix

	header := "Plan"
	if label := m.planActivityLabel(); label != "" {
		header = strings.TrimSpace(m.spinner.View()) + " " + label
		if !m.runStartedAt.IsZero() {
			header += "  " + formatPanelElapsed(m.now().Sub(m.runStartedAt))
		}
	}

	body := []string{zeroTheme.accent.Bold(true).Render(cutRunes(header, inner))}
	body = append(body, "")
	items := m.currentPlanItems()
	if len(items) == 0 {
		body = append(body, zeroTheme.faint.Render(cutRunes("waiting for a plan…", inner)))
	} else {
		for index, item := range items {
			glyph, style := planStatusGlyph(item.Status)
			label := fmt.Sprintf("%d %s", index+1, item.Content)
			body = append(body, style.Render(glyph)+" "+zeroTheme.ink.Render(cutRunes(label, inner-2)))
		}
	}

	return framePlanPanel(body, height, inner)
}

// framePlanPanel boxes body into exactly `height` lines of planPanelOuterWidth,
// padding or clipping the body to fit, with the border doubling as the separator
// from the chat area on its left.
func framePlanPanel(body []string, height int, inner int) string {
	border := zeroTheme.faint
	rule := strings.Repeat("─", planPanelOuterWidth-2)
	bodyRows := maxInt(0, height-2)
	if len(body) > bodyRows {
		body = body[:bodyRows]
	}
	for len(body) < bodyRows {
		body = append(body, "")
	}
	lines := make([]string, 0, height)
	lines = append(lines, border.Render("╭"+rule+"╮"))
	for _, row := range body {
		fitted := fitStyledLine(row, inner)
		pad := maxInt(0, inner-lipgloss.Width(fitted))
		lines = append(lines, border.Render("│ ")+fitted+strings.Repeat(" ", pad)+border.Render(" │"))
	}
	lines = append(lines, border.Render("╰"+rule+"╯"))
	return strings.Join(lines, "\n")
}

// composeWithPlanPanel docks the plan panel onto the right of the rendered chat
// content, row by row. The chat content is already rendered at chatAreaWidth, so
// each line is padded to that width and the matching panel line is appended. A
// no-op when the panel is inactive.
func (m model) composeWithPlanPanel(content string) string {
	if !m.planPanelActive() {
		return content
	}
	lines := strings.Split(content, "\n")
	chatW := m.chatAreaWidth()
	panelLines := strings.Split(m.renderPlanPanel(len(lines)), "\n")
	out := make([]string, len(lines))
	for index, line := range lines {
		left := fitStyledLine(line, chatW)
		left += strings.Repeat(" ", maxInt(0, chatW-lipgloss.Width(left)))
		right := ""
		if index < len(panelLines) {
			right = panelLines[index]
		}
		out[index] = left + right
	}
	return strings.Join(out, "\n")
}
