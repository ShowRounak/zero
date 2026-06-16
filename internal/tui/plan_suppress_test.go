package tui

import "testing"

func TestUpdatePlanRowsSuppressedFromChat(t *testing.T) {
	rc := buildRowContext(nil)

	if !rc.skip(transcriptRow{kind: rowToolCall, tool: planToolName, id: "p1"}) {
		t.Fatal("update_plan tool call should be skipped from the chat (it lives in the plan panel / /plan)")
	}
	if !rc.skip(transcriptRow{kind: rowToolResult, tool: planToolName, id: "p1"}) {
		t.Fatal("update_plan's Current Plan result should be skipped from the chat")
	}
	// Ordinary tool rows still render.
	if rc.skip(transcriptRow{kind: rowToolResult, tool: "read_file", id: "r1"}) {
		t.Fatal("a normal tool result must still render in the chat")
	}
}
