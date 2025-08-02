package version

// Version is set by ldflags during build
var Version = "dev"

// GetVersion returns the current agent version
func GetVersion() string {
	return Version
}