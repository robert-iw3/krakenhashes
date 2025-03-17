package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/auth"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	filesync "github.com/ZerkerEOD/krakenhashes/agent/internal/sync"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
	"github.com/gorilla/websocket"
)

// WSMessageType represents different types of WebSocket messages
type WSMessageType string

const (
	WSTypeHardwareInfo WSMessageType = "hardware_info"
	WSTypeMetrics      WSMessageType = "metrics"
	WSTypeHeartbeat    WSMessageType = "heartbeat"
	WSTypeAgentStatus  WSMessageType = "agent_status"

	// File synchronization message types
	WSTypeFileSyncRequest  WSMessageType = "file_sync_request"
	WSTypeFileSyncResponse WSMessageType = "file_sync_response"
	WSTypeFileSyncCommand  WSMessageType = "file_sync_command"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type         WSMessageType   `json:"type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	HardwareInfo *types.Info     `json:"hardware_info,omitempty"`
	Metrics      *MetricsData    `json:"metrics,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
}

// FileSyncRequestPayload represents a request for the agent to report its current files
type FileSyncRequestPayload struct {
	FileTypes []string `json:"file_types"` // "wordlist", "rule", "binary"
}

// FileInfo represents information about a file for synchronization
type FileInfo = filesync.FileInfo

// FileSyncResponsePayload represents the agent's response with its current files
type FileSyncResponsePayload struct {
	AgentID int        `json:"agent_id"`
	Files   []FileInfo `json:"files"`
}

// FileSyncCommandPayload represents a command to download specific files
type FileSyncCommandPayload struct {
	Files []FileInfo `json:"files"`
}

// MetricsData represents the metrics data sent to the server
type MetricsData struct {
	AgentID     int                `json:"agent_id"`
	CollectedAt time.Time          `json:"collected_at"`
	CPUs        []CPUMetrics       `json:"cpus"`
	GPUs        []GPUMetrics       `json:"gpus"`
	Memory      MemoryMetrics      `json:"memory"`
	Disk        []DiskMetrics      `json:"disk"`
	Network     []NetworkMetrics   `json:"network"`
	Process     []ProcessMetrics   `json:"process"`
	Custom      map[string]float64 `json:"custom,omitempty"`
}

// CPUMetrics represents CPU performance metrics
type CPUMetrics struct {
	Index       int     `json:"index"`
	Usage       float64 `json:"usage"`
	Temperature float64 `json:"temperature"`
	Frequency   float64 `json:"frequency"`
}

// GPUMetrics represents GPU performance metrics
type GPUMetrics struct {
	Index       int     `json:"index"`
	Usage       float64 `json:"usage"`
	Temperature float64 `json:"temperature"`
	Memory      float64 `json:"memory"`
	PowerUsage  float64 `json:"power_usage"`
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	UsagePerc float64 `json:"usage_perc"`
}

// DiskMetrics represents disk usage metrics
type DiskMetrics struct {
	Path      string  `json:"path"`
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	UsagePerc float64 `json:"usage_perc"`
}

// NetworkMetrics represents network interface metrics
type NetworkMetrics struct {
	Interface string `json:"interface"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
}

// ProcessMetrics represents process metrics
type ProcessMetrics struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	CPUUsage   float64 `json:"cpu_usage"`
	MemoryUsed uint64  `json:"memory_used"`
}

// Default connection timing values
const (
	defaultWriteWait  = 10 * time.Second
	defaultPongWait   = 60 * time.Second
	defaultPingPeriod = 54 * time.Second
	maxMessageSize    = 512 * 1024 // 512KB
)

// Connection timing configuration
var (
	writeWait  time.Duration
	pongWait   time.Duration
	pingPeriod time.Duration
)

// getEnvDuration gets a duration from an environment variable with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	debug.Info("Attempting to load environment variable: %s", key)
	value := os.Getenv(key)
	debug.Info("Environment variable %s value: %q", key, value)

	if value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			debug.Info("Successfully parsed %s: %v", key, duration)
			return duration
		}
		debug.Warning("Invalid %s value: %s, using default: %v", key, value, defaultValue)
	}
	debug.Info("No %s environment variable found, using default: %v", key, defaultValue)
	return defaultValue
}

// initTimingConfig initializes the timing configuration from environment variables
func initTimingConfig() {
	debug.Info("Initializing WebSocket timing configuration")
	writeWait = getEnvDuration("KH_WRITE_WAIT", defaultWriteWait)
	pongWait = getEnvDuration("KH_PONG_WAIT", defaultPongWait)
	pingPeriod = getEnvDuration("KH_PING_PERIOD", defaultPingPeriod)
	debug.Info("WebSocket timing configuration initialized:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)
}

// Connection represents a WebSocket connection to the server
type Connection struct {
	// The WebSocket connection
	ws *websocket.Conn

	// URL configuration for the connection
	urlConfig *config.URLConfig

	// Hardware monitor
	hwMonitor *hardware.Monitor

	// Channel for outbound messages
	send chan []byte

	// Channel to signal connection closure
	done chan struct{}

	// Atomic flag to track connection status
	isConnected atomic.Bool

	// TLS configuration
	tlsConfig *tls.Config

	// File synchronization
	fileSync *filesync.FileSync
}

// loadCACertificate loads the CA certificate from disk
func loadCACertificate(urlConfig *config.URLConfig) (*x509.CertPool, error) {
	debug.Info("Loading CA certificate")
	certPool := x509.NewCertPool()

	// Try to load from disk
	certPath := filepath.Join(config.GetConfigDir(), "ca.crt")
	if _, err := os.Stat(certPath); err == nil {
		debug.Info("Found existing CA certificate at: %s", certPath)
		certData, err := os.ReadFile(certPath)
		if err != nil {
			debug.Error("Failed to read CA certificate: %v", err)
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		if !certPool.AppendCertsFromPEM(certData) {
			debug.Error("Failed to parse CA certificate")
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		debug.Info("Successfully loaded CA certificate from disk")
		return certPool, nil
	}

	debug.Error("CA certificate not found at: %s", certPath)
	return nil, fmt.Errorf("CA certificate not found")
}

// loadClientCertificate loads the client certificate and key from disk
func loadClientCertificate() (tls.Certificate, error) {
	debug.Info("Loading client certificate")
	certPath := filepath.Join(config.GetConfigDir(), "client.crt")
	keyPath := filepath.Join(config.GetConfigDir(), "client.key")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		debug.Error("Failed to load client certificate: %v", err)
		return tls.Certificate{}, fmt.Errorf("failed to load client certificate: %w", err)
	}

	debug.Info("Successfully loaded client certificate")
	return cert, nil
}

// NewConnection creates a new WebSocket connection instance
func NewConnection(urlConfig *config.URLConfig) (*Connection, error) {
	debug.Info("Creating new WebSocket connection")

	// Initialize timing configuration
	initTimingConfig()

	// Initialize hardware monitor
	hwMonitor, err := hardware.NewMonitor()
	if err != nil {
		debug.Error("Failed to create hardware monitor: %v", err)
		return nil, fmt.Errorf("failed to create hardware monitor: %w", err)
	}

	// Load CA certificate
	certPool, err := loadCACertificate(urlConfig)
	if err != nil {
		debug.Error("Failed to load CA certificate: %v", err)
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Load client certificate
	clientCert, err := loadClientCertificate()
	if err != nil {
		debug.Error("Failed to load client certificate: %v", err)
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return &Connection{
		urlConfig: urlConfig,
		hwMonitor: hwMonitor,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		tlsConfig: tlsConfig,
	}, nil
}

// connect establishes a WebSocket connection to the server
func (c *Connection) connect() error {
	debug.Info("Starting WebSocket connection attempt")

	// Get initial hardware information
	if err := c.hwMonitor.UpdateInfo(); err != nil {
		debug.Error("Failed to get hardware info: %v", err)
		return fmt.Errorf("failed to get hardware info: %w", err)
	}

	// Load API key and agent ID
	apiKey, agentID, err := auth.LoadAgentKey(config.GetConfigDir())
	if err != nil {
		debug.Error("Failed to load API key: %v", err)
		return fmt.Errorf("failed to load API key: %w", err)
	}
	debug.Info("Successfully loaded API key")

	// Get WebSocket URL from config
	wsURL := c.urlConfig.GetWebSocketURL()
	debug.Info("Generated WebSocket URL: %s", wsURL)

	// Parse server URL
	u, err := url.Parse(wsURL)
	if err != nil {
		debug.Error("Invalid server URL: %v", err)
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// Add agent ID to query parameters
	q := u.Query()
	u.RawQuery = q.Encode()
	debug.Info("Attempting WebSocket connection to: %s", u.String())
	debug.Debug("Connection details - Protocol: %s, Host: %s, Path: %s, Query: %s",
		u.Scheme, u.Host, u.Path, u.RawQuery)

	// Setup headers with API key
	header := http.Header{}
	header.Set("X-API-Key", apiKey)
	header.Set("X-Agent-ID", agentID)

	// Configure WebSocket dialer with TLS
	dialer := websocket.Dialer{
		WriteBufferSize:  maxMessageSize,
		ReadBufferSize:   maxMessageSize,
		HandshakeTimeout: writeWait,
		TLSClientConfig:  c.tlsConfig,
	}

	debug.Info("Initiating WebSocket connection with timing configuration:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)
	debug.Info("- TLS Enabled: %v", c.tlsConfig != nil)
	if c.tlsConfig != nil {
		debug.Debug("TLS Configuration:")
		debug.Debug("- Client Certificates: %d", len(c.tlsConfig.Certificates))
		debug.Debug("- RootCAs: %v", c.tlsConfig.RootCAs != nil)
	}

	ws, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			debug.Error("WebSocket connection failed with status: %d", resp.StatusCode)
			debug.Debug("Response headers: %v", resp.Header)
			body, _ := io.ReadAll(resp.Body)
			debug.Debug("Response body: %s", string(body))
			resp.Body.Close()
		} else {
			debug.Error("WebSocket connection failed with no response: %v", err)
			debug.Debug("Error type: %T", err)
			if urlErr, ok := err.(*url.Error); ok {
				debug.Debug("URL error details: %v", urlErr)
			}
		}
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	c.ws = ws
	debug.Info("Successfully established WebSocket connection")
	c.isConnected.Store(true)

	// Send initial hardware information
	hwInfo := c.hwMonitor.GetInfo()

	// Marshal hardware info to JSON for the payload
	hwInfoJSON, err := json.Marshal(hwInfo)
	if err != nil {
		debug.Error("Failed to marshal hardware info: %v", err)
		c.ws.Close()
		return fmt.Errorf("failed to marshal hardware info: %w", err)
	}

	msg := WSMessage{
		Type:      WSTypeHardwareInfo,
		Payload:   hwInfoJSON,
		Timestamp: time.Now(),
	}

	if err := c.ws.WriteJSON(msg); err != nil {
		debug.Error("Failed to send initial hardware info: %v", err)
		c.ws.Close()
		return fmt.Errorf("failed to send hardware info: %w", err)
	}

	debug.Info("Successfully sent initial hardware information")
	return nil
}

// maintainConnection maintains the WebSocket connection with exponential backoff
func (c *Connection) maintainConnection() {
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Minute // Increased to 10 minutes
	multiplier := 2.0
	attempt := 1

	debug.Info("Starting connection maintenance routine")

	for {
		select {
		case <-c.done:
			debug.Info("Connection maintenance stopped")
			return
		default:
			if !c.isConnected.Load() {
				debug.Info("Connection state: disconnected")
				debug.Info("Reconnection attempt %d - Waiting %v before retry", attempt, backoff)
				time.Sleep(backoff)

				if err := c.connect(); err != nil {
					debug.Error("Reconnection attempt %d failed: %v", attempt, err)
					nextBackoff := time.Duration(float64(backoff) * multiplier)
					if nextBackoff > maxBackoff {
						nextBackoff = maxBackoff
					}
					debug.Info("Increasing backoff from %v to %v (max: %v)", backoff, nextBackoff, maxBackoff)
					backoff = nextBackoff
					attempt++
				} else {
					debug.Info("Reconnection successful after %d attempts - Resetting backoff", attempt)
					backoff = 1 * time.Second
					attempt = 1
					debug.Info("Starting read and write pumps")
					go c.readPump()
					go c.writePump()
				}
			} else {
				debug.Debug("Connection state: connected")
			}
			time.Sleep(time.Second)
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Connection) readPump() {
	defer func() {
		debug.Info("ReadPump closing, marking connection as disconnected")
		c.isConnected.Store(false)
		c.Close()
	}()

	debug.Info("Starting readPump with timing configuration:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))

	// Set handlers for ping/pong
	c.ws.SetPingHandler(func(appData string) error {
		debug.Info("Received ping from server, sending pong")
		err := c.ws.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			debug.Error("Failed to set read deadline: %v", err)
			return err
		}
		// Send pong response immediately
		err = c.ws.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(writeWait))
		if err != nil {
			debug.Error("Failed to send pong: %v", err)
			c.isConnected.Store(false)
			return err
		}
		debug.Info("Successfully sent pong response")
		return nil
	})

	c.ws.SetPongHandler(func(string) error {
		debug.Info("Received pong from server")
		err := c.ws.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			debug.Error("Failed to set read deadline: %v", err)
			c.isConnected.Store(false)
			return err
		}
		debug.Info("Successfully updated read deadline after pong")
		return nil
	})

	debug.Info("Ping/Pong handlers configured")

	for {
		var msg WSMessage
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				debug.Error("Unexpected WebSocket close error: %v", err)
			} else {
				debug.Info("WebSocket connection closed: %v", err)
			}
			c.isConnected.Store(false)
			break
		}

		// Handle different message types
		switch msg.Type {
		case WSTypeHeartbeat:
			// Send heartbeat response
			response := WSMessage{
				Type:      WSTypeHeartbeat,
				Timestamp: time.Now(),
			}
			if err := c.ws.WriteJSON(response); err != nil {
				debug.Error("Failed to send heartbeat response: %v", err)
			}
		case WSTypeMetrics:
			// Server requested metrics update
			if err := c.hwMonitor.UpdateMetrics(); err != nil {
				debug.Error("Failed to update hardware metrics: %v", err)
				continue
			}
			// TODO: Implement metrics collection and sending
			// This will be implemented later when we add the metrics collection functionality
			debug.Info("Metrics update requested but not yet implemented")
		case WSTypeHardwareInfo:
			// Server requested hardware info update
			if err := c.hwMonitor.UpdateInfo(); err != nil {
				debug.Error("Failed to update hardware info: %v", err)
				continue
			}
			hwInfo := c.hwMonitor.GetInfo()

			// Marshal hardware info to JSON for the payload
			hwInfoJSON, err := json.Marshal(hwInfo)
			if err != nil {
				debug.Error("Failed to marshal hardware info: %v", err)
				continue
			}

			response := WSMessage{
				Type:      WSTypeHardwareInfo,
				Payload:   hwInfoJSON,
				Timestamp: time.Now(),
			}
			if err := c.ws.WriteJSON(response); err != nil {
				debug.Error("Failed to send hardware info: %v", err)
			}
		case WSTypeFileSyncRequest:
			// Server requested file list
			debug.Info("Received file sync request")

			// Parse the request payload
			var requestPayload FileSyncRequestPayload
			if err := json.Unmarshal(msg.Payload, &requestPayload); err != nil {
				debug.Error("Failed to parse file sync request: %v", err)
				continue
			}

			// Initialize file sync if not already done
			if c.fileSync == nil {
				dataDirs, err := config.GetDataDirs()
				if err != nil {
					debug.Error("Failed to get data directories: %v", err)
					continue
				}

				c.fileSync, err = filesync.NewFileSync(c.urlConfig, dataDirs)
				if err != nil {
					debug.Error("Failed to initialize file sync: %v", err)
					continue
				}
			}

			// Scan directories for files
			filesByType := make(map[string][]filesync.FileInfo)
			for _, fileType := range requestPayload.FileTypes {
				files, err := c.fileSync.ScanDirectory(fileType)
				if err != nil {
					debug.Error("Failed to scan %s directory: %v", fileType, err)
					continue
				}
				filesByType[fileType] = files
			}

			// Flatten the file list
			var allFiles []filesync.FileInfo
			for _, files := range filesByType {
				allFiles = append(allFiles, files...)
			}

			// Get agent ID
			agentID, err := GetAgentID()
			if err != nil {
				debug.Error("Failed to get agent ID: %v", err)
				continue
			}

			// Prepare response
			responsePayload := FileSyncResponsePayload{
				AgentID: agentID,
				Files:   allFiles,
			}

			// Marshal response payload
			payloadBytes, err := json.Marshal(responsePayload)
			if err != nil {
				debug.Error("Failed to marshal file sync response: %v", err)
				continue
			}

			// Send response
			response := WSMessage{
				Type:      WSTypeFileSyncResponse,
				Payload:   payloadBytes,
				Timestamp: time.Now(),
			}

			if err := c.ws.WriteJSON(response); err != nil {
				debug.Error("Failed to send file sync response: %v", err)
			} else {
				debug.Info("Sent file sync response with %d files", len(allFiles))
			}

		case WSTypeFileSyncCommand:
			// Server sent file sync command
			debug.Info("Received file sync command")

			// Parse the command payload
			var commandPayload FileSyncCommandPayload
			if err := json.Unmarshal(msg.Payload, &commandPayload); err != nil {
				debug.Error("Failed to parse file sync command: %v", err)
				continue
			}

			// Initialize file sync if not already done
			if c.fileSync == nil {
				dataDirs, err := config.GetDataDirs()
				if err != nil {
					debug.Error("Failed to get data directories: %v", err)
					continue
				}

				c.fileSync, err = filesync.NewFileSync(c.urlConfig, dataDirs)
				if err != nil {
					debug.Error("Failed to initialize file sync: %v", err)
					continue
				}
			}

			// Process each file in the command
			for _, file := range commandPayload.Files {
				go func(file filesync.FileInfo) {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
					defer cancel()

					debug.Info("Downloading file: %s (%s)", file.Name, file.FileType)
					if err := c.fileSync.DownloadFile(ctx, file.FileType, file.Name, file.Hash); err != nil {
						debug.Error("Failed to download file %s: %v", file.Name, err)
					} else {
						debug.Info("Successfully downloaded file: %s", file.Name)
					}
				}(file)
			}

			debug.Info("Started downloading %d files", len(commandPayload.Files))
		default:
			debug.Warning("Received unknown message type: %s", msg.Type)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	// Add a status update ticker that runs every minute
	statusTicker := time.NewTicker(1 * time.Minute)
	defer func() {
		debug.Info("WritePump closing, marking connection as disconnected")
		ticker.Stop()
		statusTicker.Stop()
		c.isConnected.Store(false)
		c.Close()
	}()

	debug.Info("Starting writePump with timing configuration:")
	debug.Info("- Write Wait: %v", writeWait)
	debug.Info("- Pong Wait: %v", pongWait)
	debug.Info("- Ping Period: %v", pingPeriod)
	debug.Info("- Status Update Period: 1m")

	// Send initial status update
	if err := c.sendAgentStatusUpdate(); err != nil {
		debug.Error("Failed to send initial status update: %v", err)
	}

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				debug.Info("Send channel closed, marking as disconnected")
				c.isConnected.Store(false)
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				debug.Error("Failed to get next writer: %v", err)
				c.isConnected.Store(false)
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				debug.Error("Failed to close writer: %v", err)
				c.isConnected.Store(false)
				return
			}
		case <-ticker.C:
			debug.Info("Local ticker triggered, sending ping to server")
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				debug.Error("Failed to send ping: %v", err)
				c.isConnected.Store(false)
				return
			}
			debug.Info("Successfully sent ping to server")
		case <-statusTicker.C:
			debug.Info("Status ticker triggered, sending agent status update")
			if err := c.sendAgentStatusUpdate(); err != nil {
				debug.Error("Failed to send agent status update: %v", err)
			} else {
				debug.Info("Successfully sent agent status update")
			}
		case <-c.done:
			debug.Info("WritePump received done signal")
			return
		}
	}
}

// sendAgentStatusUpdate sends an agent status update to the server
func (c *Connection) sendAgentStatusUpdate() error {
	if !c.isConnected.Load() {
		return fmt.Errorf("not connected")
	}

	// Create status payload
	statusPayload := map[string]interface{}{
		"status":      "active",
		"version":     "1.0.0", // Replace with actual version
		"updated_at":  time.Now(),
		"environment": map[string]string{},
	}

	// Marshal status payload to JSON
	statusJSON, err := json.Marshal(statusPayload)
	if err != nil {
		debug.Error("Failed to marshal agent status: %v", err)
		return fmt.Errorf("failed to marshal agent status: %w", err)
	}

	// Create and send status message
	msg := WSMessage{
		Type:      WSTypeAgentStatus,
		Payload:   statusJSON,
		Timestamp: time.Now(),
	}

	if err := c.ws.WriteJSON(msg); err != nil {
		debug.Error("Failed to send agent status update: %v", err)
		return fmt.Errorf("failed to send agent status update: %w", err)
	}

	return nil
}

// Close closes the WebSocket connection
func (c *Connection) Close() {
	debug.Info("Closing connection")
	if c.ws != nil {
		debug.Debug("Closing WebSocket connection")
		c.ws.Close()
	}
	c.isConnected.Store(false)
}

// Stop completely stops the connection and maintenance routines
func (c *Connection) Stop() {
	debug.Info("Stopping connection and maintenance")
	select {
	case <-c.done:
		debug.Debug("Connection already stopped")
	default:
		debug.Debug("Closing done channel")
		close(c.done)
	}
	c.Close()
}

// Send sends a message to the server
func (c *Connection) Send(message []byte) error {
	if !c.isConnected.Load() {
		return fmt.Errorf("not connected")
	}

	select {
	case c.send <- message:
		return nil
	default:
		c.Close()
		return fmt.Errorf("send buffer full")
	}
}

// Start starts the WebSocket connection
func (c *Connection) Start() error {
	debug.Info("Starting WebSocket connection")

	if err := c.connect(); err != nil {
		debug.Error("Initial connection failed: %v", err)
		return err
	}

	go c.maintainConnection()
	go c.readPump()
	go c.writePump()

	return nil
}

// Connect establishes a WebSocket connection to the server
func (c *Connection) Connect() error {
	return c.connect()
}
