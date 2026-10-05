// Package config は ~/.config/yomiyasu-go/config.toml を読み書きする。
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

// Config は読み込んだ設定。
type Config struct {
	Path     string // 読んだ設定ファイル。ファイルがなければ空
	Disabled map[string]bool
}

// DefaultPath は設定ファイルの場所を返す。XDG_CONFIG_HOME が絶対パスならその下を使う。
func DefaultPath(getenv func(string) string) (string, error) {
	if x := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "yomiyasu-go", "config.toml"), nil
	}
	home, err := HomeDir(getenv)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "yomiyasu-go", "config.toml"), nil
}

// HomeDir はホームディレクトリを返す。HOME がなければ、Windows で使われる USERPROFILE を見る。
func HomeDir(getenv func(string) string) (string, error) {
	for _, k := range []string{"HOME", "USERPROFILE"} {
		if v := getenv(k); v != "" {
			return v, nil
		}
	}
	return "", errors.New("HOME も USERPROFILE も設定されていないので、ホームディレクトリを決められない")
}

type fileFormat struct {
	Rules map[string]struct {
		Enabled *bool `toml:"enabled"`
	} `toml:"rules"`
}

// Load は設定ファイルを読む。ファイルがなければ、すべての規則を有効とみなす。
func Load(path string) (Config, error) {
	c := Config{Disabled: map[string]bool{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	var f fileFormat
	md, err := toml.Decode(string(b), &f)
	if err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if und := md.Undecoded(); len(und) > 0 {
		keys := make([]string, len(und))
		for i, k := range und {
			keys[i] = k.String()
		}
		return c, fmt.Errorf("%s: 知らない設定項目がある: %s", path, strings.Join(keys, ", "))
	}
	for id, r := range f.Rules {
		if !rules.Known(id) {
			return c, fmt.Errorf("%s: 知らない規則 ID %q がある（使える ID: %s）", path, id, strings.Join(rules.All, ", "))
		}
		if r.Enabled != nil && !*r.Enabled {
			c.Disabled[id] = true
		}
	}
	c.Path = path
	return c, nil
}

// Init は、すべての規則を有効にした設定の雛形を書き出す。すでにファイルがあれば上書きせずにエラーを返す。
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# yomiyasu-go の設定。enabled = false にした規則は、lint で検出しなくなり、SKILL.md の書き直しの指示からも外れる。\n")
	b.WriteString("# 設定を変えたら、yomiyasu-go skill install を実行し直して SKILL.md に反映する。\n")
	for _, id := range rules.All {
		fmt.Fprintf(&b, "\n[rules.%s]\nenabled = true\n", id)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(b.String())
	return errors.Join(werr, f.Close())
}
