package zeroline

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDrawerRendersSessions(t *testing.T) {
	d := chatWith([]Row{{Kind: "user", Text: "hi"}, {Kind: "assistant", Text: "hello"}})
	d.Width, d.Height = 110, 22
	d.Drawer = &Drawer{Sessions: DefaultSessions(), Selected: 0}
	out := RenderChat(d)
	if h := lipgloss.Height(out); h != 22 {
		t.Fatalf("drawer frame height = %d, want 22 (frame-exact)", h)
	}
	for _, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > 110 {
			t.Fatalf("drawer line exceeds width 110: %d", lipgloss.Width(line))
		}
	}
	plain := stripANSI(out)
	for _, want := range []string{"sessions", "resume / fork / export", "0a91f3", "Add streaming SSE", "14 turns"} {
		if !strings.Contains(plain, want) {
			t.Errorf("drawer missing %q", want)
		}
	}
}

func TestDrawerSelectionRail(t *testing.T) {
	s := newStyles(Resolve(0, true), 0, true)
	panel := strings.Join(s.drawerPanel(&Drawer{Sessions: DefaultSessions(), Selected: 1}, 40, 20), "\n")
	// the selected (2nd) row carries the accent rail
	if !strings.Contains(stripANSI(panel), "▌") {
		t.Error("selected session row should show the accent rail")
	}
}
