package hardware

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ZerkerEOD/hashdom-agent/pkg/debug"
)

// Dependency represents a system dependency
type Dependency struct {
	Name        string   // Name of the dependency
	Package     string   // Package name to install
	Command     string   // Command to check if installed
	Optional    bool     // Whether the dependency is optional
	Description string   // Description of what this dependency is for
	Vendors     []string // GPU vendors that require this dependency
}

var (
	// CommonDependencies lists dependencies required for basic functionality
	CommonDependencies = []Dependency{
		{
			Name:        "lspci",
			Package:     "pciutils",
			Command:     "lspci",
			Optional:    false,
			Description: "PCI device detection",
		},
		{
			Name:        "lscpu",
			Package:     "util-linux",
			Command:     "lscpu",
			Optional:    false,
			Description: "CPU information",
		},
		{
			Name:        "sensors",
			Package:     "lm-sensors",
			Command:     "sensors",
			Optional:    true,
			Description: "Temperature monitoring",
		},
	}

	// GPUDependencies lists GPU-specific dependencies
	GPUDependencies = []Dependency{
		{
			Name:        "NVIDIA Driver",
			Package:     "nvidia-driver",
			Command:     "nvidia-smi",
			Optional:    true,
			Description: "NVIDIA GPU support",
			Vendors:     []string{"NVIDIA"},
		},
		{
			Name:        "ROCm",
			Package:     "rocm-smi",
			Command:     "rocm-smi",
			Optional:    true,
			Description: "AMD GPU support (ROCm)",
			Vendors:     []string{"AMD"},
		},
		{
			Name:        "AMDGPU-PRO",
			Package:     "amdgpu-pro",
			Command:     "amdgpu-pro-top",
			Optional:    true,
			Description: "AMD GPU support (AMDGPU-PRO)",
			Vendors:     []string{"AMD"},
		},
		{
			Name:        "Intel GPU Tools",
			Package:     "intel-gpu-tools",
			Command:     "intel_gpu_top",
			Optional:    true,
			Description: "Intel GPU support",
			Vendors:     []string{"Intel"},
		},
	}
)

// CheckDependencies checks for required dependencies and returns installation instructions
func CheckDependencies() ([]string, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	var missing []string
	var instructions []string

	// Check common dependencies
	for _, dep := range CommonDependencies {
		if _, err := exec.LookPath(dep.Command); err != nil {
			debug.Warning("Missing %s dependency: %s", dep.Optional, dep.Name)
			if !dep.Optional {
				missing = append(missing, dep.Name)
			}
			instructions = append(instructions, fmt.Sprintf("# Install %s (%s)", dep.Name, dep.Description))
			instructions = append(instructions, fmt.Sprintf("sudo apt-get install %s", dep.Package))
		}
	}

	// Check GPU dependencies based on detected hardware
	gpuVendors, err := detectGPUVendors()
	if err != nil {
		debug.Warning("Failed to detect GPU vendors: %v", err)
	}

	for _, dep := range GPUDependencies {
		if shouldCheckDependency(dep, gpuVendors) {
			if _, err := exec.LookPath(dep.Command); err != nil {
				debug.Warning("Missing GPU dependency: %s", dep.Name)
				instructions = append(instructions, fmt.Sprintf("# Install %s (%s)", dep.Name, dep.Description))
				instructions = append(instructions, fmt.Sprintf("sudo apt-get install %s", dep.Package))
			}
		}
	}

	if len(missing) > 0 {
		return instructions, fmt.Errorf("missing required dependencies: %s", strings.Join(missing, ", "))
	}

	return instructions, nil
}

// detectGPUVendors uses lspci to detect installed GPU vendors
func detectGPUVendors() ([]string, error) {
	var vendors []string

	// Check for NVIDIA GPUs
	cmd := exec.Command("lspci", "-d", "10de:", "-nn")
	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		vendors = append(vendors, "NVIDIA")
	}

	// Check for AMD GPUs
	cmd = exec.Command("lspci", "-d", "1002:", "-nn")
	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		vendors = append(vendors, "AMD")
	}

	// Check for Intel GPUs
	cmd = exec.Command("lspci", "-d", "8086:", "-nn")
	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		if strings.Contains(string(output), "VGA") || strings.Contains(string(output), "Display") {
			vendors = append(vendors, "Intel")
		}
	}

	return vendors, nil
}

// shouldCheckDependency determines if a dependency should be checked based on detected vendors
func shouldCheckDependency(dep Dependency, vendors []string) bool {
	if len(dep.Vendors) == 0 {
		return true // Common dependency
	}

	for _, vendor := range dep.Vendors {
		for _, detected := range vendors {
			if vendor == detected {
				return true
			}
		}
	}

	return false
}
