package main

import "testing"

func TestRemoveGitmojiPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes gitmoji prefix",
			input:    "✨ Add new feature",
			expected: "Add new feature",
		},
		{
			name:     "does not remove non-gitmoji prefix",
			input:    "Add new feature",
			expected: "Add new feature",
		},
		{
			name:     "removes gitmoji prefix with multiple spaces",
			input:    "✨   Add new feature",
			expected: "Add new feature",
		},
		{
			name:     "does not remove gitmoji prefix if it's part of a word",
			input:    "✨Add new feature",
			expected: "✨Add new feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeGitmojiPrefix(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveGitmojiPrefix(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
