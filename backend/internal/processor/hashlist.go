package processor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// HashlistProcessor handles asynchronous hashlist processing
type HashlistProcessor struct {
	statusChan       chan<- ProcessingStatus
	hashTypeDetector *HashTypeDetector
	deduplicator     *Deduplicator
	validator        *HashlistValidator
}

// ProcessingStatus represents the current processing state
type ProcessingStatus struct {
	HashlistID    int64
	TotalHashes   int
	Processed     int
	Duplicates    int
	Invalid       int
	LastUpdate    time.Time
	CurrentStatus string
}

// NewHashlistProcessor creates a new processor instance
func NewHashlistProcessor(statusChan chan<- ProcessingStatus) *HashlistProcessor {
	return &HashlistProcessor{
		statusChan:       statusChan,
		hashTypeDetector: NewHashTypeDetector(),
		deduplicator:     NewDeduplicator(),
		validator:        NewHashlistValidator(),
	}
}

// Process handles the full processing pipeline
func (p *HashlistProcessor) Process(
	ctx context.Context,
	file io.Reader,
	hashlist *models.HashList,
) error {
	// Validate file format
	if _, err := p.validator.ValidateFormat(file); err != nil {
		return fmt.Errorf("invalid file format: %w", err)
	}

	// Initialize processing status
	status := ProcessingStatus{
		HashlistID:    hashlist.ID,
		CurrentStatus: "validating",
	}
	p.sendStatus(status)

	// Process file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Validate hash format
		if err := p.validator.ValidateHash(line, hashlist.HashTypeID); err != nil {
			status.Invalid++
			continue
		}

		// Detect hash type if not specified
		if hashlist.HashTypeID == 0 {
			hashTypeID, confidence := p.hashTypeDetector.Detect(line)
			if confidence < 0.8 {
				status.Invalid++
				continue
			}
			hashlist.HashTypeID = hashTypeID
		}

		// Normalize hash
		normalized := normalizeHash(line, hashlist.HashTypeID)

		// Deduplicate
		if !p.deduplicator.Add(normalized) {
			status.Duplicates++
			continue
		}

		status.Processed++
		status.TotalHashes++

		// Send periodic updates
		if status.Processed%1000 == 0 {
			p.sendStatus(status)
		}
	}

	// Final status update
	status.CurrentStatus = "completed"
	p.sendStatus(status)
	return nil
}

func (p *HashlistProcessor) sendStatus(status ProcessingStatus) {
	status.LastUpdate = time.Now()
	select {
	case p.statusChan <- status:
	default:
		debug.Error("Status channel blocked, dropping update")
	}
}

// Helper functions would include:
// - normalizeHash()
// - NewHashTypeDetector()
// - NewDeduplicator()
// - NewHashlistValidator()
