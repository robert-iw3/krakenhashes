package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// HashProcessor handles hash-type specific transformations
type HashProcessor struct {
	mu          sync.RWMutex
	handlers    map[int]HashHandler
	rulesLoader RulesLoader
}

// HashHandler defines the interface for hash type processors
type HashHandler interface {
	Process(hash string) (string, bool)
	Validate(hash string) bool
}

// RulesLoader loads processing rules from storage
type RulesLoader interface {
	LoadRules(ctx context.Context, hashTypeID int) ([]string, error)
}

// NewHashProcessor creates a new hash processor
func NewHashProcessor(rulesLoader RulesLoader) *HashProcessor {
	return &HashProcessor{
		handlers:    make(map[int]HashHandler),
		rulesLoader: rulesLoader,
	}
}

// Process applies type-specific transformations to a hash
func (p *HashProcessor) Process(ctx context.Context, hash string, hashTypeID int) (string, error) {
	handler, err := p.getHandler(ctx, hashTypeID)
	if err != nil {
		return "", fmt.Errorf("failed to get handler: %w", err)
	}

	processed, changed := handler.Process(hash)
	if !changed {
		return hash, nil
	}

	if !handler.Validate(processed) {
		return "", fmt.Errorf("invalid hash after processing")
	}

	return processed, nil
}

// getHandler loads or creates a handler for the given hash type
func (p *HashProcessor) getHandler(ctx context.Context, hashTypeID int) (HashHandler, error) {
	p.mu.RLock()
	handler, exists := p.handlers[hashTypeID]
	p.mu.RUnlock()

	if exists {
		return handler, nil
	}

	return p.loadHandler(ctx, hashTypeID)
}

// loadHandler dynamically loads processing rules and creates a new handler
func (p *HashProcessor) loadHandler(ctx context.Context, hashTypeID int) (HashHandler, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check again in case another goroutine loaded it
	if handler, exists := p.handlers[hashTypeID]; exists {
		return handler, nil
	}

	rules, err := p.rulesLoader.LoadRules(ctx, hashTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	handler := newDynamicHandler(hashTypeID, rules)
	p.handlers[hashTypeID] = handler

	return handler, nil
}

// dynamicHandler implements HashHandler with configurable rules
type dynamicHandler struct {
	hashTypeID int
	patterns   []*regexp.Regexp
	extractors []*regexp.Regexp
}

func newDynamicHandler(hashTypeID int, rules []string) HashHandler {
	h := &dynamicHandler{
		hashTypeID: hashTypeID,
	}

	for _, rule := range rules {
		// Parse rule format: "pattern|extractor"
		parts := strings.Split(rule, "|")
		if len(parts) != 2 {
			debug.Error("Invalid rule format: %s", rule)
			continue
		}

		pattern, err := regexp.Compile(parts[0])
		if err != nil {
			debug.Error("Failed to compile pattern %s: %v", parts[0], err)
			continue
		}

		extractor, err := regexp.Compile(parts[1])
		if err != nil {
			debug.Error("Failed to compile extractor %s: %v", parts[1], err)
			continue
		}

		h.patterns = append(h.patterns, pattern)
		h.extractors = append(h.extractors, extractor)
	}

	return h
}

func (h *dynamicHandler) Process(hash string) (string, bool) {
	for i, pattern := range h.patterns {
		if pattern.MatchString(hash) {
			matches := h.extractors[i].FindStringSubmatch(hash)
			if len(matches) > 1 {
				return matches[1], true
			}
		}
	}
	return hash, false
}

func (h *dynamicHandler) Validate(hash string) bool {
	for _, pattern := range h.patterns {
		if pattern.MatchString(hash) {
			return true
		}
	}
	return false
}
