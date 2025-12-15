package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestFetchURLSource(t *testing.T) {
	// Create test HTTP server
	testContent := "# Test Standards\n\nRule 1: Test rule"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/notfound" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 Not Found"))
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  StandardsSource
		wantErr bool
		wantStr string
	}{
		{
			name: "valid URL",
			source: StandardsSource{
				Type:     "url",
				Location: server.URL,
				Priority: 1,
			},
			wantErr: false,
			wantStr: testContent,
		},
		{
			name: "invalid URL scheme",
			source: StandardsSource{
				Type:     "url",
				Location: "ftp://example.com/standards.md",
				Priority: 1,
			},
			wantErr: true,
		},
		{
			name: "HTTP 404",
			source: StandardsSource{
				Type:     "url",
				Location: server.URL + "/notfound",
				Priority: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := fetchURLSource(ctx, tt.source, cacheDir)

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

			if content != tt.wantStr {
				t.Errorf("got %q, want %q", content, tt.wantStr)
			}

			// Test caching - second fetch should use cache
			content2, err := fetchURLSource(ctx, tt.source, cacheDir)
			if err != nil {
				t.Errorf("cached fetch failed: %v", err)
			}
			if content2 != content {
				t.Error("cached content differs from original")
			}
		})
	}
}

func TestGetPresets(t *testing.T) {
	presets := getPresets()

	expectedPresets := []string{"google-go", "uber-go", "effective-go"}
	for _, name := range expectedPresets {
		sources, ok := presets[name]
		if !ok {
			t.Errorf("expected preset %s not found", name)
			continue
		}
		if len(sources) == 0 {
			t.Errorf("preset %s has no sources", name)
		}
		for _, source := range sources {
			if source.Type != "url" {
				t.Errorf("preset %s has non-url source: %s", name, source.Type)
			}
			if !strings.HasPrefix(source.Location, "https://") {
				t.Errorf("preset %s has invalid URL: %s", name, source.Location)
			}
		}
	}
}

func TestStandardsWithPreset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	input := StandardsInput{
		Preset: "google-go",
	}

	result, output, err := Standards(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Standards with preset failed: %v", err)
	}

	if result == nil || len(result.Content) == 0 {
		t.Error("expected non-empty result")
	}

	if output.Standards == "" {
		t.Error("expected non-empty standards")
	}

	// Should contain content from Google style guide
	if !strings.Contains(output.Standards, "Go") && !strings.Contains(output.Standards, "go") {
		t.Error("expected standards to mention Go")
	}
}

func TestStandardsWithInvalidPreset(t *testing.T) {
	input := StandardsInput{
		Preset: "nonexistent-preset",
	}

	_, _, err := Standards(context.Background(), nil, input)
	if err == nil {
		t.Error("expected error for invalid preset")
	}

	if !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("expected 'unknown preset' error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Available presets") {
		t.Error("error should list available presets")
	}
}

func TestStandardsPresetPriority(t *testing.T) {
	input := StandardsInput{
		Preset: "google-go",
	}

	// Preset should override custom sources
	if len(input.Sources) > 0 {
		t.Error("sources should be empty when using preset")
	}

	_, _, err := Standards(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Standards failed: %v", err)
	}

	// After Standards call, sources should be populated from preset
	// (this is internal behavior, just verifying no errors)
}
