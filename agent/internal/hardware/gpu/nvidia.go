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

// NVIDIADetector implements GPU detection for NVIDIA GPUs
type NVIDIADetector struct {
	initialized bool
}

// NewNVIDIADetector creates a new NVIDIA GPU detector
func NewNVIDIADetector() *NVIDIADetector {
	return &NVIDIADetector{}
}

// Initialize prepares the NVIDIA detector
func (n *NVIDIADetector) Initialize() error {
	if n.initialized {
		return nil
	}

	// Check for required tools
	tools := []string{"lspci", "nvidia-smi"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			debug.Warning("Required tool not found: %s", tool)
		}
	}

	n.initialized = true
	return nil
}

// IsAvailable checks if NVIDIA GPUs can be detected
func (n *NVIDIADetector) IsAvailable() bool {
	if !n.initialized {
		if err := n.Initialize(); err != nil {
			debug.Error("Failed to initialize NVIDIA detector: %v", err)
			return false
		}
	}

	// Check for NVIDIA GPUs using lspci
	cmd := exec.Command("lspci", "-d", "10de:", "-nn")
	output, err := cmd.Output()
	if err != nil {
		debug.Error("Failed to detect NVIDIA GPUs: %v", err)
		return false
	}

	hasGPUs := strings.TrimSpace(string(output)) != ""
	if hasGPUs {
		debug.Info("NVIDIA GPUs detected")
	} else {
		debug.Info("No NVIDIA GPUs found")
	}
	return hasGPUs
}

// GetGPUs detects and returns information about NVIDIA GPUs
func (n *NVIDIADetector) GetGPUs() ([]types.GPU, error) {
	if !n.initialized {
		if err := n.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize NVIDIA detector: %w", err)
		}
	}

	// Try nvidia-smi first
	cmd := exec.Command("nvidia-smi", "--query-gpu=gpu_name,memory.total,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		var gpus []types.GPU
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), ", ")
			if len(fields) >= 3 {
				gpu := types.GPU{
					Vendor: types.VendorNVIDIA,
					Model:  strings.TrimSpace(fields[0]),
					Driver: strings.TrimSpace(fields[2]),
				}

				// Convert memory from MiB to bytes
				if mem, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64); err == nil {
					gpu.Memory = mem * 1024 * 1024 // Convert MiB to bytes
					debug.Debug("GPU memory: %d MiB", mem)
				} else {
					debug.Warning("Failed to parse GPU memory: %v", err)
				}

				debug.Debug("Found NVIDIA GPU: %s", gpu.Model)
				gpus = append(gpus, gpu)
			}
		}
		debug.Info("Found %d NVIDIA GPUs using nvidia-smi", len(gpus))
		return gpus, nil
	}

	debug.Warning("nvidia-smi failed, falling back to lspci: %v", err)

	// Fall back to lspci if nvidia-smi fails
	cmd = exec.Command("lspci", "-d", "10de:", "-v")
	output, err = cmd.Output()
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
				Vendor: types.VendorNVIDIA,
			}

			// Extract model name
			parts := strings.Split(line, ": ")
			if len(parts) > 1 {
				currentGPU.Model = strings.TrimSpace(parts[1])
				debug.Debug("Found NVIDIA GPU: %s", currentGPU.Model)
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

	debug.Info("Found %d NVIDIA GPUs using lspci", len(gpus))
	return gpus, nil
}

// GetMetrics updates metrics for a specific GPU
func (n *NVIDIADetector) GetMetrics(gpu *types.GPU) error {
	if !n.initialized {
		if err := n.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize NVIDIA detector: %w", err)
		}
	}

	// Try to get metrics from nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=temperature.gpu,utilization.gpu,power.draw", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), ", ")
			if len(fields) >= 3 {
				// Parse temperature
				if temp, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64); err == nil {
					gpu.Temperature = temp
					debug.Debug("GPU temperature: %.1f°C", gpu.Temperature)
				} else {
					debug.Warning("Failed to parse GPU temperature: %v", err)
				}

				// Parse utilization
				if util, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
					gpu.Utilization = util
					debug.Debug("GPU utilization: %.1f%%", gpu.Utilization)
				} else {
					debug.Warning("Failed to parse GPU utilization: %v", err)
				}

				// Parse power usage
				if power, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
					gpu.PowerUsage = power
					debug.Debug("GPU power usage: %.1f W", gpu.PowerUsage)
				} else {
					debug.Warning("Failed to parse GPU power usage: %v", err)
				}
			}
		}
	} else {
		debug.Warning("Failed to read GPU metrics: %v", err)
	}

	return nil
}

// UpdateMetrics refreshes metrics for multiple GPUs
func (n *NVIDIADetector) UpdateMetrics(gpus []types.GPU) error {
	var errs []error
	for idx := range gpus {
		if gpus[idx].Vendor != types.VendorNVIDIA {
			continue
		}
		debug.Debug("Updating metrics for NVIDIA GPU: %s", gpus[idx].Model)
		if err := n.GetMetrics(&gpus[idx]); err != nil {
			debug.Error("Failed to update metrics for GPU %s: %v", gpus[idx].Model, err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to update NVIDIA GPU metrics: %v", errs)
	}
	return nil
}

// Cleanup releases resources
func (n *NVIDIADetector) Cleanup() error {
	debug.Info("Cleaning up NVIDIA detector")
	n.initialized = false
	return nil
}
