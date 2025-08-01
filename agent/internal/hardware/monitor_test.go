package hardware

import (
	"testing"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	dataDir := t.TempDir()
	
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, dataDir, monitor.dataDirectory)
	assert.NotNil(t, monitor.hashcatDetector)
	assert.Empty(t, monitor.devices)
}

func TestMonitor_Cleanup(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	err = monitor.Cleanup()
	// Should not error even if nothing to cleanup
	assert.NoError(t, err)
}

func TestMonitor_DeviceManagement(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	// Initially no devices
	devices := monitor.GetDevices()
	assert.Empty(t, devices)

	// Detect devices (may fail if hashcat not available)
	result, err := monitor.DetectDevices()
	if err != nil {
		t.Logf("Device detection failed (expected without hashcat): %v", err)
		return
	}

	assert.NotNil(t, result)
	
	// Get devices after detection
	devices = monitor.GetDevices()
	assert.Len(t, devices, len(result.Devices))
}

func TestMonitor_UpdateDeviceStatus(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	// Manually add some test devices
	monitor.mu.Lock()
	monitor.devices = []types.Device{
		{ID: 1, Name: "Test GPU 1", Enabled: true},
		{ID: 2, Name: "Test GPU 2", Enabled: false},
	}
	monitor.mu.Unlock()

	// Test enabling a device
	err = monitor.UpdateDeviceStatus(2, true)
	assert.NoError(t, err)

	devices := monitor.GetDevices()
	for _, d := range devices {
		if d.ID == 2 {
			assert.True(t, d.Enabled)
		}
	}

	// Test disabling a device
	err = monitor.UpdateDeviceStatus(1, false)
	assert.NoError(t, err)

	devices = monitor.GetDevices()
	for _, d := range devices {
		if d.ID == 1 {
			assert.False(t, d.Enabled)
		}
	}

	// Test updating non-existent device
	err = monitor.UpdateDeviceStatus(99, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device with ID")
}

func TestMonitor_GetEnabledDeviceFlags(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	// Test with no devices
	flags := monitor.GetEnabledDeviceFlags()
	assert.Equal(t, "", flags)

	// Add test devices
	monitor.mu.Lock()
	monitor.devices = []types.Device{
		{ID: 1, Enabled: true},
		{ID: 2, Enabled: false},
		{ID: 3, Enabled: true},
		{ID: 4, Enabled: true},
	}
	monitor.mu.Unlock()

	flags = monitor.GetEnabledDeviceFlags()
	assert.Equal(t, "1,3,4", flags)
}

func TestMonitor_HasEnabledDevices(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	// No devices initially
	assert.False(t, monitor.HasEnabledDevices())

	// Add disabled devices
	monitor.mu.Lock()
	monitor.devices = []types.Device{
		{ID: 1, Enabled: false},
		{ID: 2, Enabled: false},
	}
	monitor.mu.Unlock()
	assert.False(t, monitor.HasEnabledDevices())

	// Enable one device
	monitor.mu.Lock()
	monitor.devices[0].Enabled = true
	monitor.mu.Unlock()
	assert.True(t, monitor.HasEnabledDevices())
}

func TestMonitor_ConcurrentAccess(t *testing.T) {
	dataDir := t.TempDir()
	monitor, err := NewMonitor(dataDir)
	require.NoError(t, err)

	// Add some test data
	monitor.mu.Lock()
	monitor.devices = []types.Device{
		{ID: 1, Enabled: true},
		{ID: 2, Enabled: false},
	}
	monitor.mu.Unlock()

	// Run concurrent operations
	done := make(chan bool, 3)
	
	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.GetDevices()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.HasEnabledDevices()
		}
		done <- true
	}()

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.UpdateDeviceStatus(1, i%2 == 0)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
	// Test passes if no race conditions occur
}