# Agent Package Test Coverage Plan

## Current Coverage: 9.3%

## Areas Needing Coverage:

### connection.go (Currently 0% for most functions)

1. **Certificate Management**:
   - `certificatesExist()` - Test checking for cert files
   - `loadCACertificate()` - Test loading CA cert with valid/invalid files
   - `loadClientCertificate()` - Test loading client cert
   - `isCertificateError()` - Test certificate error detection
   - `RenewCertificates()` - Test certificate renewal flow

2. **Connection Lifecycle**:
   - `NewConnection()` - Test connection creation with various configs
   - `connect()` - Test WebSocket connection establishment
   - `maintainConnection()` - Test connection maintenance and reconnection
   - `Start()` / `Stop()` / `Close()` - Test lifecycle management

3. **Message Handling**:
   - `readPump()` - Test reading messages from WebSocket
   - `writePump()` - Test writing messages to WebSocket
   - `Send()` - Test message sending with queuing
   - `handleFileSyncAsync()` - Test async file sync handling

4. **Status and Monitoring**:
   - `createAgentStatusMessage()` - Test status message creation
   - `SendJobProgress()` - Test job progress updates
   - `SendHashcatOutput()` - Test hashcat output sending
   - `DetectAndSendDevices()` - Test device detection reporting

5. **Configuration**:
   - `getEnvDuration()` - Test environment variable parsing
   - `initTimingConfig()` - Test timing configuration

### registration.go (Low coverage ~21%)

1. **Registration Flow**:
   - `RegisterAgent()` - Full registration flow with mocked HTTP
   - `sendRegistrationRequest()` - Test HTTP request/response
   - `storeCredentials()` - Test credential storage
   - `LoadCredentials()` - Test credential loading

2. **Certificate Download**:
   - `downloadCACertificate()` - Test CA cert download with retries

## Testing Strategy:

1. **Use Existing Mocks**:
   - MockWebSocketConn for WebSocket testing
   - MockHTTPClient for HTTP requests
   - MockHardwareMonitor for device detection
   - MockSyncManager for file sync

2. **Create Test Fixtures**:
   - Valid/invalid certificates
   - Test configuration files
   - Mock server responses

3. **Test Patterns**:
   - Table-driven tests for multiple scenarios
   - Concurrent access tests
   - Error handling and edge cases
   - Resource cleanup verification