package types

// Device represents a compute device detected by hashcat
type Device struct {
	ID       int    `json:"device_id"`
	Name     string `json:"device_name"`
	Type     string `json:"device_type"` // "GPU" or "CPU"
	Enabled  bool   `json:"enabled"`
	
	// Additional properties from hashcat output
	Processors  int    `json:"processors,omitempty"`
	Clock       int    `json:"clock,omitempty"`       // MHz
	MemoryTotal int64  `json:"memory_total,omitempty"` // MB
	MemoryFree  int64  `json:"memory_free,omitempty"`  // MB
	PCIAddress  string `json:"pci_address,omitempty"`
	
	// Backend information
	Backend     string `json:"backend,omitempty"`      // "HIP", "OpenCL", "CUDA", etc.
	IsAlias     bool   `json:"is_alias,omitempty"`
	AliasOf     int    `json:"alias_of,omitempty"`     // Device ID this is an alias of
}

// DeviceDetectionResult represents the result of device detection
type DeviceDetectionResult struct {
	Devices []Device `json:"devices"`
	Error   string   `json:"error,omitempty"`
}

// DeviceUpdate represents a device update request
type DeviceUpdate struct {
	DeviceID int  `json:"device_id"`
	Enabled  bool `json:"enabled"`
}