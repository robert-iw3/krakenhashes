package types

// GPUVendor represents a GPU manufacturer
type GPUVendor string

const (
	VendorNVIDIA GPUVendor = "NVIDIA"
	VendorAMD    GPUVendor = "AMD"
	VendorIntel  GPUVendor = "Intel"
)

// GPU represents GPU information
type GPU struct {
	Vendor      GPUVendor `json:"vendor"`
	Model       string    `json:"model"`
	Memory      int64     `json:"memory"`
	Driver      string    `json:"driver"`
	Temperature float64   `json:"temperature"`
	PowerUsage  float64   `json:"powerUsage"`
	Utilization float64   `json:"utilization"`
}

// CPU represents CPU information
type CPU struct {
	Model       string  `json:"model"`
	Cores       int     `json:"cores"`
	Threads     int     `json:"threads"`
	Frequency   float64 `json:"frequency"`
	Temperature float64 `json:"temperature"`
}

// NetworkInterface represents network interface information
type NetworkInterface struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
}

// Info represents complete hardware information
type Info struct {
	CPUs              []CPU              `json:"cpus"`
	GPUs              []GPU              `json:"gpus"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

// GPUDetector defines the interface for GPU detection
type GPUDetector interface {
	Initialize() error
	IsAvailable() bool
	GetGPUs() ([]GPU, error)
	GetMetrics(*GPU) error
	UpdateMetrics([]GPU) error
	Cleanup() error
}

// OSSpecific defines the interface for OS-specific operations
type OSSpecific interface {
	Initialize() error
	GetCPUInfo() ([]CPU, error)
	GetCPUTemperature() (float64, error)
	Cleanup() error
}
