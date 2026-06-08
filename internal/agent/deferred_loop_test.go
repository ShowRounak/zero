package agent

import (
	"context"
	"testing"

	"github.com/Gitlawb/zero/internal/tools"
)

func TestOptionsDeferThresholdFieldExists(t *testing.T) {
	options := Options{DeferThreshold: 10}
	if options.DeferThreshold != 10 {
		t.Fatalf("expected DeferThreshold 10, got %d", options.DeferThreshold)
	}
}

func TestToolResultLoadedToolsField(t *testing.T) {
	result := ToolResult{LoadedTools: []string{"Alpha", "Beta"}}
	if len(result.LoadedTools) != 2 || result.LoadedTools[0] != "Alpha" || result.LoadedTools[1] != "Beta" {
		t.Fatalf("expected LoadedTools [Alpha Beta], got %#v", result.LoadedTools)
	}
	// Default zero value is nil for an ordinary result.
	if (ToolResult{}).LoadedTools != nil {
		t.Fatalf("expected nil LoadedTools by default")
	}
}

// loadSignalTool returns Meta["load_tools"] like tool_search does, so we can
// assert executeToolCall lifts it into ToolResult.LoadedTools.
type loadSignalTool struct{ value string }

func (t loadSignalTool) Name() string       { return "load_signal" }
func (t loadSignalTool) Description() string { return "emits a load_tools signal" }
func (t loadSignalTool) Parameters() tools.Schema {
	return tools.Schema{Type: "object", AdditionalProperties: false}
}
func (t loadSignalTool) Safety() tools.Safety {
	return tools.Safety{SideEffect: tools.SideEffectRead, Permission: tools.PermissionAllow}
}
func (t loadSignalTool) Run(_ context.Context, _ map[string]any) tools.Result {
	return tools.Result{Status: tools.StatusOK, Output: "ok"}
}
func (t loadSignalTool) RunWithOptions(_ context.Context, _ map[string]any, _ tools.RunOptions) tools.Result {
	return tools.Result{Status: tools.StatusOK, Output: "ok", Meta: map[string]string{"load_tools": t.value}}
}

func TestExecuteToolCallLiftsLoadTools(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(loadSignalTool{value: " Alpha , Beta ,, "})

	result, abortErr := executeToolCall(
		context.Background(),
		registry,
		ToolCall{ID: "c1", Name: "load_signal", Arguments: ""},
		PermissionModeAuto,
		Options{},
	)
	if abortErr != nil {
		t.Fatalf("unexpected abort error: %v", abortErr)
	}
	want := []string{"Alpha", "Beta"}
	if len(result.LoadedTools) != len(want) {
		t.Fatalf("expected LoadedTools %#v, got %#v", want, result.LoadedTools)
	}
	for i := range want {
		if result.LoadedTools[i] != want[i] {
			t.Fatalf("expected LoadedTools %#v, got %#v", want, result.LoadedTools)
		}
	}
}

func TestExecuteToolCallNoLoadToolsMetaLeavesNil(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(secretEmittingTool{output: "plain"})

	result, abortErr := executeToolCall(
		context.Background(),
		registry,
		ToolCall{ID: "c1", Name: "leak", Arguments: ""},
		PermissionModeAuto,
		Options{},
	)
	if abortErr != nil {
		t.Fatalf("unexpected abort error: %v", abortErr)
	}
	if result.LoadedTools != nil {
		t.Fatalf("expected nil LoadedTools for a tool with no load_tools meta, got %#v", result.LoadedTools)
	}
}
