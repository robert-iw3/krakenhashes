package auth

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/ZerkerEOD/hashdom-backend/internal/services/ca"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/ZerkerEOD/hashdom-backend/pkg/jwt"
)

/*
 * JWTMiddleware handles JWT-based authentication for protected routes.
 * It validates the JWT token from cookies and adds the user ID to the request context.
 *
 * Authentication Flow:
 * 1. Extracts JWT token from 'token' cookie
 * 2. Validates the token cryptographically
 * 3. Verifies the token exists in the database
 * 4. Adds user ID to request context if valid
 *
 * Parameters:
 *   - next: The next handler in the middleware chain
 *
 * Returns:
 *   - http.Handler: Middleware handler that processes JWT authentication
 *
 * Context Values Added:
 *   - "user_id": (string) The authenticated user's ID
 *
 * Response Codes:
 *   - 401: Unauthorized - Missing, invalid, or revoked token
 *   - Next handler's response if authentication succeeds
 */
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Processing JWT middleware for request: %s %s from %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
		)

		// Extract token from cookie
		cookie, err := r.Cookie("token")
		if err != nil {
			debug.Warning("No token cookie found in request from %s: %v",
				r.RemoteAddr,
				err,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Debug("Token cookie found in request")

		// Validate the token cryptographically
		userID, err := jwt.ValidateJWT(cookie.Value)
		if err != nil {
			debug.Error("Token validation failed for request from %s: %v",
				r.RemoteAddr,
				err,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Debug("Token cryptographically validated for user ID: %s", userID)

		// Verify token exists in database
		exists, err := database.TokenExists(cookie.Value)
		if err != nil {
			debug.Error("Error checking token in database: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !exists {
			debug.Warning("Token not found in database for user ID: %s", userID)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		debug.Info("Token validated successfully for user ID: %s, IP: %s",
			userID,
			r.RemoteAddr,
		)

		// Add user ID to context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		debug.Debug("Added user ID %s to request context", userID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
		debug.Debug("Completed processing request for user ID: %s, Path: %s",
			userID,
			r.URL.Path,
		)
	})
}

// CertificateAuthMiddleware authenticates agents using client certificates
func CertificateAuthMiddleware(next http.Handler) http.Handler {
	caManager := ca.NewManager()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debug.Debug("Processing certificate authentication")

		// Check if client provided a certificate
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			debug.Error("No client certificate provided")
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		cert := r.TLS.PeerCertificates[0]

		// Get CA instance
		caInstance, err := caManager.GetCA()
		if err != nil {
			debug.Error("Failed to get CA instance: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Verify certificate
		if err := verifyCertificate(caInstance, cert); err != nil {
			debug.Error("Certificate verification failed: %v", err)
			http.Error(w, "Invalid certificate", http.StatusUnauthorized)
			return
		}

		// Get agent ID from certificate
		agentID := cert.Subject.CommonName

		// Create agent context
		agent := &models.Agent{
			ID: agentID,
		}
		ctx := context.WithValue(r.Context(), "agent", agent)

		// Proceed with request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func verifyCertificate(ca *ca.CA, cert *x509.Certificate) error {
	// Verify certificate against CA
	if err := ca.VerifyCertificate(cert); err != nil {
		return err
	}

	// Check if certificate is expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	// Verify certificate usage
	hasClientAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}
	if !hasClientAuth {
		return fmt.Errorf("certificate not valid for client authentication")
	}

	return nil
}
