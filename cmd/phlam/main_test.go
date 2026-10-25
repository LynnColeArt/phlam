package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	path := "../../examples/counter/counter.phlam"
	var out, stderr bytes.Buffer
	if code := run([]string{"check", path}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "bootstrap syntax checks only") {
		t.Fatalf("check: %d %s %s", code, &out, &stderr)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"inspect", path}, &out, &stderr); code != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("inspect: %d %s", code, &stderr)
	}
	invalid := filepath.Join(t.TempDir(), "bad.phlam")
	if err := os.WriteFile(invalid, []byte("class Undocumented {}"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := run([]string{"check", invalid}, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "P101") || !strings.Contains(stderr.String(), "bad.phlam:1:1") {
		t.Fatalf("diagnostic: %d %s", code, &stderr)
	}
	for _, args := range [][]string{nil, {"serve", path}, {"check"}, {"inspect", path, "extra"}} {
		if run(args, &out, &stderr) != 2 {
			t.Fatal("bad usage accepted", args)
		}
	}
	if run([]string{"check", "/nonexistent/phlam-source"}, &out, &stderr) != 1 {
		t.Fatal("missing file accepted")
	}
}
