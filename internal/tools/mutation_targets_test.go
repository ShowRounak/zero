package tools

import (
	"reflect"
	"testing"
)

func TestMutationTargets(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		tool string
		args map[string]any
		want []string
	}{
		{"write_file", map[string]any{"path": "a/b.txt", "content": "x"}, []string{"a/b.txt"}},
		{"edit_file", map[string]any{"path": "c.txt", "old_string": "x", "new_string": "y"}, []string{"c.txt"}},
		{"apply_patch", map[string]any{"patch": "--- a/d.txt\n+++ b/d.txt\n@@ -1 +1 @@\n-x\n+y\n"}, []string{"d.txt"}},
		{"bash", map[string]any{"command": "echo hi"}, nil},
		{"read_file", map[string]any{"path": "e.txt"}, nil},
		{"grep", map[string]any{"pattern": "x"}, nil},
	}
	for _, tc := range cases {
		got := MutationTargets(root, tc.tool, tc.args)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.tool, got, tc.want)
		}
	}
}

func TestMutationTargetsResolvesAliasKeys(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name string
		tool string
		args map[string]any
		want []string
	}{
		{"write_file file alias", "write_file", map[string]any{"file": "x.go", "content": "x"}, []string{"x.go"}},
		{"write_file file_path alias", "write_file", map[string]any{"file_path": "y.go", "content": "x"}, []string{"y.go"}},
		{"write_file filename alias", "write_file", map[string]any{"filename": "z.go", "content": "x"}, []string{"z.go"}},
		{"edit_file file alias", "edit_file", map[string]any{"file": "e.go", "old_string": "a", "new_string": "b"}, []string{"e.go"}},
		{"apply_patch diff alias", "apply_patch", map[string]any{"diff": "--- a/d.txt\n+++ b/d.txt\n@@ -1 +1 @@\n-x\n+y\n"}, []string{"d.txt"}},
	}
	for _, tc := range cases {
		got := MutationTargets(root, tc.tool, tc.args)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMutationTargetsRejectsEscapingPaths(t *testing.T) {
	root := t.TempDir()
	if got := MutationTargets(root, "write_file", map[string]any{"path": "../escape.txt", "content": "x"}); len(got) != 0 {
		t.Errorf("expected no targets for escaping path, got %v", got)
	}
}

func TestStripPatchPrefixStripsOnlyOne(t *testing.T) {
	root := t.TempDir()
	// A workspace file under a directory literally named "b".
	got := MutationTargets(root, "apply_patch", map[string]any{
		"patch": "--- a/b/foo.txt\n+++ b/b/foo.txt\n@@ -1 +1 @@\n-x\n+y\n",
	})
	if len(got) != 1 || got[0] != "b/foo.txt" {
		t.Fatalf("expected [b/foo.txt], got %v", got)
	}
}
