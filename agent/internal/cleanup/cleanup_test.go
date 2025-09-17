package cleanup

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCleanupHashlists tests cleanup of old hashlist files
func TestCleanupHashlists(t *testing.T) {
	// Create temporary directories
	tempDir, err := ioutil.TempDir("", "cleanup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	hashlistDir := filepath.Join(tempDir, "hashlists")
	err = os.MkdirAll(hashlistDir, 0755)
	require.NoError(t, err)

	// Create test files with different ages
	oldFile := filepath.Join(hashlistDir, "old.txt")
	newFile := filepath.Join(hashlistDir, "new.txt")
	keepFile := filepath.Join(hashlistDir, "keep.doc") // Wrong extension

	// Create files
	err = ioutil.WriteFile(oldFile, []byte("old content"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(newFile, []byte("new content"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(keepFile, []byte("keep content"), 0644)
	require.NoError(t, err)

	// Modify the old file's timestamp to be 4 days old
	oldTime := time.Now().Add(-4 * 24 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	require.NoError(t, err)

	// Create cleanup service
	dataDirs := &config.DataDirs{
		Hashlists: hashlistDir,
	}
	service := NewCleanupService(dataDirs)

	// Perform cleanup
	deleted, size := service.cleanupHashlists()

	// Verify results
	assert.Equal(t, 1, deleted, "Should delete 1 old file")
	assert.Greater(t, size, int64(0), "Should report size of deleted file")

	// Verify old file is deleted
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "Old file should be deleted")

	// Verify new file still exists
	_, err = os.Stat(newFile)
	assert.NoError(t, err, "New file should still exist")

	// Verify file with wrong extension still exists
	_, err = os.Stat(keepFile)
	assert.NoError(t, err, "File with wrong extension should still exist")
}

// TestCleanupRuleChunks tests cleanup of temporary rule chunks
func TestCleanupRuleChunks(t *testing.T) {
	// Create temporary directories
	tempDir, err := ioutil.TempDir("", "cleanup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	rulesDir := filepath.Join(tempDir, "rules")
	err = os.MkdirAll(rulesDir, 0755)
	require.NoError(t, err)

	// Create test files
	oldChunk := filepath.Join(rulesDir, "rules_chunk_001.txt")
	newChunk := filepath.Join(rulesDir, "rules_chunk_002.txt")
	baseRule := filepath.Join(rulesDir, "base_rules.txt") // Should not be deleted
	tempChunk := filepath.Join(rulesDir, "temp_rules_001.txt")

	// Create files
	err = ioutil.WriteFile(oldChunk, []byte("chunk 1"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(newChunk, []byte("chunk 2"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(baseRule, []byte("base rules"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(tempChunk, []byte("temp chunk"), 0644)
	require.NoError(t, err)

	// Modify timestamps
	oldTime := time.Now().Add(-4 * 24 * time.Hour)
	err = os.Chtimes(oldChunk, oldTime, oldTime)
	require.NoError(t, err)
	err = os.Chtimes(tempChunk, oldTime, oldTime)
	require.NoError(t, err)
	err = os.Chtimes(baseRule, oldTime, oldTime) // Old but should not be deleted
	require.NoError(t, err)

	// Create cleanup service
	dataDirs := &config.DataDirs{
		Rules: rulesDir,
	}
	service := NewCleanupService(dataDirs)

	// Perform cleanup
	deleted, size := service.cleanupRuleChunks()

	// Verify results
	assert.Equal(t, 2, deleted, "Should delete 2 old chunks")
	assert.Greater(t, size, int64(0), "Should report size of deleted files")

	// Verify old chunks are deleted
	_, err = os.Stat(oldChunk)
	assert.True(t, os.IsNotExist(err), "Old chunk should be deleted")
	_, err = os.Stat(tempChunk)
	assert.True(t, os.IsNotExist(err), "Temp chunk should be deleted")

	// Verify new chunk still exists
	_, err = os.Stat(newChunk)
	assert.NoError(t, err, "New chunk should still exist")

	// Verify base rule file still exists (even though old)
	_, err = os.Stat(baseRule)
	assert.NoError(t, err, "Base rule file should not be deleted")
}

// TestCleanupChunkIDFiles tests cleanup of orphaned chunk ID files
func TestCleanupChunkIDFiles(t *testing.T) {
	// Create temporary directories
	tempDir, err := ioutil.TempDir("", "cleanup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	rulesDir := filepath.Join(tempDir, "rules")
	err = os.MkdirAll(rulesDir, 0755)
	require.NoError(t, err)

	// Create test files
	chunkIDWithChunks := filepath.Join(rulesDir, "job123.chunkid")
	chunkIDOrphaned := filepath.Join(rulesDir, "job456.chunkid")
	associatedChunk := filepath.Join(rulesDir, "job123_chunk_001.txt")

	// Create files
	err = ioutil.WriteFile(chunkIDWithChunks, []byte("chunk info"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(chunkIDOrphaned, []byte("orphan info"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(associatedChunk, []byte("chunk data"), 0644)
	require.NoError(t, err)

	// Make chunk ID files old
	oldTime := time.Now().Add(-4 * 24 * time.Hour)
	err = os.Chtimes(chunkIDWithChunks, oldTime, oldTime)
	require.NoError(t, err)
	err = os.Chtimes(chunkIDOrphaned, oldTime, oldTime)
	require.NoError(t, err)
	// Keep associated chunk new so it's not deleted
	err = os.Chtimes(associatedChunk, time.Now(), time.Now())
	require.NoError(t, err)

	// Create cleanup service
	dataDirs := &config.DataDirs{
		Rules: rulesDir,
	}
	service := NewCleanupService(dataDirs)

	// Perform cleanup
	deleted, _ := service.cleanupChunkIDFiles()

	// Verify results
	assert.Equal(t, 1, deleted, "Should delete 1 orphaned chunk ID file")

	// Verify orphaned chunk ID is deleted
	_, err = os.Stat(chunkIDOrphaned)
	assert.True(t, os.IsNotExist(err), "Orphaned chunk ID should be deleted")

	// Verify chunk ID with associated chunks still exists
	_, err = os.Stat(chunkIDWithChunks)
	assert.NoError(t, err, "Chunk ID with associated chunks should still exist")
}

// TestPerformCleanup tests the complete cleanup process
func TestPerformCleanup(t *testing.T) {
	// Create temporary directories
	tempDir, err := ioutil.TempDir("", "cleanup_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	hashlistDir := filepath.Join(tempDir, "hashlists")
	rulesDir := filepath.Join(tempDir, "rules")
	err = os.MkdirAll(hashlistDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(rulesDir, 0755)
	require.NoError(t, err)

	// Create old files in each directory
	oldHashlist := filepath.Join(hashlistDir, "old.hash")
	oldChunk := filepath.Join(rulesDir, "old_chunk_001.txt")

	err = ioutil.WriteFile(oldHashlist, []byte("hashes"), 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(oldChunk, []byte("chunk"), 0644)
	require.NoError(t, err)

	// Make files old
	oldTime := time.Now().Add(-4 * 24 * time.Hour)
	err = os.Chtimes(oldHashlist, oldTime, oldTime)
	require.NoError(t, err)
	err = os.Chtimes(oldChunk, oldTime, oldTime)
	require.NoError(t, err)

	// Create cleanup service
	dataDirs := &config.DataDirs{
		Hashlists: hashlistDir,
		Rules:     rulesDir,
	}
	service := NewCleanupService(dataDirs)

	// Perform cleanup
	ctx := context.Background()
	service.performCleanup(ctx)

	// Verify files are deleted
	_, err = os.Stat(oldHashlist)
	assert.True(t, os.IsNotExist(err), "Old hashlist should be deleted")
	_, err = os.Stat(oldChunk)
	assert.True(t, os.IsNotExist(err), "Old chunk should be deleted")
}

// TestFormatBytes tests the byte formatting function
func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{10737418240, "10.0 GB"},
	}

	for _, tc := range testCases {
		result := formatBytes(tc.bytes)
		assert.Equal(t, tc.expected, result)
	}
}