package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// RespondWithError sends an error response with the given status code and message
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, ErrorResponse{Error: message})
}

// RespondWithJSON sends a JSON response with the given status code and data
func RespondWithJSON(w http.ResponseWriter, code int, data interface{}) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Set status code
	w.WriteHeader(code)

	// Encode response
	if err := json.NewEncoder(w).Encode(data); err != nil {
		debug.Error("Failed to encode JSON response: %v", err)
		// If we can't encode the response, send a plain text error
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ParseJSONBody parses the request body into the given struct
func ParseJSONBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// GetQueryParam gets a query parameter from the request
func GetQueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// GetQueryParamWithDefault gets a query parameter from the request with a default value
func GetQueryParamWithDefault(r *http.Request, key, defaultValue string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetBoolQueryParam gets a boolean query parameter from the request
func GetBoolQueryParam(r *http.Request, key string) bool {
	value := r.URL.Query().Get(key)
	return value == "true" || value == "1" || value == "yes"
}

// GetIntQueryParam gets an integer query parameter from the request
func GetIntQueryParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := json.Number(value).Int64()
	if err != nil {
		return defaultValue
	}

	return int(intValue)
}
