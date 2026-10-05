package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCompatWithUpstreamPython(t *testing.T) {
	root := repoRoot(t)
	t.Chdir(root)
	cases, err := os.ReadFile("testdata/compat/expected/cases.tsv")
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"XDG_CONFIG_HOME": t.TempDir(), "HOME": t.TempDir()}
	for _, line := range strings.Split(strings.TrimRight(string(cases), "\n"), "\n") {
		cols := strings.Split(line, "\t")
		id, stdinPath, args := cols[0], cols[1], cols[2:]
		t.Run(id+" "+strings.Join(args, " "), func(t *testing.T) {
			var stdin []byte
			if stdinPath != "-" {
				if stdin, err = os.ReadFile(stdinPath); err != nil {
					t.Fatal(err)
				}
			}
			var out, errOut bytes.Buffer
			code := Run(args, bytes.NewReader(stdin), &out, &errOut, func(k string) string { return env[k] })
			wantOut, _ := os.ReadFile(filepath.Join("testdata/compat/expected", id+".out"))
			wantExitRaw, _ := os.ReadFile(filepath.Join("testdata/compat/expected", id+".exit"))
			wantExit, _ := strconv.Atoi(strings.TrimSpace(string(wantExitRaw)))
			if code != wantExit {
				t.Errorf("終了コード = %d, want %d（stderr: %s）", code, wantExit, errOut.String())
			}
			if out.String() != string(wantOut) {
				t.Errorf("標準出力が違う\n--- got\n%s\n--- want\n%s", out.String(), wantOut)
			}
		})
	}
}
