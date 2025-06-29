package hardware

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// HashcatDetector detects devices using hashcat -I command
type HashcatDetector struct {
	dataDirectory string
}

// NewHashcatDetector creates a new hashcat-based device detector
func NewHashcatDetector(dataDirectory string) *HashcatDetector {
	return &HashcatDetector{
		dataDirectory: dataDirectory,
	}
}

// DetectDevices detects all available compute devices using hashcat
func (d *HashcatDetector) DetectDevices() (*types.DeviceDetectionResult, error) {
	debug.Info("Starting hashcat device detection")
	
	// Find the most recent hashcat binary
	binaryPath, err := d.findLatestHashcatBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find hashcat binary: %w", err)
	}
	
	debug.Info("Using hashcat binary: %s", binaryPath)
	
	// Run hashcat -I command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, binaryPath, "-I")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run hashcat -I: %w", err)
	}
	
	// Parse the output
	outputStr := string(output)
	debug.Info("Raw hashcat -I output:\n%s", outputStr)
	
	devices, err := d.parseHashcatOutput(outputStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hashcat output: %w", err)
	}
	
	debug.Info("Parsed %d devices before filtering", len(devices))
	
	// Filter out aliases
	filteredDevices := d.filterAliases(devices)
	
	debug.Info("Detected %d devices (filtered from %d total)", len(filteredDevices), len(devices))
	
	return &types.DeviceDetectionResult{
		Devices: filteredDevices,
	}, nil
}

// findLatestHashcatBinary finds the most recent hashcat binary in the binaries directory
func (d *HashcatDetector) findLatestHashcatBinary() (string, error) {
	binariesDir := filepath.Join(d.dataDirectory, "binaries")
	
	// Look for the latest version directory
	entries, err := os.ReadDir(binariesDir)
	if err != nil {
		return "", fmt.Errorf("failed to read binaries directory: %w", err)
	}
	
	var latestVersion int
	var latestDir string
	
	for _, entry := range entries {
		if entry.IsDir() {
			// Try to parse directory name as version number
			version, err := strconv.Atoi(entry.Name())
			if err == nil && version > latestVersion {
				latestVersion = version
				latestDir = entry.Name()
			}
		}
	}
	
	if latestDir == "" {
		return "", fmt.Errorf("no hashcat binary versions found")
	}
	
	// Determine binary extension based on OS
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "hashcat.exe"
	} else {
		binaryName = "hashcat.bin"
	}
	
	binaryPath := filepath.Join(binariesDir, latestDir, binaryName)
	
	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("hashcat binary not found at %s", binaryPath)
	}
	
	return binaryPath, nil
}

// parseHashcatOutput parses the output of hashcat -I command
func (d *HashcatDetector) parseHashcatOutput(output string) ([]types.Device, error) {
	var devices []types.Device
	aliasMap := make(map[int]bool) // Track which devices are aliases
	scanner := bufio.NewScanner(strings.NewReader(output))
	
	var currentDevice *types.Device
	var currentBackend string
	
	// Regular expressions for parsing
	backendRe := regexp.MustCompile(`^(HIP|OpenCL|CUDA) Info:`)
	deviceIDRe := regexp.MustCompile(`^Backend Device ID #(\d+)(?:\s+\(Alias:\s+#(\d+)\))?`)
	nameRe := regexp.MustCompile(`^\s*Name\.+:\s+(.+)`)
	typeRe := regexp.MustCompile(`^\s*Type\.+:\s+(.+)`)
	processorsRe := regexp.MustCompile(`^\s*Processor\(s\)\.+:\s+(\d+)`)
	clockRe := regexp.MustCompile(`^\s*Clock\.+:\s+(\d+)`)
	memoryTotalRe := regexp.MustCompile(`^\s*Memory\.Total\.+:\s+(\d+)\s+MB`)
	memoryFreeRe := regexp.MustCompile(`^\s*Memory\.Free\.+:\s+(\d+)\s+MB`)
	pciAddrRe := regexp.MustCompile(`^\s*PCI\.Addr\.(BDF|BDFe)\.+:\s+(.+)`)
	
	// First pass: identify which devices are aliases
	tempScanner := bufio.NewScanner(strings.NewReader(output))
	for tempScanner.Scan() {
		line := tempScanner.Text()
		if matches := deviceIDRe.FindStringSubmatch(line); matches != nil {
			// If this device has an alias notation, the device in parentheses IS the alias
			if len(matches) > 2 && matches[2] != "" {
				aliasID, _ := strconv.Atoi(matches[2])
				aliasMap[aliasID] = true
				debug.Info("Device #%s declares that device #%s is its alias", matches[1], matches[2])
			}
		}
	}
	
	// Second pass: parse all devices
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for backend section
		if matches := backendRe.FindStringSubmatch(line); matches != nil {
			currentBackend = matches[1]
			continue
		}
		
		// Check for device ID
		if matches := deviceIDRe.FindStringSubmatch(line); matches != nil {
			// Save previous device if any
			if currentDevice != nil {
				devices = append(devices, *currentDevice)
			}
			
			// Start new device
			deviceID, _ := strconv.Atoi(matches[1])
			currentDevice = &types.Device{
				ID:       deviceID,
				Backend:  currentBackend,
				Type:     "GPU", // Default to GPU type
				Enabled:  true, // Default to enabled
				IsAlias:  aliasMap[deviceID], // Mark if this device is an alias
			}
			continue
		}
		
		// Parse device properties
		if currentDevice != nil {
			if matches := nameRe.FindStringSubmatch(line); matches != nil {
				currentDevice.Name = strings.TrimSpace(matches[1])
			} else if matches := typeRe.FindStringSubmatch(line); matches != nil {
				currentDevice.Type = strings.TrimSpace(matches[1])
			} else if matches := processorsRe.FindStringSubmatch(line); matches != nil {
				currentDevice.Processors, _ = strconv.Atoi(matches[1])
			} else if matches := clockRe.FindStringSubmatch(line); matches != nil {
				currentDevice.Clock, _ = strconv.Atoi(matches[1])
			} else if matches := memoryTotalRe.FindStringSubmatch(line); matches != nil {
				currentDevice.MemoryTotal, _ = strconv.ParseInt(matches[1], 10, 64)
			} else if matches := memoryFreeRe.FindStringSubmatch(line); matches != nil {
				currentDevice.MemoryFree, _ = strconv.ParseInt(matches[1], 10, 64)
			} else if matches := pciAddrRe.FindStringSubmatch(line); matches != nil {
				currentDevice.PCIAddress = strings.TrimSpace(matches[2])
			}
		}
	}
	
	// Don't forget the last device
	if currentDevice != nil {
		devices = append(devices, *currentDevice)
	}
	
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices found in hashcat output")
	}
	
	return devices, nil
}

// filterAliases removes aliased devices from the list
func (d *HashcatDetector) filterAliases(devices []types.Device) []types.Device {
	var filtered []types.Device
	
	for _, device := range devices {
		if !device.IsAlias {
			filtered = append(filtered, device)
			debug.Info("Keeping primary device #%d: %s", device.ID, device.Name)
		} else {
			debug.Info("Filtering out alias device #%d: %s", device.ID, device.Name)
		}
	}
	
	return filtered
}

// BuildDeviceFlags builds the -d flag for hashcat based on enabled devices
func BuildDeviceFlags(devices []types.Device) string {
	var enabledIDs []string
	allEnabled := true
	
	for _, device := range devices {
		if device.Enabled {
			enabledIDs = append(enabledIDs, strconv.Itoa(device.ID))
		} else {
			allEnabled = false
		}
	}
	
	// If all devices are enabled, no need for -d flag
	if allEnabled || len(enabledIDs) == len(devices) {
		return ""
	}
	
	// If no devices are enabled, this is an error condition
	if len(enabledIDs) == 0 {
		return ""
	}
	
	// Return comma-separated list of enabled device IDs
	return strings.Join(enabledIDs, ",")
}