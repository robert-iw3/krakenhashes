package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
)

// DBRulesLoader loads processing rules from the database
type DBRulesLoader struct {
	hashTypeRepo repository.HashTypeRepository
}

// NewDBRulesLoader creates a new database rules loader
func NewDBRulesLoader(hashTypeRepo repository.HashTypeRepository) *DBRulesLoader {
	return &DBRulesLoader{
		hashTypeRepo: hashTypeRepo,
	}
}

// LoadRules implements RulesLoader interface
func (l *DBRulesLoader) LoadRules(ctx context.Context, hashTypeID int) ([]string, error) {
	hashType, err := l.hashTypeRepo.GetByID(ctx, hashTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash type: %w", err)
	}

	if hashType == nil {
		return nil, fmt.Errorf("hash type not found")
	}

	// Rules are stored as comma-separated pattern|extractor pairs
	// Check if ProcessingLogic is nil or points to an empty string
	if hashType.ProcessingLogic == nil || *hashType.ProcessingLogic == "" {
		return []string{}, nil
	}

	// Dereference the pointer to get the string value for Split
	return strings.Split(*hashType.ProcessingLogic, ","), nil
}
