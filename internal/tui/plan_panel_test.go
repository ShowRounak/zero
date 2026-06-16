package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestPlanPanelActiveAndNarrowsChat(t *testing.T) {
	m := mouseTestModel() // alt-screen, width 100, height 30
	if m.planPanelActive() {
		t.Fatal("panel must be inactive when idle with no plan")
	}
	if m.chatAreaWidth() != chatWidth(m.width) {
		t.Fatal("chat area should be full width when the panel is hidden")
	}

	m.pending = true
	if !m.planPanelActive() {
		t.Fatal("panel should activate while a run is pending")
	}
	if m.chatAreaWidth() != chatWidth(m.width)-planPanelOuterWidth {
		t.Fatalf("chat area should narrow by the panel width, got %d", m.chatAreaWidth())
	}
}

func TestPlanPanelHiddenWhenNotApplicable(t *testing.T) {
	narrow := mouseTestModel()
	narrow.width = 60 // 60 - 34 < 52 → keep the transcript readable, hide the panel
	narrow.pending = true
	if narrow.planPanelActive() {
		t.Fatal("panel must hide on a narrow terminal")
	}
	if narrow.chatAreaWidth() != chatWidth(narrow.width) {
		t.Fatal("chat area should be full width when the panel is hidden")
	}

	inSetup := mouseTestModel()
	inSetup.pending = true
	inSetup.setup.visible = true
	if inSetup.planPanelActive() {
		t.Fatal("no panel during setup")
	}

	inline := mouseTestModel()
	inline.pending = true
	inline.altScreen = false
	if inline.planPanelActive() {
		t.Fatal("no panel in inline (non-alt-screen) mode")
	}
}

func TestComposeWithPlanPanelDocksOnRight(t *testing.T) {
	m := mouseTestModel()
	m.pending = true

	out := m.composeWithPlanPanel("hello\nworld\n")
	plain := plainRender(t, out)
	if !strings.Contains(plain, "╭") || !strings.Contains(plain, "╮") || !strings.Contains(plain, "╰") {
		t.Fatalf("expected a docked panel box, got:\n%s", plain)
	}
	// Every composed row spans chat + panel.
	full := m.chatAreaWidth() + planPanelOuterWidth
	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w != full {
			t.Fatalf("composed line width = %d, want %d: %q", w, full, plainRender(t, line))
		}
	}

	// Inactive panel → content is returned unchanged.
	idle := mouseTestModel()
	if got := idle.composeWithPlanPanel("hello"); got != "hello" {
		t.Fatalf("inactive compose should be a no-op, got %q", got)
	}
}

func TestPlanActivityLabel(t *testing.T) {
	m := mouseTestModel()
	if m.planActivityLabel() != "" {
		t.Fatal("no activity label when idle")
	}
	m.pending = true
	if got := m.planActivityLabel(); got != "Thinking" {
		t.Fatalf("pending with no tool = %q, want Thinking", got)
	}
	m.streamingText = "writing the answer"
	if got := m.planActivityLabel(); got != "Responding" {
		t.Fatalf("streaming = %q, want Responding", got)
	}
}

func TestPlanActivityFromRunningTool(t *testing.T) {
	m := mouseTestModel()
	m.pending = true
	m.transcript = appendTranscriptRow(m.transcript, transcriptRow{kind: rowToolCall, id: "c1", tool: "write_file", runID: 1})
	if got := m.planActivityLabel(); got != "Building" {
		t.Fatalf("running write_file = %q, want Building", got)
	}
	// A result resolves the call → no longer "running".
	m.transcript = appendTranscriptRow(m.transcript, transcriptRow{kind: rowToolResult, id: "c1", tool: "write_file", runID: 1})
	if got := m.planActivityLabel(); got != "Thinking" {
		t.Fatalf("resolved tool = %q, want Thinking", got)
	}
}

func TestToolActivityKindAndStatusGlyphs(t *testing.T) {
	kinds := map[string]string{
		"update_plan": "plan",
		"write_file":  "build",
		"edit_file":   "build",
		"read_file":   "scan",
		"grep":        "scan",
		"bash":        "shell",
		"web_search":  "search",
		"mystery":     "",
	}
	for tool, want := range kinds {
		if got := toolActivityKind(tool); got != want {
			t.Fatalf("toolActivityKind(%q) = %q, want %q", tool, got, want)
		}
	}
	for status, want := range map[string]string{"in_progress": "◐", "done": "✓", "pending": "○", "": "○", "failed": "✗"} {
		if got, _ := planStatusGlyph(status); got != want {
			t.Fatalf("planStatusGlyph(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestFormatPanelElapsed(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                 "0:00",
		12 * time.Second:  "0:12",
		83 * time.Second:  "1:23",
		605 * time.Second: "10:05",
	} {
		if got := formatPanelElapsed(d); got != want {
			t.Fatalf("formatPanelElapsed(%s) = %q, want %q", d, got, want)
		}
	}
}
