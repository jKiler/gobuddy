package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCollectGoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func() string
		wantErr   bool
		wantCount int
		wantFiles []string
	}{
		{
			name: "single Go file",
			setup: func() string {
				file := filepath.Join(tmpDir, "main.go")
				os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0644)
				return file
			},
			wantErr:   false,
			wantCount: 1,
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
			wantErr:   false,
			wantCount: 2,
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
			files, err := collectGoFiles(path)

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

			if len(files) != tt.wantCount {
				t.Errorf("expected %d files, got %d", tt.wantCount, len(files))
			}

			for _, file := range files {
				if strings.HasSuffix(file, "_test.go") {
					t.Error("test files should be excluded")
				}
			}
		})
	}
}

func TestWriteStandardsFile(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	standards := "# Test Standards\nUse clear naming"

	path, err := writeStandardsFile(standards)
	if err != nil {
		t.Fatalf("writeStandardsFile failed: %v", err)
	}

	expectedPath := ".gobuddy/standards.md"
	if path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read standards file: %v", err)
	}

	if string(content) != standards {
		t.Errorf("expected content %q, got %q", standards, string(content))
	}

	info, err := os.Stat(".gobuddy")
	if err != nil {
		t.Fatalf("expected .gobuddy directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .gobuddy to be a directory")
	}
}

func TestReviewWithMissingStandards(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	input := ReviewInput{
		Path: testFile,
	}

	_, _, err := Review(context.Background(), nil, input)
	if err == nil {
		t.Error("expected error when neither preset nor standards provided")
	}

	if !strings.Contains(err.Error(), "either preset or standards must be provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReviewWithInvalidPreset(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	input := ReviewInput{
		Path:   testFile,
		Preset: "nonexistent-preset",
	}

	_, _, err := Review(context.Background(), nil, input)
	if err == nil {
		t.Error("expected error for invalid preset")
	}

	if !strings.Contains(err.Error(), "failed to fetch preset standards") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReviewWithCustomStandards(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

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

	input := ReviewInput{
		Path:      testFile,
		Standards: standards,
	}

	result, output, err := Review(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected non-empty result")
	}

	if output.StandardsPath == "" {
		t.Error("expected non-empty StandardsPath")
	}

	if len(output.FilePaths) != 1 {
		t.Errorf("expected 1 file path, got %d", len(output.FilePaths))
	}

	if output.FilePaths[0] != testFile {
		t.Errorf("expected file path %s, got %s", testFile, output.FilePaths[0])
	}

	if _, err := os.Stat(output.StandardsPath); os.IsNotExist(err) {
		t.Errorf("standards file does not exist at %s", output.StandardsPath)
	}

	resultText := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(resultText, "Please review") {
		t.Error("expected result to contain review instructions")
	}
	if !strings.Contains(resultText, output.StandardsPath) {
		t.Error("expected result to mention standards path")
	}
}

func TestReviewWithPreset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	testFile := filepath.Join(tmpDir, "main.go")
	code := `package main

func main() {
    println("Hello, World!")
}
`
	os.WriteFile(testFile, []byte(code), 0644)

	input := ReviewInput{
		Path:   testFile,
		Preset: "google-go",
	}

	result, output, err := Review(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Review with preset failed: %v", err)
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected non-empty result")
	}

	if output.StandardsPath == "" {
		t.Error("expected non-empty StandardsPath")
	}

	if len(output.FilePaths) != 1 {
		t.Errorf("expected 1 file path, got %d", len(output.FilePaths))
	}

	content, err := os.ReadFile(output.StandardsPath)
	if err != nil {
		t.Fatalf("failed to read standards file: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty standards file")
	}
}

func TestReviewDirectory(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	testDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(testDir, 0755)

	os.WriteFile(filepath.Join(testDir, "file1.go"), []byte("package main\n\nfunc Foo() {}\n"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("package main\n\nfunc Bar() {}\n"), 0644)

	standards := "Use clear, descriptive names"

	input := ReviewInput{
		Path:      testDir,
		Standards: standards,
	}

	result, output, err := Review(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Review directory failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}

	if output.StandardsPath == "" {
		t.Error("expected non-empty StandardsPath")
	}

	if len(output.FilePaths) != 2 {
		t.Errorf("expected 2 file paths, got %d", len(output.FilePaths))
	}

	resultText := result.Content[0].(*mcp.TextContent).Text
	for _, file := range output.FilePaths {
		if !strings.Contains(resultText, filepath.Base(file)) {
			t.Errorf("expected result to mention file %s", file)
		}
	}
}

func TestReviewErrorResultFormat(t *testing.T) {
	err := context.DeadlineExceeded
	result, output, retErr := reviewErrorResult(err)

	if retErr == nil {
		t.Error("expected error to be returned")
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected error result with content")
	}

	if output.StandardsPath != "" {
		t.Error("error output should have empty StandardsPath")
	}

	if len(output.FilePaths) != 0 {
		t.Error("error output should have empty FilePaths")
	}
}
