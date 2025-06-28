package rule

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/fsutil"
	"github.com/google/uuid"
)

// Manager handles rule operations
type Manager interface {
	ListRules(ctx context.Context, filters map[string]interface{}) ([]*models.Rule, error)
	GetRule(ctx context.Context, id int) (*models.Rule, error)
	GetRuleByFilename(ctx context.Context, filename string) (*models.Rule, error)
	GetRuleByMD5Hash(ctx context.Context, md5Hash string) (*models.Rule, error)
	GetRuleByName(ctx context.Context, name string) (*models.Rule, error)
	AddRule(ctx context.Context, req *models.RuleAddRequest, userID uuid.UUID) (*models.Rule, error)
	UpdateRule(ctx context.Context, id int, req *models.RuleUpdateRequest, userID uuid.UUID) (*models.Rule, error)
	DeleteRule(ctx context.Context, id int) error
	VerifyRule(ctx context.Context, id int, req *models.RuleVerifyRequest) error
	UpdateRuleFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error
	AddRuleTag(ctx context.Context, id int, tag string, userID uuid.UUID) error
	DeleteRuleTag(ctx context.Context, id int, tag string) error
	GetRulePath(filename string, ruleType string) string
	CountRulesInFile(filepath string) (int64, error)
	CalculateFileMD5(filepath string) (string, error)
}

// RuleStore defines the interface for rule data storage operations
type RuleStore interface {
	// Rule operations
	ListRules(ctx context.Context, filter *models.RuleFilter) ([]*models.Rule, error)
	GetRule(ctx context.Context, id int) (*models.Rule, error)
	GetRuleByFilename(ctx context.Context, filename string) (*models.Rule, error)
	GetRuleByMD5Hash(ctx context.Context, md5Hash string) (*models.Rule, error)
	GetRuleByName(ctx context.Context, name string) (*models.Rule, error)
	CreateRule(ctx context.Context, rule *models.Rule) error
	UpdateRule(ctx context.Context, rule *models.Rule) error
	DeleteRule(ctx context.Context, id int) error
	UpdateRuleVerification(ctx context.Context, id int, status string, ruleCount *int64) error
	UpdateRuleFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error

	// Tag operations
	GetRuleTags(ctx context.Context, id int) ([]string, error)
	AddRuleTag(ctx context.Context, id int, tag string, userID uuid.UUID) error
	DeleteRuleTag(ctx context.Context, id int, tag string) error
}

type manager struct {
	store            RuleStore
	rulesDir         string
	maxUploadSize    int64
	allowedFormats   []string
	allowedMimeTypes []string
}

// NewManager creates a new rule manager
func NewManager(store RuleStore, rulesDir string, maxUploadSize int64, allowedFormats, allowedMimeTypes []string) Manager {
	// Ensure rules directory exists
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		debug.Error("Failed to create rules directory: %v", err)
		panic(err)
	}

	// Add .rules to allowed formats if not already present
	hasRulesExt := false
	for _, format := range allowedFormats {
		if format == "rules" {
			hasRulesExt = true
			break
		}
	}
	if !hasRulesExt {
		allowedFormats = append(allowedFormats, "rules")
	}

	return &manager{
		store:            store,
		rulesDir:         rulesDir,
		maxUploadSize:    maxUploadSize,
		allowedFormats:   allowedFormats,
		allowedMimeTypes: allowedMimeTypes,
	}
}

// ListRules retrieves all rules with optional filtering
func (m *manager) ListRules(ctx context.Context, filters map[string]interface{}) ([]*models.Rule, error) {
	// Convert map[string]interface{} to *models.RuleFilter
	filter := &models.RuleFilter{}

	if filters != nil {
		if search, ok := filters["search"].(string); ok {
			filter.Search = search
		}
		if ruleType, ok := filters["rule_type"].(string); ok {
			filter.RuleType = ruleType
		}
		if status, ok := filters["verification_status"].(string); ok {
			filter.VerificationStatus = status
		}
		if sortBy, ok := filters["sort_by"].(string); ok {
			filter.SortBy = sortBy
		}
		if sortOrder, ok := filters["sort_order"].(string); ok {
			filter.SortOrder = sortOrder
		}
	}

	return m.store.ListRules(ctx, filter)
}

// GetRule retrieves a rule by ID
func (m *manager) GetRule(ctx context.Context, id int) (*models.Rule, error) {
	return m.store.GetRule(ctx, id)
}

// GetRuleByFilename retrieves a rule by filename
func (m *manager) GetRuleByFilename(ctx context.Context, filename string) (*models.Rule, error) {
	return m.store.GetRuleByFilename(ctx, filename)
}

// GetRuleByMD5Hash retrieves a rule by MD5 hash
func (m *manager) GetRuleByMD5Hash(ctx context.Context, md5Hash string) (*models.Rule, error) {
	return m.store.GetRuleByMD5Hash(ctx, md5Hash)
}

// GetRuleByName retrieves a rule by its name
func (m *manager) GetRuleByName(ctx context.Context, name string) (*models.Rule, error) {
	return m.store.GetRuleByName(ctx, name)
}

// AddRule adds a new rule
func (m *manager) AddRule(ctx context.Context, req *models.RuleAddRequest, userID uuid.UUID) (*models.Rule, error) {
	// Create rule model
	rule := &models.Rule{
		Name:               req.Name,
		Description:        req.Description,
		RuleType:           req.RuleType,
		FileName:           req.FileName,
		MD5Hash:            req.MD5Hash,
		FileSize:           req.FileSize,
		RuleCount:          req.RuleCount,
		CreatedBy:          userID,
		VerificationStatus: "pending",
		Tags:               req.Tags,
	}

	// Create rule in database
	if err := m.store.CreateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// UpdateRule updates an existing rule
func (m *manager) UpdateRule(ctx context.Context, id int, req *models.RuleUpdateRequest, userID uuid.UUID) (*models.Rule, error) {
	// Get existing rule
	rule, err := m.store.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("rule not found")
	}

	// Update fields
	rule.Name = req.Name
	rule.Description = req.Description
	rule.RuleType = req.RuleType
	rule.UpdatedBy = userID

	// Update in database
	if err := m.store.UpdateRule(ctx, rule); err != nil {
		return nil, err
	}

	// Handle tags
	if req.Tags != nil {
		// Get current tags
		currentTags, err := m.store.GetRuleTags(ctx, id)
		if err != nil {
			return nil, err
		}

		// Add new tags
		for _, tag := range req.Tags {
			found := false
			for _, currentTag := range currentTags {
				if tag == currentTag {
					found = true
					break
				}
			}
			if !found {
				if err := m.store.AddRuleTag(ctx, id, tag, userID); err != nil {
					return nil, err
				}
			}
		}

		// Remove tags that are no longer present
		for _, currentTag := range currentTags {
			found := false
			for _, tag := range req.Tags {
				if currentTag == tag {
					found = true
					break
				}
			}
			if !found {
				if err := m.store.DeleteRuleTag(ctx, id, currentTag); err != nil {
					return nil, err
				}
			}
		}

		// Update tags in rule object
		rule.Tags = req.Tags
	}

	return rule, nil
}

// DeleteRule deletes a rule
func (m *manager) DeleteRule(ctx context.Context, id int) error {
	// Get rule to find filename
	rule, err := m.store.GetRule(ctx, id)
	if err != nil {
		return err
	}
	if rule == nil {
		return fmt.Errorf("rule not found")
	}

	// Delete from database
	if err := m.store.DeleteRule(ctx, id); err != nil {
		return err
	}

	// Delete file
	filePath := filepath.Join(m.rulesDir, rule.FileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		debug.Error("Failed to delete rule file %s: %v", filePath, err)
		// Don't return error, as the database entry is already deleted
	}

	return nil
}

// VerifyRule updates a rule's verification status
func (m *manager) VerifyRule(ctx context.Context, id int, req *models.RuleVerifyRequest) error {
	// Get rule
	rule, err := m.store.GetRule(ctx, id)
	if err != nil {
		return err
	}
	if rule == nil {
		return fmt.Errorf("rule not found")
	}

	// If status is "verified" and rule count is not provided, calculate it
	if req.Status == "verified" && req.RuleCount == nil {
		filePath := filepath.Join(m.rulesDir, rule.FileName)
		ruleCount, err := m.CountRulesInFile(filePath)
		if err != nil {
			debug.Error("Failed to count rules in file %s: %v", filePath, err)
			return err
		}
		req.RuleCount = &ruleCount
	}

	// Update verification status
	return m.store.UpdateRuleVerification(ctx, id, req.Status, req.RuleCount)
}

// UpdateRuleFileInfo updates a rule's file information (MD5 hash and file size)
func (m *manager) UpdateRuleFileInfo(ctx context.Context, id int, md5Hash string, fileSize int64) error {
	return m.store.UpdateRuleFileInfo(ctx, id, md5Hash, fileSize)
}

// AddRuleTag adds a tag to a rule
func (m *manager) AddRuleTag(ctx context.Context, id int, tag string, userID uuid.UUID) error {
	return m.store.AddRuleTag(ctx, id, tag, userID)
}

// DeleteRuleTag deletes a tag from a rule
func (m *manager) DeleteRuleTag(ctx context.Context, id int, tag string) error {
	return m.store.DeleteRuleTag(ctx, id, tag)
}

// GetRulePath returns the full path to a rule file
func (m *manager) GetRulePath(filename string, ruleType string) string {
	// Check if the filename already contains a subdirectory
	if strings.Contains(filename, string(filepath.Separator)) {
		return filepath.Join(m.rulesDir, filename)
	}

	// If no rule type is provided, use a default
	if ruleType == "" {
		ruleType = "hashcat" // Default type

		// Try to determine from filename
		if strings.Contains(strings.ToLower(filename), "john") {
			ruleType = "john"
		}
	}

	// Place in appropriate subdirectory
	return filepath.Join(m.rulesDir, ruleType, filename)
}

// CountRulesInFile counts the number of rules in a file
func (m *manager) CountRulesInFile(filepath string) (int64, error) {
	return fsutil.CountLinesInFile(filepath)
}

// CalculateFileMD5 calculates the MD5 hash of a file
func (m *manager) CalculateFileMD5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
