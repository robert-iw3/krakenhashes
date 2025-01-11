package gpu

import (
	"github.com/ZerkerEOD/krakenhashes/agent/internal/hardware/types"
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
)

// Factory manages GPU detectors
type Factory struct {
	detectors map[types.GPUVendor]types.GPUDetector
}

// NewDetectorFactory creates a new GPU detector factory
func NewDetectorFactory() *Factory {
	f := &Factory{
		detectors: make(map[types.GPUVendor]types.GPUDetector),
	}

	// Initialize detectors
	f.detectors[types.VendorNVIDIA] = NewNVIDIADetector()
	f.detectors[types.VendorAMD] = NewAMDDetector()
	f.detectors[types.VendorIntel] = NewIntelDetector()

	return f
}

// GetGPUs returns information about all detected GPUs
func (f *Factory) GetGPUs() ([]types.GPU, error) {
	var gpus []types.GPU

	for vendor, detector := range f.detectors {
		if !detector.IsAvailable() {
			debug.Debug("No %s GPUs available", vendor)
			continue
		}

		vendorGPUs, err := detector.GetGPUs()
		if err != nil {
			debug.Error("Failed to get %s GPUs: %v", vendor, err)
			continue
		}

		gpus = append(gpus, vendorGPUs...)
	}

	return gpus, nil
}

// UpdateMetrics refreshes metrics for all GPUs
func (f *Factory) UpdateMetrics(gpus []types.GPU) error {
	for vendor, detector := range f.detectors {
		if !detector.IsAvailable() {
			debug.Debug("No %s GPUs available for metrics update", vendor)
			continue
		}

		if err := detector.UpdateMetrics(gpus); err != nil {
			debug.Error("Failed to update %s GPU metrics: %v", vendor, err)
		}
	}

	return nil
}

// Cleanup releases resources for all detectors
func (f *Factory) Cleanup() error {
	for vendor, detector := range f.detectors {
		if err := detector.Cleanup(); err != nil {
			debug.Error("Failed to cleanup %s detector: %v", vendor, err)
		}
	}
	return nil
}
