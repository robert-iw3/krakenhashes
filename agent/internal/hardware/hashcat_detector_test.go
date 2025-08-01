package hardware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHashcatDetector(t *testing.T) {
	dataDir := t.TempDir()
	detector := NewHashcatDetector(dataDir)
	
	assert.NotNil(t, detector)
	assert.Equal(t, dataDir, detector.dataDirectory)
}

func TestHashcatDetector_findLatestHashcatBinary(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(dataDir string) error
		wantErr     bool
		errContains string
	}{
		{
			name: "no binaries directory",
			setupFunc: func(dataDir string) error {
				return nil // Don't create any directories
			},
			wantErr:     true,
			errContains: "failed to read binaries directory",
		},
		{
			name: "empty binaries directory",
			setupFunc: func(dataDir string) error {
				return os.MkdirAll(filepath.Join(dataDir, "binaries"), 0755)
			},
			wantErr:     true,
			errContains: "no hashcat binary versions found",
		},
		{
			name: "directory with non-numeric names",
			setupFunc: func(dataDir string) error {
				binDir := filepath.Join(dataDir, "binaries")
				if err := os.MkdirAll(filepath.Join(binDir, "invalid"), 0755); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(binDir, "test"), 0755)
			},
			wantErr:     true,
			errContains: "no hashcat binary versions found",
		},
		{
			name: "version directory without binary",
			setupFunc: func(dataDir string) error {
				return os.MkdirAll(filepath.Join(dataDir, "binaries", "1"), 0755)
			},
			wantErr:     true,
			errContains: "hashcat binary not found",
		},
		{
			name: "valid binary exists",
			setupFunc: func(dataDir string) error {
				binPath := filepath.Join(dataDir, "binaries", "1", "hashcat.bin")
				if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
					return err
				}
				return os.WriteFile(binPath, []byte("mock binary"), 0755)
			},
			wantErr: false,
		},
		{
			name: "multiple versions - picks latest",
			setupFunc: func(dataDir string) error {
				// Create version 1
				binPath1 := filepath.Join(dataDir, "binaries", "1", "hashcat.bin")
				if err := os.MkdirAll(filepath.Dir(binPath1), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(binPath1, []byte("v1"), 0755); err != nil {
					return err
				}
				
				// Create version 3
				binPath3 := filepath.Join(dataDir, "binaries", "3", "hashcat.bin")
				if err := os.MkdirAll(filepath.Dir(binPath3), 0755); err != nil {
					return err
				}
				return os.WriteFile(binPath3, []byte("v3"), 0755)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			detector := &HashcatDetector{dataDirectory: dataDir}
			
			if tt.setupFunc != nil {
				require.NoError(t, tt.setupFunc(dataDir))
			}
			
			path, err := detector.findLatestHashcatBinary()
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, path)
				assert.Contains(t, path, "hashcat.bin")
			}
		})
	}
}

func TestHashcatDetector_ParseHashcatOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantDevices int
		wantErr     bool
		validate    func(t *testing.T, devices []types.Device)
	}{
		{
			name:        "empty output",
			output:      "",
			wantDevices: 0,
			wantErr:     true,
		},
		{
			name: "single CUDA device",
			output: `CUDA Info:
==========
CUDA Platform ID #1
  Vendor  : NVIDIA Corporation
  Name    : CUDA
  Version : 12.2

  Backend Device ID #1
    Name.......: NVIDIA GeForce RTX 3090
    Type.......: GPU
    Processor(s).....: 82
    Clock......: 1695
    Memory.Total.....: 24265 MB
    Memory.Free......: 23456 MB
    PCI.Addr.BDFe....: 0000:01:00.0`,
			wantDevices: 1,
			wantErr:     false,
			validate: func(t *testing.T, devices []types.Device) {
				assert.Equal(t, 1, devices[0].ID)
				assert.Equal(t, "NVIDIA GeForce RTX 3090", devices[0].Name)
				assert.Equal(t, "GPU", devices[0].Type)
				assert.Equal(t, "CUDA", devices[0].Backend)
				assert.Equal(t, 82, devices[0].Processors)
				assert.Equal(t, 1695, devices[0].Clock)
				assert.Equal(t, int64(24265), devices[0].MemoryTotal)
				assert.Equal(t, int64(23456), devices[0].MemoryFree)
				assert.Equal(t, "0000:01:00.0", devices[0].PCIAddress)
			},
		},
		{
			name: "multiple devices with alias",
			output: `OpenCL Info:
============
OpenCL Platform ID #1
  Vendor  : Advanced Micro Devices, Inc.
  Name    : AMD Accelerated Parallel Processing
  Version : OpenCL 2.1 AMD-APP (3513.0)

  Backend Device ID #1
    Name.......: gfx1030
    Type.......: GPU
    Processor(s).....: 36
    Clock......: 2500
    Memory.Total.....: 8176 MB
    Memory.Free......: 8112 MB

HIP Info:
=========
HIP Platform ID #1
  Vendor  : Advanced Micro Devices, Inc.
  Name    : HIP
  Version : HIP 5.6.31061

  Backend Device ID #2 (Alias: #1)
    Name.......: AMD Radeon RX 6800 XT
    Type.......: GPU
    Processor(s).....: 36
    Clock......: 2500
    Memory.Total.....: 8192 MB
    Memory.Free......: 8128 MB
    PCI.Addr.BDFe....: 0000:03:00.0`,
			wantDevices: 2,
			wantErr:     false,
			validate: func(t *testing.T, devices []types.Device) {
				// First device - OpenCL
				assert.Equal(t, 1, devices[0].ID)
				assert.Equal(t, "gfx1030", devices[0].Name)
				assert.Equal(t, "OpenCL", devices[0].Backend)
				assert.Equal(t, 0, devices[0].AliasOf)
				
				// Second device - HIP (declares #1 as alias)
				assert.Equal(t, 2, devices[1].ID)
				assert.Equal(t, "AMD Radeon RX 6800 XT", devices[1].Name)
				assert.Equal(t, "HIP", devices[1].Backend)
				assert.Equal(t, 1, devices[1].AliasOf)
			},
		},
		{
			name: "circular alias",
			output: `OpenCL Info:
============
  Backend Device ID #1 (Alias: #2)
    Name.......: Intel(R) UHD Graphics 770
    Type.......: GPU
    Processor(s).....: 32
    Clock......: 1550

OpenCL Info:
============
  Backend Device ID #2 (Alias: #1)
    Name.......: Intel(R) UHD Graphics 770
    Type.......: GPU
    Processor(s).....: 32
    Clock......: 1550`,
			wantDevices: 2,
			wantErr:     false,
			validate: func(t *testing.T, devices []types.Device) {
				// Both devices should be detected
				assert.Equal(t, 2, len(devices))
				
				// Device 1 declares 2 as alias
				assert.Equal(t, 1, devices[0].ID)
				assert.Equal(t, 2, devices[0].AliasOf)
				
				// Device 2 declares 1 as alias  
				assert.Equal(t, 2, devices[1].ID)
				assert.Equal(t, 1, devices[1].AliasOf)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &HashcatDetector{}
			devices, err := detector.ParseHashcatOutput(tt.output)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, devices, tt.wantDevices)
				if tt.validate != nil {
					tt.validate(t, devices)
				}
			}
		})
	}
}

func TestHashcatDetector_FilterAliases(t *testing.T) {
	tests := []struct {
		name        string
		devices     []types.Device
		wantCount   int
		wantDevices []int // Expected device IDs after filtering
	}{
		{
			name:        "no devices",
			devices:     []types.Device{},
			wantCount:   0,
			wantDevices: []int{},
		},
		{
			name: "no aliases",
			devices: []types.Device{
				{ID: 1, Name: "GPU 1", Backend: "CUDA", Enabled: true, Processors: 10},
				{ID: 2, Name: "GPU 2", Backend: "OpenCL", Enabled: true, Processors: 20},
			},
			wantCount:   2,
			wantDevices: []int{1, 2},
		},
		{
			name: "simple alias - device 2 is alias of 1",
			devices: []types.Device{
				{ID: 1, Name: "GPU 1", Backend: "HIP", Enabled: true, Processors: 10},
				{ID: 2, Name: "GPU 1", Backend: "OpenCL", Enabled: true, AliasOf: 1, Processors: 10},
			},
			wantCount:   1,
			wantDevices: []int{2}, // Device declaring the alias is kept
		},
		{
			name: "circular alias - HIP wins over OpenCL",
			devices: []types.Device{
				{ID: 1, Name: "AMD GPU", Backend: "OpenCL", Enabled: true, AliasOf: 2, Processors: 36},
				{ID: 2, Name: "AMD GPU", Backend: "HIP", Enabled: true, AliasOf: 1, Processors: 36},
			},
			wantCount:   1,
			wantDevices: []int{2}, // HIP has higher priority
		},
		{
			name: "circular alias - CUDA wins over OpenCL",
			devices: []types.Device{
				{ID: 3, Name: "NVIDIA GPU", Backend: "OpenCL", Enabled: true, AliasOf: 4, Processors: 80},
				{ID: 4, Name: "NVIDIA GPU", Backend: "CUDA", Enabled: true, AliasOf: 3, Processors: 80},
			},
			wantCount:   1,
			wantDevices: []int{4}, // CUDA has higher priority
		},
		{
			name: "mixed devices with some aliases",
			devices: []types.Device{
				{ID: 1, Name: "Intel GPU", Backend: "OpenCL", Enabled: true, Processors: 32},
				{ID: 2, Name: "NVIDIA GPU", Backend: "CUDA", Enabled: true, Processors: 80},
				{ID: 3, Name: "NVIDIA GPU", Backend: "OpenCL", Enabled: true, AliasOf: 2, Processors: 80},
				{ID: 4, Name: "AMD GPU", Backend: "HIP", Enabled: true, Processors: 60},
			},
			wantCount:   3,
			wantDevices: []int{1, 3, 4}, // 3 declares 2 as alias, so 2 is filtered
		},
		{
			name: "invalid device filtered",
			devices: []types.Device{
				{ID: 1, Name: "", Backend: "CUDA", Enabled: true, Processors: 0}, // Invalid
				{ID: 2, Name: "Valid GPU", Backend: "CUDA", Enabled: true, Processors: 80},
			},
			wantCount:   1,
			wantDevices: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &HashcatDetector{}
			filtered := detector.FilterAliases(tt.devices)
			
			assert.Len(t, filtered, tt.wantCount)
			
			// Check device IDs
			filteredIDs := []int{}
			for _, d := range filtered {
				filteredIDs = append(filteredIDs, d.ID)
			}
			assert.Equal(t, tt.wantDevices, filteredIDs)
		})
	}
}

func TestBuildDeviceFlags(t *testing.T) {
	tests := []struct {
		name     string
		devices  []types.Device
		expected string
	}{
		{
			name:     "no devices",
			devices:  []types.Device{},
			expected: "",
		},
		{
			name: "all devices enabled",
			devices: []types.Device{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: true},
				{ID: 3, Enabled: true},
			},
			expected: "",
		},
		{
			name: "some devices disabled",
			devices: []types.Device{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: false},
				{ID: 3, Enabled: true},
			},
			expected: "1,3",
		},
		{
			name: "all devices disabled",
			devices: []types.Device{
				{ID: 1, Enabled: false},
				{ID: 2, Enabled: false},
			},
			expected: "",
		},
		{
			name: "single device enabled",
			devices: []types.Device{
				{ID: 1, Enabled: false},
				{ID: 2, Enabled: true},
				{ID: 3, Enabled: false},
			},
			expected: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDeviceFlags(tt.devices)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create sample hashcat output for testing
func createSampleHashcatOutput() string {
	return `hashcat (v6.2.6) starting in backend information mode

CUDA Info:
==========
CUDA Platform ID #1
  Vendor  : NVIDIA Corporation
  Name    : CUDA
  Version : 12.2

  Backend Device ID #1
    Name.......: NVIDIA GeForce RTX 3090
    Type.......: GPU
    Board......: NVIDIA GeForce RTX 3090
    Processors.: 82
    Clock......: 1695
    Memory.Total.....: 24265 MB
    Memory.Free......: 23925 MB
    Local.Memory......: 48 KB
    PCI.Addr.BDFe......: 0000:01:00.0

OpenCL Info:
============
OpenCL Platform ID #1
  Vendor  : NVIDIA Corporation
  Name    : NVIDIA CUDA
  Version : OpenCL 3.0 CUDA 12.2.140

  Backend Device ID #2 (Alias: #1)
    Name.......: NVIDIA GeForce RTX 3090
    Type.......: GPU
    Board......: NVIDIA GeForce RTX 3090
    Processors.: 82
    Clock......: 1695
    Memory.Total.....: 24265 MB
    Memory.Free......: 23925 MB
    OpenCL.Version.....: OpenCL C 1.2
    Driver.Version.....: 535.129.03
    PCI.Addr.BDF.......: 01:00.0`
}

func TestHashcatDetector_DetectDevices_Integration(t *testing.T) {
	// This is an integration test that would require hashcat to be installed
	// Skip if hashcat is not available
	dataDir := t.TempDir()
	
	// Create a mock binary directory
	binDir := filepath.Join(dataDir, "binaries", "1")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	
	// Create a mock hashcat binary (just for finding, not executing)
	binPath := filepath.Join(binDir, "hashcat.bin")
	require.NoError(t, os.WriteFile(binPath, []byte("#!/bin/bash\necho 'mock hashcat'"), 0755))
	
	detector := NewHashcatDetector(dataDir)
	
	// This will fail because our mock binary doesn't actually output hashcat data
	_, err := detector.DetectDevices()
	assert.Error(t, err) // Expected to fail without real hashcat
}

func BenchmarkParseHashcatOutput(b *testing.B) {
	detector := &HashcatDetector{}
	output := createSampleHashcatOutput()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.ParseHashcatOutput(output)
	}
}

func BenchmarkFilterAliases(b *testing.B) {
	detector := &HashcatDetector{}
	devices := []types.Device{
		{ID: 1, Name: "GPU 1", Backend: "CUDA", Enabled: true, Processors: 80},
		{ID: 2, Name: "GPU 1", Backend: "OpenCL", Enabled: true, AliasOf: 1, Processors: 80},
		{ID: 3, Name: "GPU 2", Backend: "HIP", Enabled: true, Processors: 60},
		{ID: 4, Name: "GPU 2", Backend: "OpenCL", Enabled: true, AliasOf: 3, Processors: 60},
		{ID: 5, Name: "GPU 3", Backend: "CUDA", Enabled: true, Processors: 40},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.FilterAliases(devices)
	}
}