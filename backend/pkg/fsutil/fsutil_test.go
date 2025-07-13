package fsutil

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "spaces to hyphens",
			input:    "my file name.txt",
			expected: "my-file-name.txt",
		},
		{
			name:     "forward slashes to hyphens",
			input:    "path/to/file.txt",
			expected: "path-to-file.txt",
		},
		{
			name:     "backslashes to hyphens",
			input:    "path\\to\\file.txt",
			expected: "path-to-file.txt",
		},
		{
			name:     "mixed case to lowercase",
			input:    "MyFile.TXT",
			expected: "myfile.txt",
		},
		{
			name:     "special filename",
			input:    "_NSAKEY.v2.dive.rule",
			expected: "_nsakey.v2.dive.rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractBaseNameWithoutExt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple extension",
			input:    "file.txt",
			expected: "file",
		},
		{
			name:     "multiple extensions",
			input:    "_nsakey.v2.dive.rule",
			expected: "_nsakey.v2.dive",
		},
		{
			name:     "no extension",
			input:    "filename",
			expected: "filename",
		},
		{
			name:     "path with extension",
			input:    "path/to/file.txt",
			expected: "file",
		},
		{
			name:     "hidden file",
			input:    ".gitignore",
			expected: ".gitignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBaseNameWithoutExt(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractBaseNameWithoutExt(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
