package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestZerolineSnapshotToolFlag(t *testing.T) {
	cases := map[string][]string{
		"diff": {"edit_file", "ShowVersion"},
		"read": {"read_file", "Loop"},
		"bash": {"bash", "exit 0"},
		"grep": {"grep", "matches"},
	}
	for kind, wants := range cases {
		var out, errb bytes.Buffer
		code := runZeroline([]string{"--snapshot", "--page", "chat", "--tool", kind, "--width", "100", "--height", "20"}, &out, &errb, appDeps{})
		if code != 0 {
			t.Fatalf("--tool %s: exit %d, stderr=%s", kind, code, errb.String())
		}
		for _, w := range wants {
			if !strings.Contains(out.String(), w) {
				t.Errorf("--tool %s snapshot missing %q", kind, w)
			}
		}
	}
}

func TestZerolineSnapshotModes(t *testing.T) {
	for _, args := range [][]string{
		{"--snapshot", "--page", "home"},
		{"--snapshot", "--page", "chat"},
		{"--snapshot", "--page", "chat", "--json"},
		{"--snapshot", "--page", "chat", "--sessions"},
		{"--snapshot", "--page", "chat", "--perm"},
		{"--snapshot", "--page", "chat", "--stream", "--frame", "3"},
	} {
		var out, errb bytes.Buffer
		full := append(args, "--width", "100", "--height", "24")
		if code := runZeroline(full, &out, &errb, appDeps{}); code != 0 {
			t.Fatalf("%v: exit %d, stderr=%s", args, code, errb.String())
		}
		if strings.TrimSpace(out.String()) == "" {
			t.Errorf("%v produced empty output", args)
		}
	}
}
