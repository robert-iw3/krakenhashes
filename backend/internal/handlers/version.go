package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/version"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// GetVersion handles the version information endpoint
func GetVersion(w http.ResponseWriter, r *http.Request) {
	debug.Debug("Version information requested from %s", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")

	versionInfo := version.GetVersionInfo()
	if err := json.NewEncoder(w).Encode(versionInfo); err != nil {
		debug.Error("Failed to encode version information: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	debug.Debug("Version information sent successfully")
}
