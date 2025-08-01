package mocks

import (
	"sync"
	
	filesync "github.com/ZerkerEOD/krakenhashes/agent/internal/sync"
)

// MockSyncManager implements a mock file sync manager for testing
type MockSyncManager struct {
	mu sync.RWMutex
	
	// Control behavior
	GetFileListFunc     func(fileType string) ([]filesync.FileInfo, error)
	SyncFileFunc        func(fileType, filePath string) error
	FileExistsFunc      func(fileType, filePath string) (bool, error)
	GetFilePathFunc     func(fileType, fileName string) string
	ProcessSyncCommandFunc  func(command interface{}) error
	
	// Default data
	Files map[string][]filesync.FileInfo
	
	// Call tracking
	GetFileListCalls      int
	SyncFileCalls         int
	FileExistsCalls       int
	GetFilePathCalls      int
	ProcessSyncCommandCalls int
}

// NewMockSyncManager creates a new mock sync manager
func NewMockSyncManager() *MockSyncManager {
	return &MockSyncManager{
		Files: make(map[string][]filesync.FileInfo),
	}
}

// GetFileList implements sync.Manager
func (m *MockSyncManager) GetFileList(fileType string) ([]filesync.FileInfo, error) {
	m.mu.Lock()
	m.GetFileListCalls++
	m.mu.Unlock()
	
	if m.GetFileListFunc != nil {
		return m.GetFileListFunc(fileType)
	}
	
	// Default implementation
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if files, ok := m.Files[fileType]; ok {
		return files, nil
	}
	
	return []filesync.FileInfo{}, nil
}

// SyncFile implements sync.Manager
func (m *MockSyncManager) SyncFile(fileType, filePath string) error {
	m.mu.Lock()
	m.SyncFileCalls++
	m.mu.Unlock()
	
	if m.SyncFileFunc != nil {
		return m.SyncFileFunc(fileType, filePath)
	}
	
	// Default implementation - success
	return nil
}

// FileExists implements sync.Manager
func (m *MockSyncManager) FileExists(fileType, filePath string) (bool, error) {
	m.mu.Lock()
	m.FileExistsCalls++
	m.mu.Unlock()
	
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(fileType, filePath)
	}
	
	// Default implementation
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if files, ok := m.Files[fileType]; ok {
		for _, f := range files {
			if f.Name == filePath {
				return true, nil
			}
		}
	}
	
	return false, nil
}

// GetFilePath implements sync.Manager
func (m *MockSyncManager) GetFilePath(fileType, fileName string) string {
	m.mu.Lock()
	m.GetFilePathCalls++
	m.mu.Unlock()
	
	if m.GetFilePathFunc != nil {
		return m.GetFilePathFunc(fileType, fileName)
	}
	
	// Default implementation
	return "/mock/path/" + fileType + "/" + fileName
}

// ProcessSyncCommand implements sync.Manager
func (m *MockSyncManager) ProcessSyncCommand(command interface{}) error {
	m.mu.Lock()
	m.ProcessSyncCommandCalls++
	m.mu.Unlock()
	
	if m.ProcessSyncCommandFunc != nil {
		return m.ProcessSyncCommandFunc(command)
	}
	
	// Default implementation - success
	return nil
}

// AddFile is a helper method to add a file for testing
func (m *MockSyncManager) AddFile(fileType string, fileInfo filesync.FileInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Files[fileType] == nil {
		m.Files[fileType] = []filesync.FileInfo{}
	}
	
	m.Files[fileType] = append(m.Files[fileType], fileInfo)
}

// SetFiles is a helper method to set files for a specific type
func (m *MockSyncManager) SetFiles(fileType string, files []filesync.FileInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Files[fileType] = files
}