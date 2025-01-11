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

// AMDDetector implements GPU detection for AMD GPUs
type AMDDetector struct {
	initialized bool
}

// NewAMDDetector creates a new AMD GPU detector
func NewAMDDetector() *AMDDetector {
	return &AMDDetector{}
}

// Initialize prepares the AMD detector
func (a *AMDDetector) Initialize() error {
	if a.initialized {
		return nil
	}

	// Check for required tools
	tools := []string{"lspci", "rocm-smi"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			debug.Warning("Required tool not found: %s", tool)
		}
	}

	a.initialized = true
	return nil
}

// IsAvailable checks if AMD GPUs can be detected
func (a *AMDDetector) IsAvailable() bool {
	if !a.initialized {
		if err := a.Initialize(); err != nil {
			debug.Error("Failed to initialize AMD detector: %v", err)
			return false
		}
	}

	// Check for AMD GPUs using lspci
	cmd := exec.Command("lspci", "-d", "1002:", "-nn")
	output, err := cmd.Output()
	if err != nil {
		debug.Error("Failed to detect AMD GPUs: %v", err)
		return false
	}

	hasGPUs := strings.TrimSpace(string(output)) != ""
	if hasGPUs {
		debug.Info("AMD GPUs detected")
	} else {
		debug.Info("No AMD GPUs found")
	}
	return hasGPUs
}

// GetGPUs detects and returns information about AMD GPUs
func (a *AMDDetector) GetGPUs() ([]types.GPU, error) {
	if !a.initialized {
		if err := a.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize AMD detector: %w", err)
		}
	}

	cmd := exec.Command("lspci", "-d", "1002:", "-v")
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
				Vendor: types.VendorAMD,
			}

			// Extract model name
			parts := strings.Split(line, ": ")
			if len(parts) > 1 {
				currentGPU.Model = strings.TrimSpace(parts[1])
				debug.Debug("Found AMD GPU: %s", currentGPU.Model)
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

	debug.Info("Found %d AMD GPUs", len(gpus))
	return gpus, nil
}

// GetMetrics updates metrics for a specific GPU
func (a *AMDDetector) GetMetrics(gpu *types.GPU) error {
	if !a.initialized {
		if err := a.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize AMD detector: %w", err)
		}
	}

	// Try to get metrics from rocm-smi
	cmd := exec.Command("rocm-smi", "--showtemp", "--showuse", "--showpower")
	output, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Temperature") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if temp, err := strconv.ParseFloat(parts[1], 64); err == nil {
						gpu.Temperature = temp
						debug.Debug("GPU temperature: %.1f°C", gpu.Temperature)
					} else {
						debug.Warning("Failed to parse GPU temperature: %v", err)
					}
				}
			} else if strings.Contains(line, "GPU use") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if util, err := strconv.ParseFloat(parts[1], 64); err == nil {
						gpu.Utilization = util
						debug.Debug("GPU utilization: %.1f%%", gpu.Utilization)
					} else {
						debug.Warning("Failed to parse GPU utilization: %v", err)
					}
				}
			} else if strings.Contains(line, "Power") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if power, err := strconv.ParseFloat(parts[1], 64); err == nil {
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

	return nil
}

// UpdateMetrics refreshes metrics for multiple GPUs
func (a *AMDDetector) UpdateMetrics(gpus []types.GPU) error {
	var errs []error
	for idx := range gpus {
		if gpus[idx].Vendor != types.VendorAMD {
			continue
		}
		debug.Debug("Updating metrics for AMD GPU: %s", gpus[idx].Model)
		if err := a.GetMetrics(&gpus[idx]); err != nil {
			debug.Error("Failed to update metrics for GPU %s: %v", gpus[idx].Model, err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to update AMD GPU metrics: %v", errs)
	}
	return nil
}

// Cleanup releases resources
func (a *AMDDetector) Cleanup() error {
	debug.Info("Cleaning up AMD detector")
	a.initialized = false
	return nil
}
