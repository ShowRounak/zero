package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir string, name string, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func TestLoadParsesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "confirmation-policy", "---\nname: confirmation-policy\ndescription: When to ask the user before risky actions.\n---\n\n# Confirmation Policy\n\nAsk first.\n")

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded))
	}
	skill := loaded[0]
	if skill.Name != "confirmation-policy" {
		t.Fatalf("Name = %q, want confirmation-policy", skill.Name)
	}
	if skill.Description != "When to ask the user before risky actions." {
		t.Fatalf("Description = %q", skill.Description)
	}
	wantContent := "# Confirmation Policy\n\nAsk first."
	if skill.Content != wantContent {
		t.Fatalf("Content = %q, want %q", skill.Content, wantContent)
	}
	if skill.Path == "" {
		t.Fatalf("Path is empty")
	}
}

func TestLoadDerivesNameFromDirectoryWithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "no-frontmatter", "# Just markdown\n\nNo frontmatter here.\n")

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(loaded))
	}
	skill := loaded[0]
	if skill.Name != "no-frontmatter" {
		t.Fatalf("Name = %q, want no-frontmatter", skill.Name)
	}
	if skill.Description != "" {
		t.Fatalf("Description = %q, want empty", skill.Description)
	}
	if skill.Content != "# Just markdown\n\nNo frontmatter here." {
		t.Fatalf("Content = %q", skill.Content)
	}
}

func TestLoadSkipsMalformedAndContinues(t *testing.T) {
	dir := t.TempDir()
	// A directory whose SKILL.md is a directory itself (unreadable as a file) is skipped.
	badDir := filepath.Join(dir, "broken")
	if err := os.MkdirAll(filepath.Join(badDir, "SKILL.md"), 0o755); err != nil {
		t.Fatalf("mkdir broken: %v", err)
	}
	writeSkill(t, dir, "good", "---\nname: good\ndescription: works\n---\nbody\n")

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 skill (malformed skipped), got %d", len(loaded))
	}
	if loaded[0].Name != "good" {
		t.Fatalf("Name = %q, want good", loaded[0].Name)
	}
}

func TestLoadIgnoresDirectoriesWithoutSkillFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(loaded))
	}
}

func TestLoadMissingDirYieldsEmpty(t *testing.T) {
	loaded, err := Load(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("Load on missing dir returned error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 skills for missing dir, got %d", len(loaded))
	}
}

func TestLoadSortsByName(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "zeta", "body")
	writeSkill(t, dir, "alpha", "body")

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(loaded))
	}
	if loaded[0].Name != "alpha" || loaded[1].Name != "zeta" {
		t.Fatalf("skills not sorted: %q, %q", loaded[0].Name, loaded[1].Name)
	}
}

func TestGetByName(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "one", "---\nname: one\ndescription: first\n---\ncontent one\n")

	skill, ok := Get(dir, "one")
	if !ok {
		t.Fatalf("Get(one) not found")
	}
	if skill.Content != "content one" {
		t.Fatalf("Content = %q", skill.Content)
	}

	if _, ok := Get(dir, "missing"); ok {
		t.Fatalf("Get(missing) should not be found")
	}
}

func TestListReturnsNamesAndDescriptions(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "b", "---\nname: b\ndescription: bee\n---\nbody")
	writeSkill(t, dir, "a", "---\nname: a\ndescription: ay\n---\nbody")

	listed, err := List(dir)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2, got %d", len(listed))
	}
	if listed[0].Name != "a" || listed[0].Description != "ay" {
		t.Fatalf("unexpected first skill: %+v", listed[0])
	}
}

func TestDefaultDirHonorsEnvOverride(t *testing.T) {
	got := DefaultDir(map[string]string{"ZERO_SKILLS_DIR": "/custom/skills"})
	if got != "/custom/skills" {
		t.Fatalf("DefaultDir override = %q, want /custom/skills", got)
	}
}

func TestDefaultDirHonorsXDGDataHome(t *testing.T) {
	got := DefaultDir(map[string]string{"XDG_DATA_HOME": "/xdg/data"})
	want := filepath.Join("/xdg/data", "zero", "skills")
	if got != want {
		t.Fatalf("DefaultDir = %q, want %q", got, want)
	}
}

func TestDefaultDirFallsBackToHome(t *testing.T) {
	got := DefaultDir(map[string]string{"HOME": "/home/zero"})
	want := filepath.Join("/home/zero", ".local", "share", "zero", "skills")
	if got != want {
		t.Fatalf("DefaultDir = %q, want %q", got, want)
	}
}
