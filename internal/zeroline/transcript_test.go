package zeroline

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func chatWith(rows []Row) ChatData {
	return ChatData{Variant: 0, Dark: true, Width: 90, Height: 24, Header: Header{Model: "m"}, Rows: rows}
}

func TestTranscriptBlocks(t *testing.T) {
	// user: accent ❯ + ink, no "you" label
	out := stripANSI(RenderChat(chatWith([]Row{{Kind: "user", Text: "refactor the loop"}})))
	if !strings.Contains(out, "❯ refactor the loop") {
		t.Errorf("user block missing ❯ + text")
	}
	if strings.Contains(out, "you ") {
		t.Errorf("user block should not show the 'you' label")
	}

	// assistant say: muted text, no "✦ zero" label
	out = stripANSI(RenderChat(chatWith([]Row{{Kind: "assistant", Text: "Here is the plan."}})))
	if !strings.Contains(out, "Here is the plan.") {
		t.Errorf("say text missing")
	}
	if strings.Contains(out, "✦ zero") {
		t.Errorf("say block should not show the zero label")
	}

	// final: accent rail + ink text
	out = stripANSI(RenderChat(chatWith([]Row{{Kind: "final", Text: "All done."}})))
	if !strings.Contains(out, "│") || !strings.Contains(out, "All done.") {
		t.Errorf("final block missing rail/text: %q", out)
	}

	// done: ● + faint meta
	out = stripANSI(RenderChat(chatWith([]Row{{Kind: "done", Text: "12 tools · 1,284 tok · $0.04", Status: "ok"}})))
	if !strings.Contains(out, "●") || !strings.Contains(out, "12 tools") {
		t.Errorf("done block missing dot/meta: %q", out)
	}

	// notes: sys + deny
	if out = stripANSI(RenderChat(chatWith([]Row{{Kind: "system", Text: "compacted older turns"}}))); !strings.Contains(out, "compacted older turns") {
		t.Errorf("sys note missing")
	}
	if out = stripANSI(RenderChat(chatWith([]Row{{Kind: "error", Text: "denied: bash"}}))); !strings.Contains(out, "denied: bash") {
		t.Errorf("deny note missing")
	}
}

func TestStreamingCaretOnlyMidStream(t *testing.T) {
	// Streaming → accent caret ▌ trails the say text.
	d := chatWith(nil)
	d.Stream = "partial answer"
	d.Working = true
	stream := stripANSI(RenderChat(d))
	if !strings.Contains(stream, "partial answer") || !strings.Contains(stream, "▌") {
		t.Errorf("streaming should show text + caret: %q", stream)
	}
	// Not streaming → no caret.
	still := stripANSI(RenderChat(chatWith([]Row{{Kind: "assistant", Text: "settled answer"}})))
	if strings.Contains(still, "▌") {
		t.Errorf("non-streaming transcript must not show the caret")
	}
}

func TestTranscriptFrameExact(t *testing.T) {
	d := chatWith([]Row{
		{Kind: "user", Text: "do the thing"},
		{Kind: "assistant", Text: "Working on it."},
		{Kind: "final", Text: "Finished."},
		{Kind: "done", Text: "2 tools · 100 tok", Status: "ok"},
	})
	d.Width, d.Height = 100, 22
	out := RenderChat(d)
	if h := lipgloss.Height(out); h != 22 {
		t.Fatalf("chat height = %d, want 22 (frame-exact)", h)
	}
	for _, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > 100 {
			t.Fatalf("chat line exceeds width 100: %d", lipgloss.Width(line))
		}
	}
}
