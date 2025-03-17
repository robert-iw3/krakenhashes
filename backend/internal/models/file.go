package models

// FileType represents the type of file for synchronization
type FileType string

// File types
const (
	FileTypeBinary   FileType = "binary"
	FileTypeWordlist FileType = "wordlist"
	FileTypeRule     FileType = "rule"
	FileTypeHashlist FileType = "hashlist"
)

// FileInfo represents information about a file for synchronization
type FileInfo struct {
	Name     string   `json:"name"`
	Hash     string   `json:"hash"`
	Size     int64    `json:"size"`
	FileType FileType `json:"file_type"`
}

// FileSyncRequestPayload represents a request for an agent to report its current files
type FileSyncRequestPayload struct {
	FileTypes []FileType `json:"file_types"`
}

// FileSyncResponsePayload represents an agent's response with its current files
type FileSyncResponsePayload struct {
	AgentID int        `json:"agent_id"`
	Files   []FileInfo `json:"files"`
}

// FileSyncCommandPayload represents a command to download specific files
type FileSyncCommandPayload struct {
	Files []FileInfo `json:"files"`
}
