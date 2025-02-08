package api

import (
	"context"
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// APIKeyMiddleware authenticates requests using API keys
func APIKeyMiddleware(agentService *services.AgentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debug.Info("Processing API key authentication for %s %s", r.Method, r.URL.Path)

			// Get API key from header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				debug.Error("No API key provided")
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}

			// Get agent ID from header
			agentID := r.Header.Get("X-Agent-ID")
			if agentID == "" {
				debug.Error("No agent ID provided")
				http.Error(w, "Agent ID required", http.StatusUnauthorized)
				return
			}

			// Validate API key and get agent
			agent, err := agentService.GetByAPIKey(r.Context(), apiKey)
			if err != nil {
				debug.Error("Invalid API key: %v", err)
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Store agent in context
			ctx := context.WithValue(r.Context(), "agent", agent)
			r = r.WithContext(ctx)

			debug.Info("API key authentication successful for agent %d", agent.ID)
			next.ServeHTTP(w, r)
		})
	}
}
