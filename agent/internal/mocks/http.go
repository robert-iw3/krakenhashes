package mocks

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

// MockHTTPClient implements a mock HTTP client for testing
type MockHTTPClient struct {
	// Control behavior
	DoFunc     func(req *http.Request) (*http.Response, error)
	CallCount  int
	LastRequest *http.Request
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{}
}

// Do implements the http.Client Do method
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.CallCount++
	m.LastRequest = req
	
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	
	// Default response
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"success": true}`))),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// MockResponseWriter implements http.ResponseWriter for testing
type MockResponseWriter struct {
	HeaderMap   http.Header
	Body        *bytes.Buffer
	Code        int
	WroteHeader bool
}

// NewMockResponseWriter creates a new mock response writer
func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
		Code:      200,
	}
}

// Header implements http.ResponseWriter
func (m *MockResponseWriter) Header() http.Header {
	return m.HeaderMap
}

// Write implements http.ResponseWriter
func (m *MockResponseWriter) Write(data []byte) (int, error) {
	if !m.WroteHeader {
		m.WriteHeader(200)
	}
	return m.Body.Write(data)
}

// WriteHeader implements http.ResponseWriter
func (m *MockResponseWriter) WriteHeader(code int) {
	if !m.WroteHeader {
		m.Code = code
		m.WroteHeader = true
	}
}

// MockRoundTripper implements http.RoundTripper for testing
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
	CallCount     int
}

// RoundTrip implements http.RoundTripper
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.CallCount++
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}
	
	// Default response
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// MockHTTPServer provides a simple mock HTTP server for testing
type MockHTTPServer struct {
	URL        *url.URL
	Handlers   map[string]http.HandlerFunc
	CallCounts map[string]int
}

// NewMockHTTPServer creates a new mock HTTP server
func NewMockHTTPServer(baseURL string) (*MockHTTPServer, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	
	return &MockHTTPServer{
		URL:        u,
		Handlers:   make(map[string]http.HandlerFunc),
		CallCounts: make(map[string]int),
	}, nil
}

// Handle registers a handler for a specific path
func (m *MockHTTPServer) Handle(path string, handler http.HandlerFunc) {
	m.Handlers[path] = handler
}

// ServeHTTP implements http.Handler
func (m *MockHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	m.CallCounts[path]++
	
	if handler, ok := m.Handlers[path]; ok {
		handler(w, r)
		return
	}
	
	// Default 404 response
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - Not Found"))
}