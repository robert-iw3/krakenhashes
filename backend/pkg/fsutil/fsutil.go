package fsutil

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CountLinesInFile counts the number of lines in a file
func CountLinesInFile(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	// For very large files (over 1GB), use a more efficient counting method
	if fileInfo.Size() > 1024*1024*1024 {
		// Use a buffered reader with a large buffer size for better performance
		const bufferSize = 16 * 1024 * 1024
		reader := bufio.NewReaderSize(file, bufferSize)

		var count int64
		var buf [4096]byte

		for {
			c, err := reader.Read(buf[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return 0, err
			}

			// Count newlines in the buffer
			for i := 0; i < c; i++ {
				if buf[i] == '\n' {
					count++
				}
			}
		}

		// Add 1 if the file doesn't end with a newline
		if count > 0 {
			lastByte := make([]byte, 1)
			if _, err := file.ReadAt(lastByte, fileInfo.Size()-1); err == nil {
				if lastByte[0] != '\n' {
					count++
				}
			}
		}

		return count, nil
	}

	// For regular files, use scanner with increased buffer size
	// Create a scanner with a large buffer to handle long lines
	const maxScanTokenSize = 1024 * 1024 // 1MB buffer
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	var count int64
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirectoryExists checks if a directory exists
func DirectoryExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// WalkDirectory walks a directory and calls the callback for each file
func WalkDirectory(dirPath string, callback func(path string, info os.FileInfo) error) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return callback(path, info)
		}
		return nil
	})
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// SanitizeFilename sanitizes a filename for safe storage
// It replaces spaces and path separators with hyphens and converts to lowercase
func SanitizeFilename(filename string) string {
	// Replace problematic characters with hyphens
	sanitized := strings.ReplaceAll(filename, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")

	// Convert to lowercase for consistency
	sanitized = strings.ToLower(sanitized)

	return sanitized
}

// ExtractBaseNameWithoutExt extracts the base filename without extension(s)
// It handles multi-part extensions like .v2.dive.rule by removing only the final extension
func ExtractBaseNameWithoutExt(filename string) string {
	base := filepath.Base(filename)

	// Remove only the last extension (e.g., .rule, .txt)
	// This preserves multi-part names like name.v2.dive
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	
	// If the result is empty (hidden files like .gitignore), return the original base
	if nameWithoutExt == "" {
		return base
	}
	
	return nameWithoutExt
}
