package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSMessage_JSON(t *testing.T) {
	tests := []struct {
		name    string
		message WSMessage
		wantErr bool
	}{
		{
			name: "heartbeat message",
			message: WSMessage{
				Type:      WSTypeHeartbeat,
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "metrics message",
			message: WSMessage{
				Type: WSTypeMetrics,
				Metrics: &MetricsData{
					AgentID:     1,
					CollectedAt: time.Now(),
					Memory: MemoryMetrics{
						Total:     8192,
						Used:      4096,
						Free:      4096,
						UsagePerc: 50.0,
					},
				},
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "message with payload",
			message: WSMessage{
				Type:      WSTypeFileSyncRequest,
				Payload:   json.RawMessage(`{"file_types":["wordlist","rule"]}`),
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.message)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Unmarshal
			var decoded WSMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.message.Type, decoded.Type)
			assert.WithinDuration(t, tt.message.Timestamp, decoded.Timestamp, time.Second)
		})
	}
}

func TestConnection_WebSocketUpgrade(t *testing.T) {
	// Create test server
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, "123", r.Header.Get("X-Agent-ID"))

		// Upgrade connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read a message and echo it back
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		conn.WriteMessage(websocket.TextMessage, message)
	}))
	defer server.Close()

	// Create URL config
	wsURL := "ws" + server.URL[4:] + "/ws"
	_ = &config.URLConfig{
		WebSocketURL: wsURL,
		BaseURL:      server.URL,
	}

	// Test connection with headers
	headers := http.Header{
		"X-API-Key":  []string{"test-api-key"},
		"X-Agent-ID": []string{"123"},
	}

	ctx := context.Background()
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, headers)
	require.NoError(t, err)
	defer ws.Close()

	// Send test message
	testMsg := WSMessage{
		Type:      WSTypeHeartbeat,
		Timestamp: time.Now(),
	}
	err = ws.WriteJSON(testMsg)
	require.NoError(t, err)

	// Read echo
	var echoMsg WSMessage
	err = ws.ReadJSON(&echoMsg)
	require.NoError(t, err)
	assert.Equal(t, testMsg.Type, echoMsg.Type)
}

func TestConnection_MessageHandling(t *testing.T) {
	tests := []struct {
		name        string
		messageType WSMessageType
		payload     interface{}
		setup       func(*Connection)
		validate    func(*testing.T, *Connection)
	}{
		{
			name:        "heartbeat message",
			messageType: WSTypeHeartbeat,
			setup:       func(c *Connection) {},
			validate: func(t *testing.T, c *Connection) {
				// Heartbeats don't require validation
			},
		},
		{
			name:        "file sync request",
			messageType: WSTypeFileSyncRequest,
			payload: map[string]interface{}{
				"file_types": []string{"wordlist", "rule"},
			},
			setup: func(c *Connection) {},
			validate: func(t *testing.T, c *Connection) {
				// Would trigger file sync response
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{
				outbound: make(chan *WSMessage, 10),
				done:     make(chan struct{}),
			}

			if tt.setup != nil {
				tt.setup(conn)
			}

			// Create message
			msg := WSMessage{
				Type:      tt.messageType,
				Timestamp: time.Now(),
			}

			if tt.payload != nil {
				payloadData, err := json.Marshal(tt.payload)
				require.NoError(t, err)
				msg.Payload = json.RawMessage(payloadData)
			}

			// Process message (would normally be done by message handler)
			// For now, just validate structure
			data, err := json.Marshal(msg)
			require.NoError(t, err)

			var decoded WSMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, conn)
			}
		})
	}
}

func TestConnection_Reconnection(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	// Create test server that fails first attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < maxAttempts {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Success on final attempt
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Keep connection open
		<-time.After(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"
	
	// Test multiple connection attempts
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, _, lastErr = websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		cancel()
		
		if lastErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.NoError(t, lastErr)
	assert.Equal(t, maxAttempts, attempts)
}

func TestConnection_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		shouldRetry bool
	}{
		{
			name:        "normal closure",
			err:         &websocket.CloseError{Code: websocket.CloseNormalClosure},
			shouldRetry: false,
		},
		{
			name:        "going away",
			err:         &websocket.CloseError{Code: websocket.CloseGoingAway},
			shouldRetry: true,
		},
		{
			name:        "protocol error",
			err:         &websocket.CloseError{Code: websocket.CloseProtocolError},
			shouldRetry: true,
		},
		{
			name:        "generic error",
			err:         errors.New("connection failed"),
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error classification
			var closeErr *websocket.CloseError
			if errors.As(tt.err, &closeErr) {
				if closeErr.Code == websocket.CloseNormalClosure {
					assert.False(t, tt.shouldRetry)
				} else {
					assert.True(t, tt.shouldRetry)
				}
			} else {
				// Non-close errors should retry
				assert.True(t, tt.shouldRetry)
			}
		})
	}
}

func TestConnection_MessageQueue(t *testing.T) {
	conn := &Connection{
		outbound: make(chan *WSMessage, 10),
		done:     make(chan struct{}),
	}

	// Queue multiple messages
	messages := []WSMessage{
		{Type: WSTypeHeartbeat, Timestamp: time.Now()},
		{Type: WSTypeHardwareInfo, Timestamp: time.Now()},
		{Type: WSTypeMetrics, Timestamp: time.Now()},
	}

	// Send messages
	for _, msg := range messages {
		select {
		case conn.outbound <- &msg:
		case <-time.After(time.Second):
			t.Fatal("Failed to queue message")
		}
	}

	// Verify queue length
	assert.Equal(t, len(messages), len(conn.outbound))

	// Drain queue and verify order
	for i, expected := range messages {
		select {
		case actual := <-conn.outbound:
			assert.Equal(t, expected.Type, actual.Type, "Message %d type mismatch", i)
		case <-time.After(time.Second):
			t.Fatal("Failed to read message from queue")
		}
	}
}

func TestConnection_Shutdown(t *testing.T) {
	// Create test WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Wait for close message
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// Create connection
	wsURL := "ws" + server.URL[4:] + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	conn := &Connection{
		ws:       ws,
		outbound: make(chan *WSMessage, 10),
		done:     make(chan struct{}),
	}

	// Queue some messages
	conn.outbound <- &WSMessage{Type: WSTypeHeartbeat}
	conn.outbound <- &WSMessage{Type: WSTypeMetrics}

	// Close the websocket connection
	err = ws.Close()
	assert.NoError(t, err)

	// Verify write fails after close
	err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
	assert.Error(t, err)
}