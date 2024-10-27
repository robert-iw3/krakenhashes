package routes

import (
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/auth"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/api"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/dashboard"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/hashlists"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/handlers/jobs"
	"github.com/gorilla/mux"
)

func SetupRoutes(r *mux.Router) {
	// Public routes
	r.HandleFunc("/api/login", auth.LoginHandler).Methods("POST")
	r.HandleFunc("/api/logout", auth.LogoutHandler).Methods("POST")
	r.HandleFunc("/api/check-auth", auth.CheckAuthHandler).Methods("GET")

	// Protected routes
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.JWTMiddleware)

	protected.HandleFunc("/dashboard", dashboard.GetDashboard).Methods("GET")
	protected.HandleFunc("/hashlists", hashlists.GetHashlists).Methods("GET")
	protected.HandleFunc("/jobs", jobs.GetJobs).Methods("GET")

	// API subrouter (protected)
	apiRouter := protected.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/some-endpoint", api.SomeAPIHandler).Methods("GET")

	// Agent subrouter

}

// TODO: Implement agent-related functionality
// func unusedAgentPlaceholder() {
// 	_ = agent.SomeFunction // Replace SomeFunction with an actual function from the agent package
// }
