package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModule writes a Go module out of the given files into a temp dir.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGocheckClean(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/clean\n\ngo 1.25\n",
		"add.go": `// Package clean is a passing gocheck fixture.
package clean

// Add returns a + b.
func Add(a, b int) int { return a + b }
`,
		"add_test.go": `package clean

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("Add(1, 2) != 3")
	}
}
`,
	})

	result, out, err := Gocheck(context.Background(), nil, GocheckInput{WorkingDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected IsError result: %v", result.Content)
	}
	if !out.FormattedOK || !out.VetOK || !out.TestsOK {
		t.Fatalf("clean module should pass all checks, got %+v", out)
	}
	if len(out.UnformattedFiles) != 0 || len(out.VetIssues) != 0 || len(out.FailingTests) != 0 {
		t.Fatalf("clean module should report no findings, got %+v", out)
	}
}

func TestGocheckFailures(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/dirty\n\ngo 1.25\n",
		// Extra indentation keeps this file deliberately unformatted.
		"sub.go": `// Package dirty is a failing gocheck fixture.
package dirty

// Sub returns a - b.
func Sub(a, b int) int {
		return a - b
}
`,
		"sub_test.go": `package dirty

import "testing"

func TestSubBroken(t *testing.T) {
	if Sub(2, 1) != 5 {
		t.Fatal("deliberate fixture failure")
	}
}
`,
	})

	result, out, err := Gocheck(context.Background(), nil, GocheckInput{WorkingDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("check failures are report data, not IsError: %v", result.Content)
	}

	if out.FormattedOK || len(out.UnformattedFiles) != 1 || !strings.Contains(out.UnformattedFiles[0], "sub.go") {
		t.Errorf("expected sub.go to be reported unformatted, got %+v", out)
	}
	if out.TestsOK || len(out.FailingTests) != 1 {
		t.Fatalf("expected exactly one failing test, got %+v", out)
	}
	failing := out.FailingTests[0]
	if failing.Name != "TestSubBroken" || failing.Package != "example.com/dirty" {
		t.Errorf("unexpected failing test identity: %+v", failing)
	}
	if !strings.Contains(failing.OutputTail, "deliberate fixture failure") {
		t.Errorf("output tail should carry the failure message, got %q", failing.OutputTail)
	}
	if len(failing.OutputTail) > outputTailLimit {
		t.Errorf("output tail exceeds limit: %d bytes", len(failing.OutputTail))
	}
}

func TestGocheckSkipTests(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module example.com/skip\n\ngo 1.25\n",
		"skip.go": `// Package skip is a gocheck fixture without tests run.
package skip
`,
		"skip_test.go": `package skip

import "testing"

func TestAlwaysFails(t *testing.T) { t.Fatal("should not run") }
`,
	})

	skip := false
	_, out, err := Gocheck(context.Background(), nil, GocheckInput{WorkingDir: dir, IncludeTests: &skip})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.TestsOK || len(out.FailingTests) != 0 {
		t.Fatalf("tests were skipped, report should stay clean, got %+v", out)
	}
}

func TestGocheckBadWorkingDir(t *testing.T) {
	result, _, err := Gocheck(context.Background(), nil, GocheckInput{WorkingDir: "/nonexistent/gobuddy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("expected IsError result for missing working_dir")
	}
}
