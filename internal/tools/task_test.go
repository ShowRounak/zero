package tools

import (
	"context"
	"strings"
	"testing"
)

func TestParseTaskRequestReadsPromptAndDescription(t *testing.T) {
	request, err := ParseTaskRequest(map[string]any{
		"description":   "find bug",
		"prompt":        "locate the panic in the parser",
		"subagent_type": "explorer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Prompt != "locate the panic in the parser" {
		t.Fatalf("unexpected prompt: %q", request.Prompt)
	}
	if request.Description != "find bug" {
		t.Fatalf("unexpected description: %q", request.Description)
	}
	if request.SubagentType != "explorer" {
		t.Fatalf("unexpected subagent_type: %q", request.SubagentType)
	}
}

func TestParseTaskRequestAcceptsAliases(t *testing.T) {
	// instructions/title are the aliased spellings weaker models emit.
	request, err := ParseTaskRequest(map[string]any{
		"instructions": "do the thing",
		"title":        "the thing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Prompt != "do the thing" {
		t.Fatalf("expected instructions to map to prompt, got %q", request.Prompt)
	}
	if request.Description != "the thing" {
		t.Fatalf("expected title to map to description, got %q", request.Description)
	}
}

func TestParseTaskRequestRequiresPrompt(t *testing.T) {
	if _, err := ParseTaskRequest(map[string]any{"description": "no prompt here"}); err == nil {
		t.Fatalf("expected error when prompt is missing")
	}
	if _, err := ParseTaskRequest(map[string]any{"prompt": "   "}); err == nil {
		t.Fatalf("expected error when prompt is blank")
	}
}

func TestTaskToolRunIsGracefulFallback(t *testing.T) {
	result := NewTaskTool().Run(context.Background(), map[string]any{"prompt": "do work"})
	if result.Status != StatusOK {
		t.Fatalf("expected ok fallback status, got %s", result.Status)
	}
	if !strings.Contains(strings.ToLower(result.Output), "unavailable") {
		t.Fatalf("expected unavailable guidance, got %q", result.Output)
	}
}

func TestTaskToolRunRejectsMissingPrompt(t *testing.T) {
	result := NewTaskTool().Run(context.Background(), map[string]any{"description": "x"})
	if result.Status != StatusError {
		t.Fatalf("expected error status for missing prompt, got %s", result.Status)
	}
}

func TestTaskToolIsReadOnlySafety(t *testing.T) {
	safety := NewTaskTool().Safety()
	if safety.SideEffect != SideEffectRead || safety.Permission != PermissionAllow {
		t.Fatalf("expected read-only/allow safety, got %#v", safety)
	}
}
