package resolver

import (
	"testing"

	"github.com/aras/presto/internal/packagist"
)

func TestNormalizeConstraint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple caret constraint",
			input:    "^1.0",
			expected: "^1.0",
		},
		{
			name:     "OR constraint with spaces",
			input:    "^1.9 || ^2.4",
			expected: "^1.9||^2.4",
		},
		{
			name:     "Tilde constraint",
			input:    "~1.2.3",
			expected: "~1.2.3",
		},
		{
			name:     "Complex OR with multiple spaces",
			input:    "^1.0  ||  ^2.0  ||  ^3.0",
			expected: "^1.0||^2.0||^3.0",
		},
	}

	r := NewResolver(packagist.NewClient())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.normalizeConstraint(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeConstraint(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Version with v prefix",
			input:    "v1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "Dev version",
			input:    "1.0.0-dev",
			expected: "1.0.0-alpha",
		},
		{
			name:     "Normal version",
			input:    "2.3.4",
			expected: "2.3.4",
		},
	}

	r := NewResolver(packagist.NewClient())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPlatformPackage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "PHP package",
			input:    "php",
			expected: true,
		},
		{
			name:     "Extension",
			input:    "ext-json",
			expected: true,
		},
		{
			name:     "Library",
			input:    "lib-curl",
			expected: true,
		},
		{
			name:     "Regular package",
			input:    "symfony/console",
			expected: false,
		},
	}

	r := NewResolver(packagist.NewClient())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.isPlatformPackage(tt.input)
			if result != tt.expected {
				t.Errorf("isPlatformPackage(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	// This would require mocking the Packagist client
	// For now, we'll skip this test
	t.Skip("Requires Packagist client mocking")
}
