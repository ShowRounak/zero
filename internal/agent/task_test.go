package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Gitlawb/zero/internal/tools"
	"github.com/Gitlawb/zero/internal/zeroruntime"
)

// routingProvider serves different scripted turns to the parent run vs a child
// (sub-agent) run, distinguishing them by the user prompt that seeds each run.
// The parent run carries parentPrompt; the child run carries the task prompt.
// It records every request so tests can assert what each run advertised/sent.
type routingProvider struct {
	parentSubstr string
	parentTurns  [][]zeroruntime.StreamEvent
	childTurns   [][]zeroruntime.StreamEvent

	requests      []zeroruntime.CompletionRequest
	parentCount   int
	childCount    int
	childRequests []zeroruntime.CompletionRequest
}

func (provider *routingProvider) StreamCompletion(_ context.Context, request zeroruntime.CompletionRequest) (<-chan zeroruntime.StreamEvent, error) {
	provider.requests = append(provider.requests, request)

	isParent := false
	for _, message := range request.Messages {
		if message.Role == zeroruntime.MessageRoleUser && strings.Contains(message.Content, provider.parentSubstr) {
			isParent = true
			break
		}
	}

	var events []zeroruntime.StreamEvent
	if isParent {
		if provider.parentCount < len(provider.parentTurns) {
			events = provider.parentTurns[provider.parentCount]
		}
		provider.parentCount++
	} else {
		provider.childRequests = append(provider.childRequests, request)
		if provider.childCount < len(provider.childTurns) {
			events = provider.childTurns[provider.childCount]
		}
		provider.childCount++
	}
	if events == nil {
		events = []zeroruntime.StreamEvent{{Type: zeroruntime.StreamEventDone}}
	}

	ch := make(chan zeroruntime.StreamEvent, len(events))
	for _, event := range events {
		ch <- event
	}
	close(ch)
	return ch, nil
}

// taskCallTurn scripts a turn that invokes the task tool with the given child
// prompt. It reuses toolTurn (defined in guardrails_test.go).
func taskCallTurn(prompt string) []zeroruntime.StreamEvent {
	return toolTurn("task-1", "task", `{"description":"sub work","prompt":`+quoteJSONString(prompt)+`}`)
}

// (a) A top-level run where the model calls task spawns a sub-run, and the
// child's final answer comes back as the task tool result.
func TestRunSpawnsSubAgentForTaskCall(t *testing.T) {
	const childPrompt = "investigate the failing test"
	registry := tools.NewRegistry()
	registry.Register(tools.NewTaskTool())
	registry.Register(tools.NewReadFileTool(t.TempDir()))

	provider := &routingProvider{
		parentSubstr: "delegate please",
		parentTurns: [][]zeroruntime.StreamEvent{
			taskCallTurn(childPrompt),  // parent turn 1: call task
			textTurn("parent done"),    // parent turn 2: final answer
		},
		childTurns: [][]zeroruntime.StreamEvent{
			textTurn("child summary: the test was flaky"), // child turn 1: final answer
		},
	}

	var toolResults []ToolResult
	result, err := Run(context.Background(), "delegate please", provider, Options{
		Registry:     registry,
		MaxTurns:     5,
		OnToolResult: func(r ToolResult) { toolResults = append(toolResults, r) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalAnswer != "parent done" {
		t.Fatalf("expected parent final answer, got %q", result.FinalAnswer)
	}
	if provider.childCount != 1 {
		t.Fatalf("expected exactly one child turn, got %d", provider.childCount)
	}
	if len(toolResults) != 1 {
		t.Fatalf("expected one task tool result, got %d", len(toolResults))
	}
	if toolResults[0].Status != tools.StatusOK {
		t.Fatalf("expected ok task result, got %s: %s", toolResults[0].Status, toolResults[0].Output)
	}
	if toolResults[0].Output != "child summary: the test was flaky" {
		t.Fatalf("expected child final answer as task output, got %q", toolResults[0].Output)
	}
	if !strings.Contains(toolResults[0].Display.Summary, "sub work") {
		t.Fatalf("expected display summary to name the sub-task, got %q", toolResults[0].Display.Summary)
	}

	// The child run must have been seeded with the task prompt.
	if len(provider.childRequests) == 0 {
		t.Fatalf("expected the child run to issue at least one request")
	}
	var sawChildPrompt bool
	for _, message := range provider.childRequests[0].Messages {
		if message.Role == zeroruntime.MessageRoleUser && strings.Contains(message.Content, childPrompt) {
			sawChildPrompt = true
		}
	}
	if !sawChildPrompt {
		t.Fatalf("expected child run to be seeded with the task prompt %q", childPrompt)
	}
}

// A sub-agent's final answer is scrubbed before it becomes the task tool
// result, so a secret a child surfaced never lands in the parent transcript.
func TestRunRedactsSecretsInTaskChildFinalAnswer(t *testing.T) {
	const childPrompt = "find the leaked key"
	secret := "ghp_ABCDEFGHIJKLMNOP1234567890"
	registry := tools.NewRegistry()
	registry.Register(tools.NewTaskTool())
	registry.Register(tools.NewReadFileTool(t.TempDir()))

	provider := &routingProvider{
		parentSubstr: "delegate please",
		parentTurns: [][]zeroruntime.StreamEvent{
			taskCallTurn(childPrompt),
			textTurn("parent done"),
		},
		childTurns: [][]zeroruntime.StreamEvent{
			textTurn("the key is " + secret),
		},
	}

	var captured ToolResult
	_, err := Run(context.Background(), "delegate please", provider, Options{
		Registry:     registry,
		MaxTurns:     5,
		OnToolResult: func(r ToolResult) { captured = r },
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(captured.Output, secret) {
		t.Fatalf("secret leaked into task tool result: %q", captured.Output)
	}
	if !captured.Redacted {
		t.Error("expected Redacted=true when a secret was scrubbed from a sub-agent final answer")
	}
	// The redacted answer must also not reach the parent model.
	for _, request := range provider.requests {
		for _, m := range request.Messages {
			if strings.Contains(m.Content, secret) {
				t.Fatalf("secret leaked into parent model message: %q", m.Content)
			}
		}
	}
}

// (c) The child registry excludes task and ask_user, so the sub-run never
// advertises them and can never recurse via task.
func TestSubAgentRegistryExcludesTaskAndAskUser(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewTaskTool())
	registry.Register(tools.NewAskUserTool())
	registry.Register(tools.NewReadFileTool(t.TempDir()))

	provider := &routingProvider{
		parentSubstr: "please delegate",
		parentTurns: [][]zeroruntime.StreamEvent{
			taskCallTurn("do the sub work"),
			textTurn("parent done"),
		},
		childTurns: [][]zeroruntime.StreamEvent{
			textTurn("child done"),
		},
	}

	if _, err := Run(context.Background(), "please delegate", provider, Options{
		Registry: registry,
		MaxTurns: 5,
		// Ask mode so even the normally hidden tools would be advertised if present.
		PermissionMode: PermissionModeAsk,
	}); err != nil {
		t.Fatal(err)
	}

	if len(provider.childRequests) == 0 {
		t.Fatalf("expected a child run")
	}
	for _, toolDefinition := range provider.childRequests[0].Tools {
		if toolDefinition.Name == "task" {
			t.Fatalf("child run must not advertise the task tool")
		}
		if toolDefinition.Name == "ask_user" {
			t.Fatalf("child run must not advertise the ask_user tool")
		}
	}
	// read_file should still be available to the sub-agent.
	var sawReadFile bool
	for _, toolDefinition := range provider.childRequests[0].Tools {
		if toolDefinition.Name == "read_file" {
			sawReadFile = true
		}
	}
	if !sawReadFile {
		t.Fatalf("expected child run to keep read_file, advertised: %#v", provider.childRequests[0].Tools)
	}
}

// (b) At max depth, a task call returns the limit error and does NOT recurse.
func TestTaskCallAtMaxDepthDoesNotRecurse(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewTaskTool())

	provider := &routingProvider{
		parentSubstr: "delegate at depth",
		parentTurns: [][]zeroruntime.StreamEvent{
			taskCallTurn("should not run"),
			textTurn("parent done"),
		},
		// Any child turn here would indicate an illegal recursion.
		childTurns: [][]zeroruntime.StreamEvent{
			textTurn("ILLEGAL CHILD RUN"),
		},
	}

	var toolResults []ToolResult
	result, err := Run(context.Background(), "delegate at depth", provider, Options{
		Registry:     registry,
		MaxTurns:     5,
		Depth:        maxTaskDepth, // already at the cap
		OnToolResult: func(r ToolResult) { toolResults = append(toolResults, r) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalAnswer != "parent done" {
		t.Fatalf("expected parent final answer, got %q", result.FinalAnswer)
	}
	if provider.childCount != 0 {
		t.Fatalf("expected NO child run at max depth, got %d child turns", provider.childCount)
	}
	if len(toolResults) != 1 {
		t.Fatalf("expected one task tool result, got %d", len(toolResults))
	}
	if toolResults[0].Status != tools.StatusError {
		t.Fatalf("expected error status at max depth, got %s", toolResults[0].Status)
	}
	if !strings.Contains(toolResults[0].Output, "max sub-agent depth") {
		t.Fatalf("expected depth-limit error, got %q", toolResults[0].Output)
	}
}

// The child run must not inherit interactive callbacks: it runs headless even
// when the parent wired up OnAskUser / OnPermissionRequest.
func TestSubAgentRunsWithoutInteractiveCallbacks(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewTaskTool())
	registry.Register(tools.NewReadFileTool(t.TempDir()))

	provider := &routingProvider{
		parentSubstr: "delegate headless",
		parentTurns: [][]zeroruntime.StreamEvent{
			taskCallTurn("sub work in child"),
			textTurn("parent done"),
		},
		childTurns: [][]zeroruntime.StreamEvent{
			textTurn("child done"),
		},
	}

	askUserCalls := 0
	if _, err := Run(context.Background(), "delegate headless", provider, Options{
		Registry: registry,
		MaxTurns: 5,
		OnAskUser: func(context.Context, AskUserRequest) (AskUserResponse, error) {
			askUserCalls++
			return AskUserResponse{}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if askUserCalls != 0 {
		t.Fatalf("sub-agent must not invoke the parent's OnAskUser callback")
	}
}
