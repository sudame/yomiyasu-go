package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(t *testing.T, stdin string, env map[string]string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	getenv := func(k string) string { return env[k] }
	code := Run(args, strings.NewReader(stdin), &out, &errOut, getenv)
	return out.String(), errOut.String(), code
}

func TestVersionShowsUpstreamCommit(t *testing.T) {
	out, _, code := run(t, "", nil, "version")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(out, "986da6ffc89316a90e509d007c1efe1fc59057e6") {
		t.Errorf("version に本家のコミットがない: %q", out)
	}
}
