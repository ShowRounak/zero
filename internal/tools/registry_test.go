package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gitlawb/zero/internal/sandbox"
)

func TestCoreReadOnlyToolsExposeSafeMetadata(t *testing.T) {
	toolset := CoreReadOnlyTools(t.TempDir())
	if len(toolset) != 6 {
		t.Fatalf("expected 6 core read-only tools, got %d", len(toolset))
	}

	for _, tool := range toolset {
		if tool.Name() == "" {
			t.Fatalf("tool has empty name")
		}
		if tool.Description() == "" {
			t.Fatalf("%s has empty description", tool.Name())
		}
		if tool.Safety().SideEffect != SideEffectRead {
			t.Fatalf("%s side effect = %s, want read", tool.Name(), tool.Safety().SideEffect)
		}
		if tool.Safety().Permission != PermissionAllow {
			t.Fatalf("%s permission = %s, want allow", tool.Name(), tool.Safety().Permission)
		}
		if tool.Safety().Reason == "" {
			t.Fatalf("%s has empty safety reason", tool.Name())
		}

		schema := tool.Parameters()
		if schema.Type != "object" {
			t.Fatalf("%s schema type = %s, want object", tool.Name(), schema.Type)
		}
		if schema.Properties == nil {
			t.Fatalf("%s schema properties are nil", tool.Name())
		}
		if schema.AdditionalProperties {
			t.Fatalf("%s schema should disallow additional properties", tool.Name())
		}
	}
}

func TestRegistryRunsToolsThroughSafePath(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewReadFileTool(t.TempDir()))

	result := registry.Run(context.Background(), "read_file", map[string]any{
		"path": "missing.txt",
	})

	if result.Status != StatusError {
		t.Fatalf("expected read error status, got %s", result.Status)
	}
	if result.Output == "" {
		t.Fatalf("expected an error output")
	}
}

func TestRegistryWithoutReturnsFilteredRegistry(t *testing.T) {
	root := t.TempDir()
	registry := NewRegistry()
	registry.Register(NewReadFileTool(root))
	registry.Register(NewAskUserTool())
	registry.Register(NewTaskTool())

	filtered := registry.Without("task", "ask_user")

	if _, ok := filtered.Get("task"); ok {
		t.Fatalf("expected task to be removed from filtered registry")
	}
	if _, ok := filtered.Get("ask_user"); ok {
		t.Fatalf("expected ask_user to be removed from filtered registry")
	}
	if _, ok := filtered.Get("read_file"); !ok {
		t.Fatalf("expected read_file to survive filtering")
	}
	if len(filtered.All()) != 1 {
		t.Fatalf("expected exactly one tool after filtering, got %d", len(filtered.All()))
	}

	// The original registry must be untouched.
	if _, ok := registry.Get("task"); !ok {
		t.Fatalf("Without must not mutate the source registry (task missing)")
	}
	if len(registry.All()) != 3 {
		t.Fatalf("expected source registry to keep all 3 tools, got %d", len(registry.All()))
	}
}

// Finding 5: a sub-agent's child registry (built via Without) must get an
// ISOLATED update_plan instance, so the sub-agent's plan calls do NOT clobber
// the parent's plan.
func TestRegistryWithoutIsolatesUpdatePlan(t *testing.T) {
	parentPlan := NewUpdatePlanTool()
	registry := NewRegistry()
	registry.Register(parentPlan)

	// Parent records a plan.
	if res := registry.Run(context.Background(), "update_plan", map[string]any{
		"plan": []any{map[string]any{"content": "parent step"}},
	}); res.Status != StatusOK {
		t.Fatalf("parent update_plan failed: %s", res.Output)
	}

	child := registry.Without("task", "ask_user")
	childTool, ok := child.Get("update_plan")
	if !ok {
		t.Fatalf("child registry must still expose update_plan")
	}
	// The child must NOT be the same instance as the parent's.
	if childTool == Tool(parentPlan) {
		t.Fatalf("child update_plan must be an isolated instance, got the parent's")
	}

	// Sub-agent records its own (different) plan.
	if res := child.Run(context.Background(), "update_plan", map[string]any{
		"plan": []any{map[string]any{"content": "child step"}},
	}); res.Status != StatusOK {
		t.Fatalf("child update_plan failed: %s", res.Output)
	}

	// Parent's plan must be untouched.
	parentItems := parentPlan.CurrentPlan()
	if len(parentItems) != 1 || parentItems[0].Content != "parent step" {
		t.Fatalf("parent plan clobbered by sub-agent: %+v", parentItems)
	}

	// Child's own plan reflects the child step.
	childReader, ok := childTool.(*updatePlanTool)
	if !ok {
		t.Fatalf("child update_plan has unexpected type %T", childTool)
	}
	childItems := childReader.CurrentPlan()
	if len(childItems) != 1 || childItems[0].Content != "child step" {
		t.Fatalf("child plan = %+v, want [child step]", childItems)
	}
}

func TestRegistryWithoutIgnoresUnknownNames(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewReadFileTool(t.TempDir()))

	filtered := registry.Without("does_not_exist")
	if len(filtered.All()) != 1 {
		t.Fatalf("expected unknown names to be ignored, got %d tools", len(filtered.All()))
	}
}

func TestCoreToolsIncludesTask(t *testing.T) {
	var found bool
	for _, tool := range CoreTools(t.TempDir()) {
		if tool.Name() == "task" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected CoreTools to include the task tool")
	}
}

func TestRegistryReportsUnknownTools(t *testing.T) {
	result := NewRegistry().Run(context.Background(), "missing", map[string]any{})

	if result.Status != StatusError {
		t.Fatalf("expected error status, got %s", result.Status)
	}
	if result.Output != `Error: Unknown tool "missing".` {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}

func TestRegistryAppliesSandboxBeforeToolExecution(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "escape.txt")
	registry := NewRegistry()
	registry.Register(NewWriteFileTool(root))
	engine := sandbox.NewEngine(sandbox.EngineOptions{
		WorkspaceRoot: root,
		Policy:        sandbox.DefaultPolicy(),
	})

	result := registry.RunWithOptions(context.Background(), "write_file", map[string]any{
		"path":      outside,
		"content":   "escape",
		"overwrite": true,
	}, RunOptions{
		PermissionGranted: true,
		Sandbox:           engine,
		PermissionMode:    string(sandbox.PermissionUnsafe),
		Autonomy:          string(sandbox.AutonomyHigh),
	})

	if result.Status != StatusError {
		t.Fatalf("expected sandbox violation status, got %s", result.Status)
	}
	if !strings.Contains(result.Output, "Sandbox violation") || !strings.Contains(result.Output, "outside_workspace") {
		t.Fatalf("unexpected sandbox violation output: %q", result.Output)
	}
}

func TestRegistryAllowsPromptToolWithPersistentSandboxGrant(t *testing.T) {
	root := t.TempDir()
	store, err := sandbox.NewGrantStore(sandbox.StoreOptions{
		FilePath: filepath.Join(t.TempDir(), "sandbox-grants.json"),
	})
	if err != nil {
		t.Fatalf("NewGrantStore returned error: %v", err)
	}
	if _, err := store.Grant(sandbox.GrantInput{
		ToolName:    "write_file",
		Decision:    sandbox.GrantAllow,
		MaxAutonomy: sandbox.AutonomyMedium,
		Reason:      "workspace writes",
	}); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	registry := NewRegistry()
	registry.Register(NewWriteFileTool(root))
	engine := sandbox.NewEngine(sandbox.EngineOptions{
		WorkspaceRoot: root,
		Policy:        sandbox.DefaultPolicy(),
		Store:         store,
	})

	result := registry.RunWithOptions(context.Background(), "write_file", map[string]any{
		"path":      "granted.txt",
		"content":   "granted",
		"overwrite": true,
	}, RunOptions{
		PermissionGranted: false,
		Sandbox:           engine,
		PermissionMode:    string(sandbox.PermissionModeAsk),
		Autonomy:          string(sandbox.AutonomyMedium),
	})

	if result.Status != StatusOK {
		t.Fatalf("expected persistent sandbox grant to authorize write_file, got %s: %s", result.Status, result.Output)
	}
	content, err := os.ReadFile(filepath.Join(root, "granted.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(content) != "granted" {
		t.Fatalf("written content = %q, want granted", string(content))
	}
}

type secretTool struct{ out string }

func (t secretTool) Name() string             { return "secret_tool" }
func (t secretTool) Description() string       { return "emits text" }
func (t secretTool) Parameters() Schema        { return Schema{Type: "object", AdditionalProperties: false} }
func (t secretTool) Safety() Safety            { return Safety{SideEffect: SideEffectRead, Permission: PermissionAllow} }
func (t secretTool) Run(context.Context, map[string]any) Result {
	return Result{Status: StatusOK, Output: t.out}
}

// Regression: secrets must be scrubbed at the registry boundary so EVERY caller
// (agent loop AND MCP server) gets redacted output — not just the agent path.
func TestRunWithOptionsScrubsSecretsForAllCallers(t *testing.T) {
	secret := "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	reg := NewRegistry()
	reg.Register(secretTool{out: "token=" + secret})

	res := reg.RunWithOptions(context.Background(), "secret_tool", map[string]any{}, RunOptions{PermissionGranted: true})
	if res.Status != StatusOK {
		t.Fatalf("status=%s output=%s", res.Status, res.Output)
	}
	if strings.Contains(res.Output, secret) {
		t.Fatalf("registry must scrub secrets, leaked: %q", res.Output)
	}
	if !res.Redacted {
		t.Error("expected Redacted=true")
	}
}

func TestRunWithOptionsLeavesCleanOutputUnchanged(t *testing.T) {
	reg := NewRegistry()
	reg.Register(secretTool{out: "nothing secret here"})
	res := reg.RunWithOptions(context.Background(), "secret_tool", map[string]any{}, RunOptions{PermissionGranted: true})
	if res.Redacted || res.Output != "nothing secret here" {
		t.Fatalf("clean output altered: redacted=%v output=%q", res.Redacted, res.Output)
	}
}
