package tools

import (
	"context"
	"fmt"
	"strings"
)

// taskNonInteractiveMessage is returned by the tool's own Run() fallback when
// nothing intercepts the call (no provider available to spawn a child run, or
// the call reached this tool directly through the registry). It mirrors
// ask_user's graceful-degradation pattern so the model gets actionable guidance
// instead of a hard failure.
const taskNonInteractiveMessage = "Sub-agents are unavailable in this context. " +
	"Continue the task yourself using the tools available to you."

// TaskRequest is the parsed, validated form of a task tool call. It is shared by
// the tool's Run() fallback and the agent loop's interception path so both
// validate identically.
type TaskRequest struct {
	// Description is a short human-readable label for the sub-task (UI/logging).
	Description string
	// Prompt is the instruction handed to the sub-agent.
	Prompt string
	// SubagentType optionally names a specialized sub-agent profile. Unused by
	// the current synchronous runner; carried through for forward compatibility.
	SubagentType string
}

type taskTool struct {
	baseTool
}

// NewTaskTool builds the task tool, which spawns an isolated sub-agent run to
// handle a delegated sub-task. It is marked read-only for SAFETY-advertising
// purposes (it never mutates the workspace itself; the child run enforces its
// own permissions/sandbox). The agent loop intercepts task calls and runs a
// child agent synchronously; this tool's Run() is the fallback used when nothing
// intercepts the call (e.g. no provider is wired up).
func NewTaskTool() *taskTool {
	return &taskTool{
		baseTool: baseTool{
			name: "task",
			description: "Delegate a self-contained sub-task to an isolated sub-agent. " +
				"The sub-agent runs in its own context with the same tools (minus task and ask_user) " +
				"and returns a final summary of what it did. " +
				"Use for focused, parallelizable, or context-heavy work you want handled separately. " +
				"Provide a short `description` and a detailed `prompt` describing the sub-task.",
			parameters: Schema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"description": {
						Type:        "string",
						Description: "A short (3-5 word) label for the sub-task.",
					},
					"prompt": {
						Type:        "string",
						Description: "The full, self-contained instructions for the sub-agent to carry out.",
					},
					"subagent_type": {
						Type:        "string",
						Description: "Optional name of a specialized sub-agent profile to use.",
					},
				},
				Required:             []string{"prompt"},
				AdditionalProperties: false,
			},
			safety: readOnlySafety("Delegates a sub-task to an isolated sub-agent; the child run enforces its own permissions."),
		},
	}
}

// Run is the fallback path: it is only reached when nothing intercepted the call
// (no provider to spawn a child run). It validates the arguments so a malformed
// call still gets useful feedback, then tells the model that sub-agents are not
// available here. It never spawns anything itself.
func (tool *taskTool) Run(_ context.Context, args map[string]any) Result {
	if _, err := ParseTaskRequest(args); err != nil {
		return errorResult("Error: Invalid arguments for task: " + err.Error())
	}
	return okResult(taskNonInteractiveMessage)
}

// TaskNonInteractiveMessage exposes the shared graceful-degradation message so
// the agent loop and the tool fallback stay in lock-step.
func TaskNonInteractiveMessage() string {
	return taskNonInteractiveMessage
}

// ParseTaskRequest extracts the sub-task from raw tool arguments, tolerating the
// key spellings different models emit. The prompt is accepted under any of
// prompt/instructions/task/input; the description under description/title/label/
// name. The prompt is required; the description and subagent_type are optional.
func ParseTaskRequest(args map[string]any) (TaskRequest, error) {
	prompt, err := aliasedStringArg(args, []string{"prompt", "instructions", "task", "input"}, "", true, false)
	if err != nil {
		// Normalize the error to the canonical "prompt" key regardless of alias.
		return TaskRequest{}, fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return TaskRequest{}, fmt.Errorf("prompt is required")
	}

	description, err := aliasedStringArg(args, []string{"description", "title", "label", "name"}, "", false, true)
	if err != nil {
		return TaskRequest{}, err
	}

	subagentType, err := aliasedStringArg(args, []string{"subagent_type", "subagentType", "agent"}, "", false, true)
	if err != nil {
		return TaskRequest{}, err
	}

	return TaskRequest{
		Description:  strings.TrimSpace(description),
		Prompt:       prompt,
		SubagentType: strings.TrimSpace(subagentType),
	}, nil
}
