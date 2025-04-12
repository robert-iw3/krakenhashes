package hashutils

import (
	"strings"
	"unicode"
)

// --- Username Extraction ---

// UsernameRule defines how to extract a username from a hash string
type UsernameRule struct {
	Separator     string
	UsernameIndex int // 0 for first, -1 for last relative to separator splits.
}

// usernameExtractionRules maps hash_type_id to a specific extraction rule.
// Add more rules here as specific hash types are analyzed.
var usernameExtractionRules = map[int]UsernameRule{
	// Rules based on known formats (username often last after ':')
	12:   {Separator: ":", UsernameIndex: -1}, // PostgreSQL
	22:   {Separator: ":", UsernameIndex: -1}, // Juniper NetScreen/SSG
	23:   {Separator: ":", UsernameIndex: -1}, // Skype
	4711: {Separator: ":", UsernameIndex: -1}, // Huawei sha1(md5($pass).$salt)
	// Add more rules as needed...
	// 21: {Separator: ":", UsernameIndex: -1}, // osCommerce - Verify format if username is expected

	// Example for NTLM where user might be first (user:rid:lm:nt)
	1000: {Separator: ":", UsernameIndex: 0},
}

// isValidUsernameCandidate performs basic checks on a potential username found by heuristic.
func isValidUsernameCandidate(candidate string) bool {
	if len(candidate) == 0 || len(candidate) > 128 { // Reject empty or excessively long strings
		return false
	}
	hasLetterOrDigit := false
	allHex := true
	for _, r := range candidate {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasLetterOrDigit = true
		}
		// Basic check for non-printable or space characters. Adjust if needed.
		if r <= ' ' || r > '~' {
			return false
		}
		// Check if it looks purely hexadecimal (potential hash fragment)
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			allHex = false
		}
	}
	// Reject if empty, or if it looks like purely hex and is long (likely a hash)
	if !hasLetterOrDigit || (allHex && len(candidate) > 16) {
		return false
	}

	return true
}

// ExtractUsername attempts to extract a username based on defined rules
// or a heuristic focusing on the start of the hash.
// Returns nil if no username is confidently found.
func ExtractUsername(rawHash string, hashTypeID int) *string {
	// 1. Check for specific rules first
	rule, exists := usernameExtractionRules[hashTypeID]
	if exists {
		parts := strings.Split(rawHash, rule.Separator)
		var username string
		isValidIndex := false

		if rule.UsernameIndex == -1 { // Last element
			if len(parts) > 1 {
				username = parts[len(parts)-1]
				isValidIndex = true
			}
		} else if rule.UsernameIndex >= 0 && rule.UsernameIndex < len(parts) { // Specific index
			username = parts[rule.UsernameIndex]
			isValidIndex = true
		}

		// If rule existed and we extracted based on index, return it (even if empty for now)
		// We might add validation here later if needed for rule-based extractions.
		if isValidIndex {
			// Return the extracted part. If it's empty, it's still the result of the rule.
			return &username
		}
		// If rule existed but index was out of bounds, don't proceed to heuristic.
		return nil // Rule was defined but didn't apply cleanly to this input format.

	} else {
		// 2. Apply Heuristic: Check start of string if no rule exists
		firstColon := strings.Index(rawHash, ":")
		firstDollar := strings.Index(rawHash, "$") // Common in formats like $type$salt$hash

		sepIndex := -1

		// Find the earliest separator index
		if firstColon != -1 && firstDollar != -1 {
			if firstColon < firstDollar {
				sepIndex = firstColon
			} else {
				sepIndex = firstDollar
			}
		} else if firstColon != -1 {
			sepIndex = firstColon
		} else if firstDollar != -1 {
			sepIndex = firstDollar
		}

		// If a separator exists and is not the very first character
		if sepIndex > 0 {
			potentialUsername := rawHash[0:sepIndex]
			if isValidUsernameCandidate(potentialUsername) {
				// Log potentially? logging.Debugf("Heuristically extracted username '%s' from hash type %d", potentialUsername, hashTypeID)
				return &potentialUsername
			}
		}
	}

	// 3. No username found by rule or heuristic
	return nil
}

// --- Hash Processing ---

// HashProcessor defines a function type for processing a raw hash string.
type HashProcessor func(rawHash string) string

// hashProcessingRules maps hash_type_id to a specific processing function.
// Add more functions for types where needs_processing is true.
var hashProcessingRules = map[int]HashProcessor{
	1000: processNTLM, // NTLM needs specific LM hash extraction
}

// ProcessHashIfNeeded processes the hash only if required by the hash type's needs_processing flag.
// Otherwise, it returns the original hash.
func ProcessHashIfNeeded(rawHash string, hashTypeID int, needsProcessing bool) string {
	if !needsProcessing {
		return rawHash // No processing needed for this type
	}

	processor, exists := hashProcessingRules[hashTypeID]
	if !exists {
		// Log warning? logging.Warnf("Hash type %d needs processing, but no processor function found.", hashTypeID)
		return rawHash // No specific processor, return original as fallback
	}

	// Log info? logging.Debugf("Processing hash type %d using specific processor.", hashTypeID)
	return processor(rawHash)
}

// processNTLM extracts just the NTHASH portion for Hashcat mode 1000 processing.
// It prioritizes formats like user:rid:LM:NT::: and then LM:NT, finally just NT.
func processNTLM(rawHash string) string {
	parts := strings.Split(rawHash, ":")

	// Check common NTLM dump formats:
	// 1. user:rid:LMHASH:NTHASH:... (NT is index 3)
	if len(parts) >= 4 && len(parts[3]) == 32 && isHexString(parts[3]) {
		return parts[3] // Return NT hash
	}
	// 2. LMHASH:NTHASH (NT is index 1) - Common Hashcat input
	if len(parts) >= 2 && len(parts[1]) == 32 && isHexString(parts[1]) {
		return parts[1] // Return NT hash
	}
	// 3. Just NTHASH (NT is index 0) - Less common, but possible
	if len(parts) == 1 && len(parts[0]) == 32 && isHexString(parts[0]) {
		return parts[0] // Return NT hash
	}

	// Log warning? logging.Warnf("Could not extract NT hash from NTLM input format: %s", rawHash)
	// Fallback: return the original hash if format is unexpected or doesn't contain a clear NT hash.
	return rawHash
}

// isHexString checks if a string consists only of hexadecimal characters.
func isHexString(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
