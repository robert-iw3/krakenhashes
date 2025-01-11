package os

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ZerkerEOD/hashdom/agent/internal/hardware/types"
	"github.com/ZerkerEOD/hashdom/agent/pkg/debug"
)

// LinuxDetector implements OS-specific hardware detection for Linux
type LinuxDetector struct {
	initialized bool
}

// NewLinuxDetector creates a new Linux hardware detector
func NewLinuxDetector() *LinuxDetector {
	return &LinuxDetector{}
}

// Initialize prepares the Linux detector
func (l *LinuxDetector) Initialize() error {
	if l.initialized {
		return nil
	}

	// Check for required tools
	tools := []string{"lspci", "lscpu", "sensors"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			debug.Warning("Required tool not found: %s", tool)
		}
	}

	l.initialized = true
	return nil
}

// GetGPUVendors detects installed GPU vendors using lspci
func (l *LinuxDetector) GetGPUVendors() ([]types.GPUVendor, error) {
	if !l.initialized {
		if err := l.Initialize(); err != nil {
			return nil, err
		}
	}

	cmd := exec.Command("lspci", "-nn")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lspci: %v", err)
	}

	vendors := make([]types.GPUVendor, 0)
	vendorMap := make(map[types.GPUVendor]bool)

	// Regular expressions for GPU vendors
	nvidiaRE := regexp.MustCompile(`(?i)NVIDIA`)
	amdRE := regexp.MustCompile(`(?i)AMD|ATI`)
	intelRE := regexp.MustCompile(`(?i)Intel.*Graphics`)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case nvidiaRE.MatchString(line):
			vendorMap[types.VendorNVIDIA] = true
		case amdRE.MatchString(line):
			vendorMap[types.VendorAMD] = true
		case intelRE.MatchString(line):
			vendorMap[types.VendorIntel] = true
		}
	}

	// Convert map to slice
	for vendor := range vendorMap {
		vendors = append(vendors, vendor)
	}

	return vendors, nil
}

// GetCPUInfo retrieves detailed CPU information
func (l *LinuxDetector) GetCPUInfo() ([]types.CPU, error) {
	if !l.initialized {
		if err := l.Initialize(); err != nil {
			return nil, err
		}
	}

	cmd := exec.Command("lscpu")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lscpu: %v", err)
	}

	var cpu types.CPU
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Model name":
			cpu.Model = value
		case "CPU(s)":
			if threads, err := strconv.Atoi(value); err == nil {
				cpu.Threads = threads
			}
		case "Core(s) per socket":
			if cores, err := strconv.Atoi(value); err == nil {
				cpu.Cores = cores
			}
		case "CPU MHz":
			if freq, err := strconv.ParseFloat(value, 64); err == nil {
				cpu.Frequency = freq
			}
		}
	}

	return []types.CPU{cpu}, nil
}

// GetCPUTemperature reads CPU temperature using lm-sensors
func (l *LinuxDetector) GetCPUTemperature() (float64, error) {
	if !l.initialized {
		if err := l.Initialize(); err != nil {
			return 0, err
		}
	}

	cmd := exec.Command("sensors", "-u")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute sensors: %v", err)
	}

	// Parse sensors output for CPU temperature
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var temp float64
	tempRE := regexp.MustCompile(`temp1_input:\s+(\d+\.\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := tempRE.FindStringSubmatch(line); len(matches) == 2 {
			if t, err := strconv.ParseFloat(matches[1], 64); err == nil {
				temp = t
				break
			}
		}
	}

	return temp, nil
}

// Cleanup performs any necessary cleanup
func (l *LinuxDetector) Cleanup() error {
	l.initialized = false
	return nil
}
