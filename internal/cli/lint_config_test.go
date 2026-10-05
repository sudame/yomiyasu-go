package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintSkipsDisabledRules(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.trailing_colon]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"XDG_CONFIG_HOME": xdg}
	out, _, code := run(t, "項目：\n", env, "lint", "--json")
	if code != 0 || strings.Contains(out, "trailing_colon") || !strings.Contains(out, `"score": 100`) {
		t.Errorf("code=%d out=%s", code, out)
	}
}

func TestLintConfigErrorExits2(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.nope]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := run(t, "本文です。\n", map[string]string{"XDG_CONFIG_HOME": xdg}, "lint")
	if code != 2 || !strings.Contains(errOut, "設定ファイルのエラー") || !strings.Contains(errOut, "nope") {
		t.Errorf("code=%d stderr=%s", code, errOut)
	}
}

func TestLintTextReportMarksDisabledMetrics(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.excess_bold]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, _ := run(t, "本文です。\n", map[string]string{"XDG_CONFIG_HOME": xdg}, "lint")
	if !strings.Contains(out, "(検査は無効)") || !strings.Contains(out, "無効にした規則: excess_bold") {
		t.Errorf("out=%s", out)
	}
}

func TestConfigInit(t *testing.T) {
	xdg := t.TempDir()
	env := map[string]string{"XDG_CONFIG_HOME": xdg}
	if _, errOut, code := run(t, "", env, "config", "init"); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(xdg, "yomiyasu-go", "config.toml")); err != nil {
		t.Fatal(err)
	}
	if _, _, code := run(t, "", env, "config", "init"); code == 0 {
		t.Error("既存の設定を上書きした")
	}
}
