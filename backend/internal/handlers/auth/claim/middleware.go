package claim

import (
	"net/http"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// NewClaimCodeMiddleware creates a middleware that validates claim codes
func NewClaimCodeMiddleware(voucherService *services.ClaimVoucherService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debug.Info("Processing claim code authentication")

			// Extract claim code from header
			claimCode := r.Header.Get("X-Claim-Code")
			if claimCode == "" {
				debug.Error("No claim code provided")
				http.Error(w, "Claim code required", http.StatusUnauthorized)
				return
			}

			// Validate claim code
			if err := voucherService.ValidateClaimCode(r.Context(), claimCode); err != nil {
				debug.Error("Invalid claim code: %v", err)
				http.Error(w, "Invalid claim code", http.StatusUnauthorized)
				return
			}

			debug.Info("Valid claim code provided")
			next.ServeHTTP(w, r)
		})
	}
}
