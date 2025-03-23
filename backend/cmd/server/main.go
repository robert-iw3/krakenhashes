package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/database"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/handlers/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/routes"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/rule"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services"
	tlsprovider "github.com/ZerkerEOD/krakenhashes/backend/internal/tls"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/version"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	// Initialize debug package first with default settings
	debug.Reinitialize()
	debug.Info("Debug logging initialized with default settings")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		debug.Error("Failed to get working directory: %v", err)
		os.Exit(1)
	}
	debug.Debug("Current working directory: %s", cwd)

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		debug.Info("Attempting to load .env from current directory: %s", cwd)
		debug.Warning("Failed to load .env file from current directory: %v", err)

		debug.Info("Attempting to load .env from project root")
		err = godotenv.Load("../.env")
		if err != nil {
			debug.Warning("No .env file found, checking environment variables")

			// Check required environment variables
			requiredVars := []string{
				"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
				"KH_TLS_MODE",
			}

			missingVars := []string{}
			for _, v := range requiredVars {
				if os.Getenv(v) == "" {
					missingVars = append(missingVars, v)
				}
			}

			if len(missingVars) > 0 {
				debug.Error("Missing required environment variables: %v", missingVars)
				debug.Error("Please provide these variables either in a .env file or as environment variables")
				os.Exit(1)
			}

			debug.Info("All required environment variables are present")
		} else {
			debug.Info("Successfully loaded .env file from project root")
		}
	} else {
		debug.Info("Successfully loaded .env file from current directory")
	}

	// Reinitialize debug package with environment variables
	debug.Reinitialize()
	debug.Info("Debug logging initialized with environment settings")

	// Load version information
	debug.Info("Loading version information...")
	// Try different paths for versions.json
	versionPaths := []string{
		"/usr/local/share/krakenhashes/versions.json",              // Non-persistent container location
		"/etc/krakenhashes/versions.json",                          // Config directory (for bare metal installs)
		"../versions.json",                                         // From backend directory
		"versions.json",                                            // From current directory
		"../backend/versions.json",                                 // From project root
		filepath.Join(os.Getenv("KH_CONFIG_DIR"), "versions.json"), // From configured config directory
	}

	var versionPath string
	for _, path := range versionPaths {
		if path == "" {
			continue // Skip empty paths (in case KH_CONFIG_DIR is not set)
		}
		if _, err := os.Stat(path); err == nil {
			versionPath = path
			debug.Info("Found version file at: %s", path)
			break
		}
		debug.Debug("Version file not found at: %s", path)
	}

	if versionPath == "" {
		debug.Error("Version file not found in any of the expected locations. Checked:\n%s",
			"- /usr/local/share/krakenhashes/versions.json\n"+
				"- /etc/krakenhashes/versions.json\n"+
				"- "+filepath.Join(cwd, "../versions.json")+"\n"+
				"- "+filepath.Join(cwd, "versions.json")+"\n"+
				"- "+filepath.Join(cwd, "../backend/versions.json")+"\n"+
				"- "+filepath.Join(os.Getenv("KH_CONFIG_DIR"), "versions.json"))
		os.Exit(1)
	}

	if err := version.LoadVersions(versionPath); err != nil {
		debug.Error("Failed to load version information: %v", err)
		os.Exit(1)
	}
	debug.Info("KrakenHashes Backend v%s starting up", version.Version)
	debug.Info("Component versions - Frontend: %s, Agent: %s, API: %s, Database: %s",
		version.Versions.Frontend,
		version.Versions.Agent,
		version.Versions.API,
		version.Versions.Database)

	debug.Info("Initializing application...")

	// Initialize application configuration
	appConfig := config.NewConfig()
	debug.Info("Application configuration initialized")

	// Initialize TLS provider
	debug.Info("Initializing TLS provider")
	tlsProvider, err := tlsprovider.InitializeProvider(appConfig)
	if err != nil {
		debug.Error("Failed to initialize TLS provider: %v", err)
		os.Exit(1)
	}

	// Get TLS configuration for server
	serverTLSConfig, err := tlsProvider.GetTLSConfig()
	if err != nil {
		debug.Error("Failed to get TLS configuration: %v", err)
		os.Exit(1)
	}

	// Initialize database connection
	debug.Info("Initializing database connection")
	sqlDB, err := database.Connect()
	if err != nil {
		debug.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Create DB wrapper for repositories
	dbWrapper := &db.DB{DB: sqlDB}

	// Initialize repositories and services
	debug.Debug("Initializing repositories and services")
	agentRepo := repository.NewAgentRepository(dbWrapper)
	agentService := services.NewAgentService(agentRepo, repository.NewClaimVoucherRepository(dbWrapper), repository.NewFileRepository(dbWrapper))

	// Initialize wordlist and rule managers for monitoring
	wordlistStore := wordlist.NewStore(sqlDB)
	wordlistManager := wordlist.NewManager(
		wordlistStore,
		filepath.Join(appConfig.DataDir, "wordlists"),
		0, // No file size limit
		[]string{"txt", "dict", "lst", "gz", "zip"},                   // Allowed formats
		[]string{"text/plain", "application/gzip", "application/zip"}, // Allowed MIME types
	)

	ruleStore := rule.NewStore(sqlDB)
	ruleManager := rule.NewManager(
		ruleStore,
		filepath.Join(appConfig.DataDir, "rules"),
		0,                                       // No file size limit
		[]string{"rule", "rules", "txt", "lst"}, // Allowed formats
		[]string{"text/plain"},                  // Allowed MIME types
	)

	// Run migrations first
	if err := database.RunMigrations(); err != nil {
		debug.Error("Database migrations failed: %v", err)
		os.Exit(1)
	}
	debug.Info("Database migrations completed successfully")

	// Add a small delay to ensure migrations are fully applied
	debug.Info("Waiting for migrations to be fully applied...")
	time.Sleep(10 * time.Second)

	// Ensure the system user exists
	if err := database.EnsureSystemUser(); err != nil {
		debug.Error("Failed to ensure system user exists: %v", err)
		os.Exit(1)
	}
	debug.Info("System user verified")

	// Use the system user (uuid.Nil) for the monitor service
	systemUserID := uuid.Nil
	debug.Info("Using system user ID for monitor service: %s", systemUserID.String())

	// Initialize monitor service
	monitorService := services.NewMonitorService(
		wordlistManager,
		ruleManager,
		appConfig,
		systemUserID,
	)

	// Create routers
	debug.Info("Creating routers")
	httpRouter := mux.NewRouter()  // For HTTP server (CA certificate)
	httpsRouter := mux.NewRouter() // For HTTPS server (API)

	// Apply global CORS middleware to both routers
	httpRouter.Use(routes.GlobalCORSMiddleware)
	httpsRouter.Use(routes.GlobalCORSMiddleware)

	// Setup routes
	debug.Info("Setting up routes")
	routes.SetupRoutes(httpsRouter, sqlDB, tlsProvider, agentService)

	// Setup CA certificate route on HTTP router
	debug.Info("Setting up CA certificate route")
	tlsHandler := tls.NewHandler(tlsProvider)
	httpRouter.HandleFunc("/ca.crt", tlsHandler.ServeCACertificate).Methods("GET", "HEAD", "OPTIONS")

	// Create HTTPS server
	debug.Info("Creating HTTPS server")
	httpsServer := &http.Server{
		Addr:      appConfig.GetHTTPSAddress(),
		Handler:   httpsRouter,
		TLSConfig: serverTLSConfig,
	}

	// Create HTTP server for CA certificate
	httpServer := &http.Server{
		Addr:    appConfig.GetHTTPAddress(),
		Handler: httpRouter,
	}

	// Start HTTP server in a goroutine for CA certificate
	go func() {
		debug.Info("Starting HTTP server for CA certificate on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.Error("HTTP server error: %v", err)
		}
	}()

	// Channel to wait for server errors
	serverErr := make(chan error, 1)

	// Start HTTPS server in a goroutine
	go func() {
		debug.Info("Starting HTTPS server on %s", httpsServer.Addr)
		if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			debug.Error("HTTPS server error: %v", err)
			serverErr <- err
		}
	}()

	// Wait a moment for servers to start
	time.Sleep(500 * time.Millisecond)

	// Start monitor service after servers and database are ready
	debug.Info("Starting directory monitor service")
	monitorService.Start()
	defer monitorService.Stop()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	debug.Info("Server is ready to handle requests")

	// Block until we receive a signal or server error
	select {
	case err := <-serverErr:
		debug.Error("Server error: %v", err)
		os.Exit(1)
	case sig := <-sigChan:
		debug.Info("Received signal: %v", sig)
		debug.Info("Shutting down server...")

		// Create a deadline for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Shutdown both servers
		if err := httpServer.Shutdown(ctx); err != nil {
			debug.Error("Error during HTTP server shutdown: %v", err)
		}
		if err := httpsServer.Shutdown(ctx); err != nil {
			debug.Error("Error during HTTPS server shutdown: %v", err)
		}
		debug.Info("Server shutdown complete")
	}
}
