package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows では HOME がなく、USERPROFILE がホームを指す。
func TestDefaultPathFallsBackToUserProfile(t *testing.T) {
	get := func(k string) string { return map[string]string{"USERPROFILE": "/users/u"}[k] }
	if p, err := DefaultPath(get); err != nil || p != filepath.Join("/users/u", ".config", "yomiyasu-go", "config.toml") {
		t.Errorf("DefaultPath = %q, %v", p, err)
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultPath(t *testing.T) {
	env := map[string]string{"HOME": "/home/u"}
	get := func(k string) string { return env[k] }
	if p, _ := DefaultPath(get); p != "/home/u/.config/yomiyasu-go/config.toml" {
		t.Errorf("DefaultPath = %q", p)
	}
	env["XDG_CONFIG_HOME"] = "/xdg"
	if p, _ := DefaultPath(get); p != "/xdg/yomiyasu-go/config.toml" {
		t.Errorf("DefaultPath = %q", p)
	}
	env["XDG_CONFIG_HOME"] = "relative"
	if p, _ := DefaultPath(get); p != "/home/u/.config/yomiyasu-go/config.toml" {
		t.Errorf("相対パスの XDG_CONFIG_HOME を使った: %q", p)
	}
}

func TestLoadMissingFileEnablesAll(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "none.toml"))
	if err != nil || len(c.Disabled) != 0 || c.Path != "" {
		t.Errorf("Load = %+v, %v", c, err)
	}
}

func TestLoadDisabled(t *testing.T) {
	p := write(t, "[rules.trailing_colon]\nenabled = false\n\n[rules.emoji_prohibited]\nenabled = true\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Disabled["trailing_colon"] || c.Disabled["emoji_prohibited"] || c.Path != p {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadRejectsMistakes(t *testing.T) {
	cases := map[string]string{
		"[rules.trailng_colon]\nenabled = false\n":      "trailng_colon",
		"[rules.trailing_colon]\nenable = false\n":      "enable",
		"[rules.trailing_colon]\nenabled = \"false\"\n": "enabled",
		"[rule.trailing_colon]\nenabled = false\n":      "rule",
		"[rules.trailing_colon\nenabled = false\n":      "",
	}
	for body, mention := range cases {
		_, err := Load(write(t, body))
		if err == nil {
			t.Errorf("誤りを受け付けた: %q", body)
			continue
		}
		if !strings.Contains(err.Error(), mention) {
			t.Errorf("エラーに %q が含まれない: %v", mention, err)
		}
	}
}

func TestInitWritesAllRulesAndRefusesOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil || len(c.Disabled) != 0 {
		t.Fatalf("雛形を読み直せない: %+v, %v", c, err)
	}
	if err := Init(p); err == nil {
		t.Error("既存の設定を上書きした")
	}
}
