package mocks

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MockWebSocketConn implements a mock WebSocket connection for testing
type MockWebSocketConn struct {
	mu sync.Mutex
	
	// Message channels
	ReadMessages  chan []byte
	WriteMessages chan []byte
	
	// Control behavior
	ReadError    error
	WriteError   error
	CloseError   error
	Closed       bool
	
	// Handlers
	PingHandlerFunc func(appData string) error
	PongHandlerFunc func(appData string) error
	
	// Deadlines
	ReadDeadline  time.Time
	WriteDeadline time.Time
}

// NewMockWebSocketConn creates a new mock WebSocket connection
func NewMockWebSocketConn() *MockWebSocketConn {
	return &MockWebSocketConn{
		ReadMessages:  make(chan []byte, 10),
		WriteMessages: make(chan []byte, 10),
	}
}

// Close implements websocket.Conn
func (m *MockWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Closed {
		return nil
	}
	
	m.Closed = true
	close(m.ReadMessages)
	close(m.WriteMessages)
	
	return m.CloseError
}

// LocalAddr implements websocket.Conn
func (m *MockWebSocketConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
}

// RemoteAddr implements websocket.Conn  
func (m *MockWebSocketConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 31337}
}

// WriteMessage implements websocket.Conn
func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Closed {
		return websocket.ErrCloseSent
	}
	
	if m.WriteError != nil {
		return m.WriteError
	}
	
	select {
	case m.WriteMessages <- data:
		return nil
	default:
		return nil // Don't block in tests
	}
}

// ReadMessage implements websocket.Conn
func (m *MockWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	if m.ReadError != nil {
		return 0, nil, m.ReadError
	}
	
	select {
	case msg, ok := <-m.ReadMessages:
		if !ok {
			return 0, nil, websocket.ErrCloseSent
		}
		return websocket.TextMessage, msg, nil
	case <-time.After(100 * time.Millisecond): // Timeout for tests
		return 0, nil, errors.New("read timeout")
	}
}

// NextWriter implements websocket.Conn
func (m *MockWebSocketConn) NextWriter(messageType int) (w io.WriteCloser, err error) {
	// Not implemented for basic testing
	return nil, nil
}

// WriteControl implements websocket.Conn
func (m *MockWebSocketConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.Closed {
		return websocket.ErrCloseSent
	}
	
	return m.WriteError
}

// NextReader implements websocket.Conn
func (m *MockWebSocketConn) NextReader() (messageType int, r io.Reader, err error) {
	// Not implemented for basic testing
	return 0, nil, nil
}

// ReadJSON implements websocket.Conn
func (m *MockWebSocketConn) ReadJSON(v interface{}) error {
	_, msg, err := m.ReadMessage()
	if err != nil {
		return err
	}
	
	return json.Unmarshal(msg, v)
}

// WriteJSON implements websocket.Conn
func (m *MockWebSocketConn) WriteJSON(v interface{}) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}
	
	return m.WriteMessage(websocket.TextMessage, msg)
}

// WritePreparedMessage implements websocket.Conn
func (m *MockWebSocketConn) WritePreparedMessage(pm *websocket.PreparedMessage) error {
	// Not implemented for basic testing
	return nil
}

// SetWriteDeadline implements websocket.Conn
func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WriteDeadline = t
	return nil
}

// SetReadDeadline implements websocket.Conn
func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReadDeadline = t
	return nil
}

// SetReadLimit implements websocket.Conn
func (m *MockWebSocketConn) SetReadLimit(limit int64) {
	// Not implemented for basic testing
}

// CloseHandler implements websocket.Conn
func (m *MockWebSocketConn) CloseHandler() func(code int, text string) error {
	return nil
}

// SetCloseHandler implements websocket.Conn
func (m *MockWebSocketConn) SetCloseHandler(h func(code int, text string) error) {
	// Not implemented for basic testing
}

// PingHandler implements websocket.Conn
func (m *MockWebSocketConn) PingHandler() func(appData string) error {
	return m.PingHandlerFunc
}

// SetPingHandler implements websocket.Conn
func (m *MockWebSocketConn) SetPingHandler(h func(appData string) error) {
	m.PingHandlerFunc = h
}

// PongHandler implements websocket.Conn
func (m *MockWebSocketConn) PongHandler() func(appData string) error {
	return m.PongHandlerFunc
}

// SetPongHandler implements websocket.Conn
func (m *MockWebSocketConn) SetPongHandler(h func(appData string) error) {
	m.PongHandlerFunc = h
}

// UnderlyingConn implements websocket.Conn
func (m *MockWebSocketConn) UnderlyingConn() net.Conn {
	return nil
}

// EnableWriteCompression implements websocket.Conn
func (m *MockWebSocketConn) EnableWriteCompression(enable bool) {
	// Not implemented for basic testing
}

// SetCompressionLevel implements websocket.Conn
func (m *MockWebSocketConn) SetCompressionLevel(level int) error {
	// Not implemented for basic testing
	return nil
}

// Subprotocol implements websocket.Conn
func (m *MockWebSocketConn) Subprotocol() string {
	return ""
}