# Sync Package Test Coverage Plan

## Current Coverage: 14.2%

## Areas Needing Coverage:

### sync.go (Most functions at 0%)

1. **Directory Operations**:
   - `ScanAllDirectories()` - Test scanning all file type directories
   - `SyncDirectory()` - Test full directory synchronization
   - `ScanDirectory()` - Increase coverage (currently 31%)

2. **File Operations**:
   - `DownloadFileFromInfo()` - Test file download with checksum verification
   - `DownloadFileWithInfoRetry()` - Test download with retry logic
   - `retryOrFailInfo()` - Test retry decision logic

3. **Binary Handling**:
   - `FindExtractedExecutables()` - Test finding extracted binaries
   - `getBinaryIDFromPath()` - Test extracting binary ID from path
   - `ExtractBinary7z()` - Test 7z extraction

4. **Certificate Loading**:
   - `loadCACertificate()` - Test CA cert loading (increase from 62.5%)

## Testing Strategy:

1. **File System Mocking**:
   - Create temporary directories for testing
   - Mock file operations for download tests
   - Test with various file permissions

2. **HTTP Mocking**:
   - Mock download endpoints
   - Test retry logic with failures
   - Test checksum validation

3. **Test Scenarios**:
   - Empty directories
   - Large file lists
   - Concurrent operations
   - Network failures
   - Checksum mismatches
   - Permission errors

4. **Integration Tests**:
   - Full sync workflow
   - Multi-directory sync
   - Binary extraction and validation