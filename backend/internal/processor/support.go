package processor

import (
	"io"
	"regexp"
	"strings"
	"sync"
)

// HashTypeDetector identifies hash types with confidence scores
type HashTypeDetector struct {
	patterns map[int]*regexp.Regexp
}

func NewHashTypeDetector() *HashTypeDetector {
	return &HashTypeDetector{
		patterns: map[int]*regexp.Regexp{
			// NTLM
			1000: regexp.MustCompile(`^[0-9a-fA-F]{32}$`),
			// SHA1
			2000: regexp.MustCompile(`^[0-9a-fA-F]{40}$`),
			// SHA256
			3000: regexp.MustCompile(`^[0-9a-fA-F]{64}$`),
		},
	}
}

func (d *HashTypeDetector) Detect(hash string) (int, float32) {
	for id, pattern := range d.patterns {
		if pattern.MatchString(hash) {
			return id, 1.0 // Exact match = 100% confidence
		}
	}
	return 0, 0.0
}

// Deduplicator tracks seen hashes efficiently
type Deduplicator struct {
	bloomFilter *bloomFilter // Placeholder for actual implementation
	seen        map[string]struct{}
	mu          sync.Mutex
}

func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		seen: make(map[string]struct{}),
	}
}

func (d *Deduplicator) Add(hash string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[hash]; exists {
		return false
	}
	d.seen[hash] = struct{}{}
	return true
}

// HashlistValidator validates hashlist formats and individual hashes
type HashlistValidator struct{}

func NewHashlistValidator() *HashlistValidator {
	return &HashlistValidator{}
}

func (v *HashlistValidator) ValidateFormat(file io.Reader) (string, error) {
	// Basic format detection would go here
	return "raw", nil
}

func (v *HashlistValidator) ValidateHash(hash string, hashTypeID int) error {
	// Basic validation would go here
	return nil
}

// normalizeHash standardizes hash formats
func normalizeHash(hash string, hashTypeID int) string {
	// Convert to lowercase for case-insensitive hashes
	switch hashTypeID {
	case 1000: // NTLM
		return strings.ToLower(hash)
	default:
		return hash
	}
}

// bloomFilter is a placeholder for actual bloom filter implementation
type bloomFilter struct{}
