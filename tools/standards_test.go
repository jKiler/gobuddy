package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchCodingStandards(t *testing.T) {
	// Create temp dir with test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "STANDARDS.md")
	if err := os.WriteFile(testFile, []byte("# Test Standards\n\nRule 1"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   StandardsInput
		wantErr bool
	}{
		{
			name: "local source",
			input: StandardsInput{
				Sources: []StandardsSource{
					{Type: "local", Location: tmpDir, Files: []string{"STANDARDS.md"}, Priority: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "local source with include_source",
			input: StandardsInput{
				Sources: []StandardsSource{
					{Type: "local", Location: tmpDir, Files: []string{"STANDARDS.md"}, Priority: 1},
				},
				IncludeSource: true,
			},
			wantErr: false,
		},
		{
			name: "nonexistent local file",
			input: StandardsInput{
				Sources: []StandardsSource{
					{Type: "local", Location: tmpDir, Files: []string{"NONEXISTENT.md"}, Priority: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown source type",
			input: StandardsInput{
				Sources: []StandardsSource{
					{Type: "unknown", Location: "foo", Priority: 1},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, output, err := Standards(context.Background(), nil, tt.input)

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

			if result == nil || len(result.Content) == 0 {
				t.Error("expected non-empty result")
			}

			if output.Standards == "" {
				t.Error("expected non-empty standards")
			}
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	// Nonexistent dir should need refresh
	if !needsRefresh("/nonexistent/path") {
		t.Error("nonexistent path should need refresh")
	}

	// Dir without timestamp should need refresh
	tmpDir := t.TempDir()
	if !needsRefresh(tmpDir) {
		t.Error("dir without timestamp should need refresh")
	}

	// Dir with fresh timestamp should not need refresh
	touchCacheTimestamp(tmpDir)
	if needsRefresh(tmpDir) {
		t.Error("fresh cache should not need refresh")
	}
}
