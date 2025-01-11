/*
 * Package agent implements agent registration and management functionality
 * for the KrakenHashes agent service.
 */
package agent

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/auth"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// Add file lock management
var (
	fileLocks     = make(map[string]*sync.Mutex)
	fileLocksLock sync.Mutex
	lockTimeout   = 5 * time.Minute
	lastUsed      = make(map[string]time.Time)
)

// FileWithChecksum represents a file with its checksum
type FileWithChecksum struct {
	Data     []byte
	Checksum string
}

// getFileLock returns a mutex for the given file path
func getFileLock(path string) *sync.Mutex {
	fileLocksLock.Lock()
	defer fileLocksLock.Unlock()

	// Cleanup stale locks first
	cleanupStaleLocks()

	if lock, exists := fileLocks[path]; exists {
		lastUsed[path] = time.Now()
		return lock
	}

	lock := &sync.Mutex{}
	fileLocks[path] = lock
	lastUsed[path] = time.Now()
	return lock
}

// calculateChecksum generates a SHA-256 hash of the data
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// verifyChecksum validates the data against its checksum
func verifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := calculateChecksum(data)
	return actualChecksum == expectedChecksum
}

// RegistrationRequest represents the data sent to register an agent
type RegistrationRequest struct {
	ClaimCode string `json:"claim_code"`
	Hostname  string `json:"hostname"`
}

// RegistrationResponse represents the server's response to registration
type RegistrationResponse struct {
	AgentID       int               `json:"agent_id"`
	DownloadToken string            `json:"download_token"`
	Endpoints     map[string]string `json:"endpoints"`
	APIKey        string            `json:"api_key"`
	Certificate   string            `json:"certificate"`    // PEM-encoded client certificate
	PrivateKey    string            `json:"private_key"`    // PEM-encoded private key
	CACertificate string            `json:"ca_certificate"` // PEM-encoded CA certificate
}

// getHostname retrieves the system hostname for agent identification
func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		debug.Error("Failed to get system hostname: %v", err)
		return "", err
	}

	if hostname == "" {
		debug.Error("System returned empty hostname")
		return "", fmt.Errorf("empty hostname")
	}

	debug.Info("Retrieved hostname: %s", hostname)
	return hostname, nil
}

// storeCredentials stores the agent ID, API key, and certificates
func storeCredentials(agentID int, apiKey string, cert, key, caCert []byte) error {
	debug.Info("Storing agent credentials")

	// Get config directory
	configDir := config.GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		debug.Error("Failed to create config directory: %v", err)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Store agent ID and API key using auth package
	if err := auth.SaveAgentKey(configDir, apiKey, fmt.Sprintf("%d", agentID)); err != nil {
		debug.Error("Failed to save agent key: %v", err)
		return fmt.Errorf("failed to save agent key: %w", err)
	}

	// Store CA certificate
	caCertPath := filepath.Join(configDir, "ca.crt")
	caCertLock := getFileLock(caCertPath)
	caCertLock.Lock()
	defer caCertLock.Unlock()

	if err := os.WriteFile(caCertPath, caCert, 0644); err != nil {
		debug.Error("Failed to write CA certificate: %v", err)
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Store client certificate
	certPath := filepath.Join(configDir, "client.crt")
	certLock := getFileLock(certPath)
	certLock.Lock()
	defer certLock.Unlock()

	if err := os.WriteFile(certPath, cert, 0644); err != nil {
		debug.Error("Failed to write client certificate: %v", err)
		return fmt.Errorf("failed to write client certificate: %w", err)
	}

	// Store private key
	keyPath := filepath.Join(configDir, "client.key")
	keyLock := getFileLock(keyPath)
	keyLock.Lock()
	defer keyLock.Unlock()

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		debug.Error("Failed to write private key: %v", err)
		return fmt.Errorf("failed to write private key: %w", err)
	}

	debug.Info("Successfully stored agent credentials")
	return nil
}

// LoadCredentials loads the agent ID, API key, and certificates from disk
func LoadCredentials() (string, string, error) {
	debug.Info("Loading agent credentials")

	// Get config directory
	configDir := config.GetConfigDir()

	// Load credentials
	credsPath := filepath.Join(configDir, "credentials")
	credsLock := getFileLock(credsPath)
	credsLock.Lock()
	defer credsLock.Unlock()

	credsData, err := os.ReadFile(credsPath)
	if err != nil {
		if os.IsNotExist(err) {
			debug.Info("No existing credentials found")
			return "", "", nil
		}
		debug.Error("Failed to read credentials: %v", err)
		return "", "", fmt.Errorf("failed to read credentials: %w", err)
	}

	// Parse credentials
	parts := strings.Split(string(credsData), ":")
	if len(parts) != 2 {
		debug.Error("Invalid credentials format")
		return "", "", fmt.Errorf("invalid credentials format")
	}

	// Check if certificates exist
	certPath := filepath.Join(configDir, "client.crt")
	keyPath := filepath.Join(configDir, "client.key")
	if _, err := os.Stat(certPath); err != nil {
		debug.Error("Client certificate not found: %v", err)
		return "", "", fmt.Errorf("client certificate not found: %w", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		debug.Error("Private key not found: %v", err)
		return "", "", fmt.Errorf("private key not found: %w", err)
	}

	debug.Info("Successfully loaded agent credentials")
	return parts[0], parts[1], nil
}

// downloadCACertificate downloads the CA certificate from the HTTP endpoint
func downloadCACertificate(urlConfig *config.URLConfig) error {
	debug.Info("Downloading CA certificate")

	// Create HTTP client
	client := &http.Client{}

	// Get CA certificate URL
	caURL := urlConfig.GetCACertURL()
	debug.Info("Downloading CA certificate from: %s", caURL)

	// Make request
	resp, err := client.Get(caURL)
	if err != nil {
		debug.Error("Failed to download CA certificate: %v", err)
		return fmt.Errorf("failed to download CA certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		debug.Error("Server returned non-200 status: %d", resp.StatusCode)
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Read response body
	caCert, err := io.ReadAll(resp.Body)
	if err != nil {
		debug.Error("Failed to read CA certificate: %v", err)
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Create config directory if it doesn't exist
	configDir := config.GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		debug.Error("Failed to create config directory: %v", err)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save CA certificate
	caCertPath := filepath.Join(configDir, "ca.crt")
	if err := os.WriteFile(caCertPath, caCert, 0644); err != nil {
		debug.Error("Failed to write CA certificate: %v", err)
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	debug.Info("Successfully downloaded and saved CA certificate")
	return nil
}

// sendRegistrationRequest sends the registration request to the server
func sendRegistrationRequest(urlConfig *config.URLConfig, req *RegistrationRequest) (*http.Response, error) {
	// Download CA certificate first
	if err := downloadCACertificate(urlConfig); err != nil {
		return nil, fmt.Errorf("failed to download CA certificate: %w", err)
	}

	// Now load the downloaded CA certificate
	certPool, err := loadCACertificate(urlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create request
	url := urlConfig.GetRegistrationURL()
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Create HTTP client with TLS config
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// commentOutClaimCode comments out the claim code in the .env file after successful registration
func commentOutClaimCode() error {
	envFile := ".env"
	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "KH_CLAIM_CODE=") && !strings.HasPrefix(line, "#") {
			lines[i] = "# " + line + " # Commented out after successful registration"
			break
		}
	}

	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(envFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %v", err)
	}

	return nil
}

// RegisterAgent handles the agent registration process using a claim code.
// It will:
// 1. Send the claim code to the server
// 2. Receive agent ID and API key
// 3. Store the credentials locally
// 4. Comment out the claim code in .env after successful registration
func RegisterAgent(claimCode string, urlConfig *config.URLConfig) error {
	debug.Info("Registering agent with claim code")

	// Prepare registration request
	hostname, err := getHostname()
	if err != nil {
		hostname = "unknown"
		debug.Warning("Could not get hostname: %v", err)
	}

	reqBody := &RegistrationRequest{
		ClaimCode: claimCode,
		Hostname:  hostname,
	}

	// Send registration request
	resp, err := sendRegistrationRequest(urlConfig, reqBody)
	if err != nil {
		return fmt.Errorf("registration request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var regResp RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %v", err)
	}

	// Decode certificates and private key
	certBlock, _ := pem.Decode([]byte(regResp.Certificate))
	if certBlock == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	keyBlock, _ := pem.Decode([]byte(regResp.PrivateKey))
	if keyBlock == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	caCertBlock, _ := pem.Decode([]byte(regResp.CACertificate))
	if caCertBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	// Store credentials and certificates
	if err := storeCredentials(regResp.AgentID, regResp.APIKey, []byte(regResp.Certificate), []byte(regResp.PrivateKey), []byte(regResp.CACertificate)); err != nil {
		return fmt.Errorf("failed to store credentials: %v", err)
	}

	// Comment out claim code in .env
	if err := commentOutClaimCode(); err != nil {
		debug.Warning("Failed to comment out claim code: %v", err)
	}

	debug.Info("Agent registration successful")
	return nil
}

// cleanupStaleLocks removes locks that haven't been used for a while
func cleanupStaleLocks() {
	now := time.Now()
	for path, lastUse := range lastUsed {
		if now.Sub(lastUse) > lockTimeout {
			delete(fileLocks, path)
			delete(lastUsed, path)
		}
	}
}
