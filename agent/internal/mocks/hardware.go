package mocks

import (
	"strings"
	"sync"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
)

// MockHardwareMonitor implements a mock hardware monitor for testing
type MockHardwareMonitor struct {
	mu sync.RWMutex
	
	// Control behavior
	DetectDevicesFunc          func() (*types.DeviceDetectionResult, error)
	GetDevicesFunc             func() []types.Device
	UpdateDeviceStatusFunc     func(deviceID int, enabled bool) error
	GetEnabledDeviceFlagsFunc  func() string
	HasEnabledDevicesFunc      func() bool
	CleanupFunc                func() error
	
	// Default data
	Devices []types.Device
	
	// Call tracking
	DetectDevicesCalls     int
	GetDevicesCalls        int
	UpdateDeviceStatusCalls int
	CleanupCalls           int
}

// NewMockHardwareMonitor creates a new mock hardware monitor
func NewMockHardwareMonitor() *MockHardwareMonitor {
	return &MockHardwareMonitor{
		Devices: []types.Device{},
	}
}

// DetectDevices implements hardware.Monitor
func (m *MockHardwareMonitor) DetectDevices() (*types.DeviceDetectionResult, error) {
	m.mu.Lock()
	m.DetectDevicesCalls++
	m.mu.Unlock()
	
	if m.DetectDevicesFunc != nil {
		return m.DetectDevicesFunc()
	}
	
	// Default implementation
	return &types.DeviceDetectionResult{
		Devices: m.GetDevices(),
		Error:   "",
	}, nil
}

// GetDevices implements hardware.Monitor
func (m *MockHardwareMonitor) GetDevices() []types.Device {
	m.mu.Lock()
	m.GetDevicesCalls++
	m.mu.Unlock()
	
	if m.GetDevicesFunc != nil {
		return m.GetDevicesFunc()
	}
	
	// Return copy of devices
	m.mu.RLock()
	defer m.mu.RUnlock()
	devices := make([]types.Device, len(m.Devices))
	copy(devices, m.Devices)
	return devices
}

// UpdateDeviceStatus implements hardware.Monitor
func (m *MockHardwareMonitor) UpdateDeviceStatus(deviceID int, enabled bool) error {
	m.mu.Lock()
	m.UpdateDeviceStatusCalls++
	m.mu.Unlock()
	
	if m.UpdateDeviceStatusFunc != nil {
		return m.UpdateDeviceStatusFunc(deviceID, enabled)
	}
	
	// Default implementation
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Devices {
		if m.Devices[i].ID == deviceID {
			m.Devices[i].Enabled = enabled
			return nil
		}
	}
	
	return nil
}

// GetEnabledDeviceFlags implements hardware.Monitor
func (m *MockHardwareMonitor) GetEnabledDeviceFlags() string {
	if m.GetEnabledDeviceFlagsFunc != nil {
		return m.GetEnabledDeviceFlagsFunc()
	}
	
	// Default implementation
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var enabled []string
	for _, d := range m.Devices {
		if d.Enabled {
			enabled = append(enabled, string(rune(d.ID)))
		}
	}
	
	if len(enabled) == 0 {
		return ""
	}
	
	return strings.Join(enabled, ",")
}

// HasEnabledDevices implements hardware.Monitor
func (m *MockHardwareMonitor) HasEnabledDevices() bool {
	if m.HasEnabledDevicesFunc != nil {
		return m.HasEnabledDevicesFunc()
	}
	
	// Default implementation
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, d := range m.Devices {
		if d.Enabled {
			return true
		}
	}
	
	return false
}

// Cleanup implements hardware.Monitor
func (m *MockHardwareMonitor) Cleanup() error {
	m.mu.Lock()
	m.CleanupCalls++
	m.mu.Unlock()
	
	if m.CleanupFunc != nil {
		return m.CleanupFunc()
	}
	
	return nil
}

// SetDevices is a helper method to set the devices for testing
func (m *MockHardwareMonitor) SetDevices(devices []types.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Devices = devices
}

// AddDevice is a helper method to add a device for testing
func (m *MockHardwareMonitor) AddDevice(device types.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Devices = append(m.Devices, device)
}