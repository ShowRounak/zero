package tui

import (
	"context"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gitlawb/zero/internal/notify"
)

// Run starts the Zero Bubble Tea shell and returns a process-style exit code.
func Run(ctx context.Context, options Options) int {
	externalSink := options.RuntimeMessageSink
	var program *tea.Program
	options.RuntimeMessageSink = func(msg tea.Msg) {
		if externalSink != nil {
			externalSink(msg)
		}
		if program != nil {
			program.Send(msg)
		}
	}

	programOpts := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	}
	if notify.Enabled(notify.Mode(strings.TrimSpace(options.Notify.Mode))) {
		programOpts = append(programOpts, tea.WithReportFocus())
	}
	// Enable mouse cell-motion so the permission card's buttons are clickable.
	// Tradeoff: this routes clicks/drags to the program and can interfere with the
	// terminal's native text selection (most terminals let users hold Shift/Option
	// to force native selection). The card stays fully keyboard-driven (a/A/d/Esc).
	programOpts = append(programOpts, tea.WithMouseCellMotion())
	program = tea.NewProgram(newModel(ctx, options), programOpts...)

	if _, err := program.Run(); err != nil {
		return 1
	}
	return 0
}
