package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCode(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		contains string
	}{
		{
			name: "single Go file",
			setup: func() string {
				file := filepath.Join(tmpDir, "main.go")
				os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0644)
				return file
			},
			wantErr:  false,
			contains: "package main",
		},
		{
			name: "directory with Go files",
			setup: func() string {
				dir := filepath.Join(tmpDir, "testdir")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package test\n\nfunc Foo() {}\n"), 0644)
				os.WriteFile(filepath.Join(dir, "file2.go"), []byte("package test\n\nfunc Bar() {}\n"), 0644)
				os.WriteFile(filepath.Join(dir, "file_test.go"), []byte("package test\n\nfunc TestFoo(t *testing.T) {}\n"), 0644)
				return dir
			},
			wantErr:  false,
			contains: "func Foo()",
		},
		{
			name: "non-Go file",
			setup: func() string {
				file := filepath.Join(tmpDir, "readme.md")
				os.WriteFile(file, []byte("# README\n"), 0644)
				return file
			},
			wantErr: true,
		},
		{
			name: "nonexistent path",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.go")
			},
			wantErr: true,
		},
		{
			name: "empty directory",
			setup: func() string {
				dir := filepath.Join(tmpDir, "emptydir")
				os.MkdirAll(dir, 0755)
				return dir
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			code, err := readCode(path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.contains != "" && !strings.Contains(code, tt.contains) {
				t.Errorf("expected code to contain %q", tt.contains)
			}

			// Verify test files are excluded from directories
			if strings.HasSuffix(path, "testdir") {
				if strings.Contains(code, "TestFoo") {
					t.Error("test files should be excluded")
				}
			}
		})
	}
}

func TestCheckWithMissingStandards(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	input := CheckInput{
		Path: testFile,
		// Neither Preset nor Standards provided
	}

	_, _, err := Check(context.Background(), nil, input)
	if err == nil {
		t.Error("expected error when neither preset nor standards provided")
	}

	if !strings.Contains(err.Error(), "either preset or standards must be provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckWithInvalidPreset(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	input := CheckInput{
		Path:   testFile,
		Preset: "nonexistent-preset",
	}

	_, _, err := Check(context.Background(), nil, input)
	if err == nil {
		t.Error("expected error for invalid preset")
	}

	if !strings.Contains(err.Error(), "failed to fetch preset standards") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckWithCustomStandards(t *testing.T) {
	// Skip if no API key (this test would call the actual API)
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping LLM test")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	code := `package main

func hello_world() {
    println("Hello, World!")
}
`
	os.WriteFile(testFile, []byte(code), 0644)

	standards := `# Go Naming Standards
- Use MixedCaps or mixedCaps for names, not snake_case
- All exported functions must have doc comments
`

	input := CheckInput{
		Path:      testFile,
		Standards: standards,
	}

	result, output, err := Check(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected non-empty result")
	}

	if output.Review == "" {
		t.Error("expected non-empty review")
	}

	if output.FilePath != testFile {
		t.Errorf("expected FilePath %s, got %s", testFile, output.FilePath)
	}

	// Review should mention the snake_case issue
	if !strings.Contains(output.Review, "snake") && !strings.Contains(output.Review, "camel") {
		t.Error("expected review to mention naming issue")
	}
}

func TestCheckWithPreset(t *testing.T) {
	// Skip if no API key or in short mode (network + API call)
	if testing.Short() || os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("skipping LLM + network test")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	code := `package main

func main() {
    println("Hello, World!")
}
`
	os.WriteFile(testFile, []byte(code), 0644)

	input := CheckInput{
		Path:   testFile,
		Preset: "google-go",
	}

	result, output, err := Check(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Check with preset failed: %v", err)
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected non-empty result")
	}

	if output.Review == "" {
		t.Error("expected non-empty review")
	}

	// Should contain review header
	if !strings.Contains(output.Review, "# Code Review") {
		t.Error("expected review header")
	}
}

func TestCheckDirectory(t *testing.T) {
	// Skip if no API key
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping LLM test")
	}

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	// Create multiple files
	os.WriteFile(filepath.Join(testDir, "file1.go"), []byte("package main\n\nfunc Foo() {}\n"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("package main\n\nfunc Bar() {}\n"), 0644)

	standards := "Use clear, descriptive names"

	input := CheckInput{
		Path:      testDir,
		Standards: standards,
	}

	result, output, err := Check(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Check directory failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}

	if output.Review == "" {
		t.Error("expected non-empty review")
	}

	// Should mention multiple files
	if !strings.Contains(output.Review, "file1.go") && !strings.Contains(output.Review, "file2.go") {
		t.Log("Review may not explicitly mention file names, checking for general content")
	}
}

func TestCheckErrorResultFormat(t *testing.T) {
	err := context.DeadlineExceeded
	result, output, retErr := checkErrorResult(err)

	if retErr == nil {
		t.Error("expected error to be returned")
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected error result with content")
	}

	if output.Review != "" {
		t.Error("error output should have empty review")
	}
}
