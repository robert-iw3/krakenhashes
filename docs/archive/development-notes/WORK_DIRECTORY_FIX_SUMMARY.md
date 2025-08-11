# Work Directory and Job Error Status Fixes

## Summary of Changes

### 1. Removed Work Directory References
- **Agent HashcatExecutor**: Removed all references to `workDirectory` since we're capturing output from stdout
- **Agent JobManager**: Updated to not pass work directory to HashcatExecutor
- **Hashcat Command**: Removed `cmd.Dir = e.workDirectory` to prevent "chdir: no such file or directory" error

### 2. Fixed Hashcat Binary Permissions
- **File Sync**: Enhanced binary extraction to set executable permissions (0755) for hashcat binaries
- **Binary Detection**: Improved detection of executable files during extraction (hashcat, hashcat.exe, hashcat.bin)

### 3. Implemented Job Error Status Updates
- **Agent JobProgress**: Added `Status` and `ErrorMessage` fields to track task status
- **Agent HashcatExecutor**: Updated error handling to include error message in progress updates
- **Backend JobProgress Model**: Added matching `Status` and `ErrorMessage` fields
- **Backend JobWebSocketIntegration**: Added handling for failed status to update both task and job execution
- **Backend JobExecutionRepository**: Added `UpdateErrorMessage` method to store error messages

## Key Changes by File

### Agent Changes
1. `agent/internal/jobs/hashcat_executor.go`
   - Removed `workDirectory` field from struct
   - Updated `NewHashcatExecutor` to not take work directory parameter
   - Added status and error fields to JobProgress struct
   - Updated error handling to send proper error status

2. `agent/internal/jobs/jobs.go`
   - Updated `NewJobManager` to not create work directory
   - Pass only data directory to HashcatExecutor

3. `agent/internal/sync/sync.go`
   - Enhanced binary extraction to set executable permissions (0755)
   - Improved detection of hashcat executables

### Backend Changes
1. `backend/internal/models/jobs.go`
   - Added `Status` and `ErrorMessage` fields to JobProgress struct

2. `backend/internal/integration/job_websocket_integration.go`
   - Added handling for failed status in `HandleJobProgress`
   - Updates task status to failed with error message
   - Updates job execution status to failed

3. `backend/internal/repository/job_execution_repository.go`
   - Added `UpdateErrorMessage` method to store error messages

## Result
- Agent no longer tries to use non-existent work directory
- Hashcat binaries have proper execute permissions after extraction
- Failed jobs now properly show error status instead of remaining as pending
- Error messages are captured and displayed in the UI