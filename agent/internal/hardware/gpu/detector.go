package gpu

import (
	"fmt"
	"sync"

	"github.com/ZerkerEOD/hashdom-agent/internal/hardware/types"
	"github.com/ZerkerEOD/hashdom-agent/pkg/debug"
)

// DetectorFactory manages GPU detectors for different vendors
type DetectorFactory struct {
	mu        sync.RWMutex
	detectors map[types.GPUVendor]types.GPUDetector
}

// NewFactory creates a new GPU detector factory
func NewFactory() types.GPUDetector {
	return &DetectorFactory{
		detectors: make(map[types.GPUVendor]types.GPUDetector),
	}
}

// Initialize prepares all GPU detectors
func (f *DetectorFactory) Initialize() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Initialize vendor-specific detectors
	detectors := map[types.GPUVendor]types.GPUDetector{
		types.VendorNVIDIA: NewNVIDIADetector(),
		types.VendorAMD:    NewAMDDetector(),
		types.VendorIntel:  NewIntelDetector(),
	}

	// Initialize and check availability of each detector
	for vendor, detector := range detectors {
		if err := detector.Initialize(); err != nil {
			debug.Warning("Failed to initialize %s GPU detector: %v", vendor, err)
			continue
		}

		if detector.IsAvailable() {
			f.detectors[vendor] = detector
			debug.Info("Initialized %s GPU detector", vendor)
		} else {
			debug.Info("No %s GPUs detected", vendor)
			detector.Cleanup()
		}
	}

	return nil
}

// IsAvailable checks if any GPU detectors are available
func (f *DetectorFactory) IsAvailable() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.detectors) > 0
}

// GetGPUs returns information about all detected GPUs
func (f *DetectorFactory) GetGPUs() ([]types.GPU, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var gpus []types.GPU
	var errs []error

	// Collect GPUs from all available detectors
	for vendor, detector := range f.detectors {
		vendorGPUs, err := detector.GetGPUs()
		if err != nil {
			debug.Error("Failed to get %s GPUs: %v", vendor, err)
			errs = append(errs, fmt.Errorf("%s: %v", vendor, err))
			continue
		}
		gpus = append(gpus, vendorGPUs...)
	}

	if len(gpus) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("failed to detect GPUs: %v", errs)
	}

	return gpus, nil
}

// GetMetrics updates metrics for a specific GPU
func (f *DetectorFactory) GetMetrics(gpu *types.GPU) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	detector, ok := f.detectors[gpu.Vendor]
	if !ok {
		return fmt.Errorf("no detector available for %s GPU", gpu.Vendor)
	}

	return detector.GetMetrics(gpu)
}

// Cleanup releases all detector resources
func (f *DetectorFactory) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error

	// Cleanup all detectors
	for vendor, detector := range f.detectors {
		if err := detector.Cleanup(); err != nil {
			debug.Error("Failed to cleanup %s GPU detector: %v", vendor, err)
			errs = append(errs, fmt.Errorf("%s: %v", vendor, err))
		}
	}

	// Clear the detectors map
	f.detectors = make(map[types.GPUVendor]types.GPUDetector)

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

// UpdateMetrics refreshes metrics for all GPUs
func (f *DetectorFactory) UpdateMetrics(gpus []types.GPU) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var errs []error
	for i := range gpus {
		if err := f.GetMetrics(&gpus[i]); err != nil {
			errs = append(errs, fmt.Errorf("GPU %s: %v", gpus[i].Model, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to update GPU metrics: %v", errs)
	}
	return nil
}
