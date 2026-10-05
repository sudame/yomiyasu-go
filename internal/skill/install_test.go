package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var sample = map[string]string{"SKILL.md": "s", "references/a.md": "a", "LICENSE": "l"}

func TestInstallIntoNewDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := Install(dir, sample, "v"); err != nil {
		t.Fatal(err)
	}
	for rel, want := range sample {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil || string(b) != want {
			t.Errorf("%s = %q, %v", rel, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, MarkerName)); err != nil {
		t.Error("印がない")
	}
}

func TestReinstallRemovesStaleFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := Install(dir, sample, "v"); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, map[string]string{"SKILL.md": "s2"}, "v"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "references/a.md")); !os.IsNotExist(err) {
		t.Error("古いファイルが残った")
	}
}

func TestInstallRefusesForeignDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Install(dir, sample, "v")
	if err == nil || !strings.Contains(err.Error(), MarkerName) {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(b) != "mine" {
		t.Error("利用者のファイルを上書きした")
	}
}

func TestInstallRefusesSymlink(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "real")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, MarkerName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "yomiyasu-go")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	err := Install(link, sample, "v")
	if err == nil || !strings.Contains(err.Error(), "シンボリックリンク") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("リンク先に書き込んだ")
	}
}

func TestInstallRefusesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(p, sample, "v"); err == nil {
		t.Error("ファイルを上書きした")
	}
}

func TestDefaultDir(t *testing.T) {
	get := func(k string) string { return map[string]string{"HOME": "/home/u"}[k] }
	if d, _ := DefaultDir(get); d != "/home/u/.claude/skills/yomiyasu-go" {
		t.Errorf("DefaultDir = %q", d)
	}
}

func TestDefaultDirFallsBackToUserProfile(t *testing.T) {
	get := func(k string) string { return map[string]string{"USERPROFILE": "/users/u"}[k] }
	if d, err := DefaultDir(get); err != nil || d != filepath.Join("/users/u", ".claude", "skills", "yomiyasu-go") {
		t.Errorf("DefaultDir = %q, %v", d, err)
	}
}

// 末尾に / を付けると Lstat がリンク先をたどるので、リンクとして扱えているかを確かめる。
func TestInstallRefusesSymlinkWithTrailingSlash(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "real")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, MarkerName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "yomiyasu-go")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	err := Install(link+"/", sample, "v")
	if err == nil || !strings.Contains(err.Error(), "シンボリックリンク") {
		t.Fatalf("err = %v", err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("リンクが消えた")
	}
}
