package skill

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MarkerName は yomiyasu-go が書き出したディレクトリに置く印。印のないディレクトリには書き込まない。
const MarkerName = ".yomiyasu-go"

// DefaultDir は既定の書き出し先 ~/.claude/skills/yomiyasu-go を返す。
func DefaultDir(getenv func(string) string) (string, error) {
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("HOME が設定されていないので、書き出し先を決められない")
	}
	return filepath.Join(home, ".claude", "skills", "yomiyasu-go"), nil
}

// Install は files を dir に書き出し、印を置く。dir が印のあるディレクトリなら消してから書き直す。
// dir がシンボリックリンク、ディレクトリでないもの、印のないディレクトリなら、何も書かずにエラーを返す。
func Install(dir string, files map[string]string, marker string) error {
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%s はシンボリックリンクなので書き出さない。リンクを外してから実行し直す", dir)
	case !fi.IsDir():
		return fmt.Errorf("%s はディレクトリではないので書き出さない", dir)
	default:
		if _, err := os.Stat(filepath.Join(dir, MarkerName)); err != nil {
			return fmt.Errorf("%s には %s がなく、yomiyasu-go が書き出したディレクトリか分からないので書き出さない。中身を確かめて外してから実行し直す", dir, MarkerName)
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, MarkerName), []byte(marker+"\n"), 0o644)
}
