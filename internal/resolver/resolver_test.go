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
		{
			name:     "Four-part version (scrivo/highlight.php style)",
			input:    "v9.18.1.10",
			expected: "9.18.1",
		},
		{
			name:     "Four-part version without v prefix",
			input:    "9.18.1.10",
			expected: "9.18.1",
		},
		{
			name:     "Four-part version with pre-release on fourth segment is left alone",
			input:    "1.2.3.0-beta",
			expected: "1.2.3.0-beta",
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

// TestFindMatchingVersion_FourPartVersions verifies that four-part Composer
// versions like 9.18.1.10 are matched correctly against constraints like ^9.18.
// This is the root cause of issue #13 (scrivo/highlight.php).
func TestFindMatchingVersion_FourPartVersions(t *testing.T) {
	r := NewResolver(packagist.NewClient())

	info := &packagist.PackageInfo{
		Name: "scrivo/highlight.php",
		Versions: map[string]*packagist.VersionInfo{
			"v9.18.1.10": {Name: "scrivo/highlight.php", Version: "v9.18.1.10"},
			"v9.18.1.4":  {Name: "scrivo/highlight.php", Version: "v9.18.1.4"},
			"v9.12.0.0":  {Name: "scrivo/highlight.php", Version: "v9.12.0.0"},
			"v9.17.1.0":  {Name: "scrivo/highlight.php", Version: "v9.17.1.0"},
		},
	}

	tests := []struct {
		constraint  string
		wantVersion string
	}{
		{"^9.18", "v9.18.1.10"}, // should pick the highest matching
		{"^9.12", "v9.18.1.10"}, // ^9.12 allows anything >=9.12 <10
		{"~9.18.1", "v9.18.1.10"},
		{">=9.17", "v9.18.1.10"},
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			got, err := r.findMatchingVersion(info, tt.constraint)
			if err != nil {
				t.Fatalf("findMatchingVersion(%q) returned error: %v", tt.constraint, err)
			}
			if got != tt.wantVersion {
				t.Errorf("findMatchingVersion(%q) = %q, want %q", tt.constraint, got, tt.wantVersion)
			}
		})
	}
}
