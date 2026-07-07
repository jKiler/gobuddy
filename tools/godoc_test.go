package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureModule writes a self-contained Go module to a temp directory so
// godoc tests resolve documentation offline.
func fixtureModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"go.mod": "module example.com/fixture\n\ngo 1.25\n",
		"greet.go": `// Package fixture is a test fixture for godoc.
package fixture

// Greeting is the canonical fixture greeting.
const Greeting = "hello"

// Greet returns a greeting for name.
func Greet(name string) string {
	return Greeting + " " + name
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGodoc(t *testing.T) {
	dir := fixtureModule(t)

	tests := []struct {
		name       string
		input      GodocInput
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "package in working_dir module",
			input:      GodocInput{Package: "example.com/fixture", WorkingDir: dir},
			wantOutput: "Package fixture is a test fixture",
		},
		{
			name:       "symbol in working_dir module",
			input:      GodocInput{Package: "example.com/fixture", Symbol: "Greet", WorkingDir: dir},
			wantOutput: "Greet returns a greeting",
		},
		{
			name:       "mode all",
			input:      GodocInput{Package: "example.com/fixture", Mode: "all", WorkingDir: dir},
			wantOutput: "Greeting is the canonical fixture greeting",
		},
		{
			name:       "mode src",
			input:      GodocInput{Package: "example.com/fixture", Symbol: "Greet", Mode: "src", WorkingDir: dir},
			wantOutput: "return Greeting",
		},
		{
			name:    "nonexistent package",
			input:   GodocInput{Package: "example.com/fixture/nope", WorkingDir: dir},
			wantErr: true,
		},
		{
			name:    "nonexistent symbol",
			input:   GodocInput{Package: "example.com/fixture", Symbol: "Missing", WorkingDir: dir},
			wantErr: true,
		},
		{
			name:    "unknown mode",
			input:   GodocInput{Package: "example.com/fixture", Mode: "bogus", WorkingDir: dir},
			wantErr: true,
		},
		{
			name:    "working_dir does not exist",
			input:   GodocInput{Package: "fmt", WorkingDir: filepath.Join(dir, "missing")},
			wantErr: true,
		},
		{
			name:       "default working_dir resolves stdlib",
			input:      GodocInput{Package: "fmt", Symbol: "Printf"},
			wantOutput: "Printf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, output, err := Godoc(context.Background(), nil, tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected IsError result")
				}
				return
			}

			if result != nil && result.IsError {
				t.Errorf("unexpected IsError result: %v", result.Content)
				return
			}

			if result == nil || len(result.Content) == 0 {
				t.Error("expected non-empty result")
				return
			}

			if !strings.Contains(output.Documentation, tt.wantOutput) {
				t.Errorf("documentation %q does not contain %q", output.Documentation, tt.wantOutput)
			}
		})
	}
}
