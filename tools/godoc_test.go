package tools

import (
	"context"
	"testing"
)

func TestGodoc(t *testing.T) {
	tests := []struct {
		name    string
		input   GodocInput
		wantErr bool
	}{
		{
			name:    "standard package",
			input:   GodocInput{Package: "fmt"},
			wantErr: false,
		},
		{
			name:    "package with symbol",
			input:   GodocInput{Package: "fmt", Symbol: "Printf"},
			wantErr: false,
		},
		{
			name:    "nonexistent package",
			input:   GodocInput{Package: "nonexistent/package/xyz"},
			wantErr: true,
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
				t.Error("unexpected IsError result")
				return
			}

			if result == nil || len(result.Content) == 0 {
				t.Error("expected non-empty result")
			}

			if output.Documentation == "" {
				t.Error("expected non-empty documentation")
			}
		})
	}
}
