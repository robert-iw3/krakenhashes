package hardware

import (
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/gpu"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/os"
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// Monitor manages hardware monitoring
type Monitor struct {
	mu         sync.RWMutex
	info       types.Info
	gpuFactory *gpu.Factory
	osDetector types.OSSpecific
}

// NewMonitor creates a new hardware monitor
func NewMonitor() (*Monitor, error) {
	m := &Monitor{
		gpuFactory: gpu.NewDetectorFactory(),
		osDetector: os.NewLinuxDetector(),
	}

	// Initialize OS detector
	if err := m.osDetector.Initialize(); err != nil {
		debug.Error("Failed to initialize OS detector: %v", err)
		return nil, fmt.Errorf("failed to initialize OS detector: %w", err)
	}

	return m, nil
}

// GetInfo returns the current hardware information
func (m *Monitor) GetInfo() types.Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a deep copy to prevent concurrent modification
	info := types.Info{
		CPUs: make([]types.CPU, len(m.info.CPUs)),
		GPUs: make([]types.GPU, len(m.info.GPUs)),
	}

	copy(info.CPUs, m.info.CPUs)
	copy(info.GPUs, m.info.GPUs)

	return info
}

// UpdateInfo refreshes hardware information
func (m *Monitor) UpdateInfo() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update CPU information
	cpus, err := m.osDetector.GetCPUInfo()
	if err != nil {
		debug.Error("Failed to get CPU info: %v", err)
		return fmt.Errorf("failed to get CPU info: %w", err)
	}
	m.info.CPUs = cpus

	// Update GPU information
	gpus, err := m.gpuFactory.GetGPUs()
	if err != nil {
		debug.Error("Failed to get GPU info: %v", err)
		return fmt.Errorf("failed to get GPU info: %w", err)
	}
	m.info.GPUs = gpus

	return nil
}

// UpdateMetrics refreshes hardware metrics
func (m *Monitor) UpdateMetrics() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update CPU temperature
	temp, err := m.osDetector.GetCPUTemperature()
	if err != nil {
		debug.Error("Failed to get CPU temperature: %v", err)
		return fmt.Errorf("failed to get CPU temperature: %w", err)
	}

	for i := range m.info.CPUs {
		m.info.CPUs[i].Temperature = temp
	}

	// Update GPU metrics
	if err := m.gpuFactory.UpdateMetrics(m.info.GPUs); err != nil {
		debug.Error("Failed to update GPU metrics: %v", err)
		return fmt.Errorf("failed to update GPU metrics: %w", err)
	}

	return nil
}

// StartMonitoring begins periodic hardware monitoring
func (m *Monitor) StartMonitoring(interval time.Duration) {
	debug.Info("Starting hardware monitoring with interval %v", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := m.UpdateInfo(); err != nil {
				debug.Error("Failed to update hardware info: %v", err)
			}
			if err := m.UpdateMetrics(); err != nil {
				debug.Error("Failed to update hardware metrics: %v", err)
			}
		}
	}()
}

// Cleanup releases monitor resources
func (m *Monitor) Cleanup() error {
	debug.Info("Cleaning up hardware monitor")
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.gpuFactory.Cleanup(); err != nil {
		debug.Error("Failed to cleanup GPU factory: %v", err)
		return fmt.Errorf("failed to cleanup GPU factory: %w", err)
	}

	if err := m.osDetector.Cleanup(); err != nil {
		debug.Error("Failed to cleanup OS detector: %v", err)
		return fmt.Errorf("failed to cleanup OS detector: %w", err)
	}

	return nil
}
