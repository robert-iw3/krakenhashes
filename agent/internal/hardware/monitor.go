package hardware

import (
	"fmt"
	"sync"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// Monitor manages hardware monitoring
type Monitor struct {
	mu             sync.RWMutex
	devices        []types.Device
	hashcatDetector *HashcatDetector
	dataDirectory   string
}

// NewMonitor creates a new hardware monitor
func NewMonitor(dataDirectory string) (*Monitor, error) {
	m := &Monitor{
		hashcatDetector: NewHashcatDetector(dataDirectory),
		dataDirectory:   dataDirectory,
		devices:         []types.Device{},
	}

	return m, nil
}

// Cleanup releases monitor resources
func (m *Monitor) Cleanup() error {
	debug.Info("Cleaning up hardware monitor")
	m.mu.Lock()
	defer m.mu.Unlock()

	// Nothing to cleanup anymore
	return nil
}

// DetectDevices uses hashcat to detect available compute devices
func (m *Monitor) DetectDevices() (*types.DeviceDetectionResult, error) {
	result, err := m.hashcatDetector.DetectDevices()
	if err != nil {
		return nil, err
	}
	
	// Store devices in monitor
	m.mu.Lock()
	m.devices = result.Devices
	m.mu.Unlock()
	
	return result, nil
}

// HasBinary checks if any hashcat binary is available
func (m *Monitor) HasBinary() bool {
	return m.hashcatDetector.HasHashcatBinary()
}

// GetDevices returns the currently detected devices
func (m *Monitor) GetDevices() []types.Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent concurrent modification
	devices := make([]types.Device, len(m.devices))
	copy(devices, m.devices)
	
	return devices
}

// UpdateDeviceStatus updates the enabled status of a device
func (m *Monitor) UpdateDeviceStatus(deviceID int, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	found := false
	for i := range m.devices {
		if m.devices[i].ID == deviceID {
			m.devices[i].Enabled = enabled
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("device with ID %d not found", deviceID)
	}
	
	return nil
}

// GetEnabledDeviceFlags returns the -d flag value for hashcat based on enabled devices
func (m *Monitor) GetEnabledDeviceFlags() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return BuildDeviceFlags(m.devices)
}

// HasEnabledDevices returns true if at least one device is enabled
func (m *Monitor) HasEnabledDevices() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, device := range m.devices {
		if device.Enabled {
			return true
		}
	}
	
	return false
}
