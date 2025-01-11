package gpu

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ZerkerEOD/hashdom-agent/internal/hardware/types"
	"github.com/ZerkerEOD/hashdom-agent/pkg/debug"
)

// IntelDetector implements GPU detection for Intel GPUs
type IntelDetector struct {
	initialized bool
}

// NewIntelDetector creates a new Intel GPU detector
func NewIntelDetector() *IntelDetector {
	return &IntelDetector{}
}

// Initialize prepares the Intel detector
func (i *IntelDetector) Initialize() error {
	if i.initialized {
		return nil
	}

	// Check for required tools
	tools := []string{"lspci", "intel_gpu_top"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			debug.Warning("Required tool not found: %s", tool)
		}
	}

	i.initialized = true
	return nil
}

// IsAvailable checks if Intel GPUs can be detected
func (i *IntelDetector) IsAvailable() bool {
	if !i.initialized {
		if err := i.Initialize(); err != nil {
			debug.Error("Failed to initialize Intel detector: %v", err)
			return false
		}
	}

	// Check for Intel GPUs using lspci
	cmd := exec.Command("lspci", "-d", "8086:", "-nn")
	output, err := cmd.Output()
	if err != nil {
		debug.Error("Failed to detect Intel GPUs: %v", err)
		return false
	}

	hasGPUs := strings.TrimSpace(string(output)) != ""
	if hasGPUs {
		debug.Info("Intel GPUs detected")
	} else {
		debug.Info("No Intel GPUs found")
	}
	return hasGPUs
}

// GetGPUs detects and returns information about Intel GPUs
func (i *IntelDetector) GetGPUs() ([]types.GPU, error) {
	if !i.initialized {
		if err := i.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize Intel detector: %w", err)
		}
	}

	cmd := exec.Command("lspci", "-d", "8086:", "-v")
	output, err := cmd.Output()
	if err != nil {
		debug.Error("Failed to execute lspci: %v", err)
		return nil, fmt.Errorf("failed to execute lspci: %v", err)
	}

	var gpus []types.GPU
	var currentGPU *types.GPU

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "VGA") || strings.Contains(line, "Display") {
			if currentGPU != nil {
				gpus = append(gpus, *currentGPU)
			}
			currentGPU = &types.GPU{
				Vendor: types.VendorIntel,
			}

			// Extract model name
			parts := strings.Split(line, ": ")
			if len(parts) > 1 {
				currentGPU.Model = strings.TrimSpace(parts[1])
				debug.Debug("Found Intel GPU: %s", currentGPU.Model)
			}
		}

		if currentGPU == nil {
			continue
		}

		if strings.Contains(line, "Memory") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				memStr := strings.TrimSpace(parts[1])
				memStr = strings.TrimSuffix(memStr, " MB")
				if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
					currentGPU.Memory = mem * 1024 * 1024 // Convert MB to bytes
					debug.Debug("GPU memory: %d MB", mem)
				} else {
					debug.Warning("Failed to parse GPU memory: %v", err)
				}
			}
		} else if strings.Contains(line, "Kernel driver in use") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				currentGPU.Driver = strings.TrimSpace(parts[1])
				debug.Debug("GPU driver: %s", currentGPU.Driver)
			}
		}
	}

	if currentGPU != nil {
		gpus = append(gpus, *currentGPU)
	}

	debug.Info("Found %d Intel GPUs", len(gpus))
	return gpus, nil
}

// GetMetrics updates metrics for a specific GPU
func (i *IntelDetector) GetMetrics(gpu *types.GPU) error {
	if !i.initialized {
		if err := i.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize Intel detector: %w", err)
		}
	}

	// Try to get metrics from intel_gpu_top
	cmd := exec.Command("intel_gpu_top", "-J")
	output, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "busy") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					if util, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						gpu.Utilization = util
						debug.Debug("GPU utilization: %.1f%%", gpu.Utilization)
					} else {
						debug.Warning("Failed to parse GPU utilization: %v", err)
					}
				}
			} else if strings.Contains(line, "power") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					if power, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						gpu.PowerUsage = power
						debug.Debug("GPU power usage: %.1f W", gpu.PowerUsage)
					} else {
						debug.Warning("Failed to parse GPU power usage: %v", err)
					}
				}
			}
		}
	} else {
		debug.Warning("Failed to read GPU metrics: %v", err)
	}

	// Try to get temperature from sysfs
	cmd = exec.Command("cat", "/sys/class/drm/card0/device/hwmon/hwmon*/temp1_input")
	output, err = cmd.Output()
	if err == nil {
		if temp, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); err == nil {
			gpu.Temperature = temp / 1000.0 // Convert millidegrees to degrees
			debug.Debug("GPU temperature: %.1f°C", gpu.Temperature)
		} else {
			debug.Warning("Failed to parse GPU temperature: %v", err)
		}
	} else {
		debug.Warning("Failed to read GPU temperature: %v", err)
	}

	return nil
}

// UpdateMetrics refreshes metrics for multiple GPUs
func (i *IntelDetector) UpdateMetrics(gpus []types.GPU) error {
	var errs []error
	for idx := range gpus {
		if gpus[idx].Vendor != types.VendorIntel {
			continue
		}
		debug.Debug("Updating metrics for Intel GPU: %s", gpus[idx].Model)
		if err := i.GetMetrics(&gpus[idx]); err != nil {
			debug.Error("Failed to update metrics for GPU %s: %v", gpus[idx].Model, err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to update Intel GPU metrics: %v", errs)
	}
	return nil
}

// Cleanup releases resources
func (i *IntelDetector) Cleanup() error {
	debug.Info("Cleaning up Intel detector")
	i.initialized = false
	return nil
}
