package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugRegex(t *testing.T) {
	tests := []struct {
		name  string
		slug  string
		valid bool
	}{
		// Valid slugs
		{"simple lowercase", "hello", true},
		{"with single hyphen", "hello-world", true},
		{"with multiple hyphens", "my-cool-project", true},
		{"with numbers", "project123", true},
		{"numbers and hyphens", "project-123-test", true},
		{"single character", "a", true},
		{"single digit", "1", true},
		{"starts with number", "123abc", true},
		{"ends with number", "abc123", true},
		{"alternating", "a1b2c3", true},

		// Invalid slugs
		{"uppercase letter", "Hello", false},
		{"mixed case", "HelloWorld", false},
		{"leading hyphen", "-hello", false},
		{"trailing hyphen", "hello-", false},
		{"consecutive hyphens", "hello--world", false},
		{"multiple consecutive hyphens", "hello---world", false},
		{"space", "hello world", false},
		{"empty string", "", false},
		{"special char @", "hello@world", false},
		{"special char !", "hello!", false},
		{"underscore", "hello_world", false},
		{"dot", "hello.world", false},
		{"only hyphen", "-", false},
		{"only hyphens", "---", false},
		{"hyphen between hyphens", "a--b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slugRegex.MatchString(tt.slug)
			assert.Equal(t, tt.valid, result, "slug: %q", tt.slug)
		})
	}
}

func TestRegisterCustomValidators(t *testing.T) {
	// This test verifies that RegisterCustomValidators doesn't panic
	// The actual validation is tested through integration tests
	t.Run("does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			RegisterCustomValidators()
		})
	})
}
