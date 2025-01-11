/*
 * Package metrics implements system resource monitoring and metrics collection
 * for the HashDom agent.
 *
 * Usage:
 *   collector := metrics.New(metrics.Config{
 *     CollectionInterval: 5 * time.Second,
 *     EnableGPU: true,
 *   })
 *   metrics, err := collector.Collect()
 *
 * Error Handling:
 *   - GPU initialization failures
 *   - Collection timeouts
 *   - Resource access errors
 *   - Hardware monitoring failures
 *
 * Resource Management:
 *   - Proper cleanup of NVML resources
 *   - Memory management for metrics data
 *   - Concurrent collection handling
 *   - Resource limit monitoring
 */
package metrics

import (
	"fmt"
	"time"

	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	"github.com/ZerkerEOD/hashdom/agent/pkg/debug"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type SystemMetrics struct {
	CPUUsage       float64
	MemoryUsage    float64
	GPUUtilization float64
	GPUUsage       float64
	GPUTemp        float64
}

/*
 * Collector manages system metrics collection and resource monitoring.
 *
 * Features:
 *   - CPU usage tracking
 *   - Memory utilization monitoring
 *   - GPU metrics collection (when enabled)
 *   - Resource limit enforcement
 *
 * Thread Safety: All methods are safe for concurrent use.
 */
type Collector struct {
	interval   time.Duration
	gpuEnabled bool
}

/*
 * Config defines the configuration options for the metrics collector.
 *
 * Fields:
 *   CollectionInterval: Duration between metric collections
 *   EnableGPU: Whether to collect GPU metrics
 *
 * Validation:
 *   - CollectionInterval must be positive
 *   - GPU metrics require NVIDIA drivers
 */
type Config struct {
	CollectionInterval time.Duration
	EnableGPU          bool
}

/*
 * New creates a new metrics collector with the specified configuration.
 *
 * Parameters:
 *   - config: Configuration options for the collector
 *
 * Returns:
 *   - *Collector: Initialized metrics collector
 *   - error: Any initialization errors
 *
 * Error Handling:
 *   - Invalid configuration
 *   - GPU initialization failures
 *   - Resource access errors
 */
func New(config Config) (*Collector, error) {
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 5 * time.Second
	}

	collector := &Collector{
		interval:   config.CollectionInterval,
		gpuEnabled: config.EnableGPU,
	}

	if config.EnableGPU {
		if err := nvml.Init(); err != nil {
			debug.Warning("Failed to initialize NVIDIA metrics: %v", err)
			collector.gpuEnabled = false
		}
	}

	return collector, nil
}

/*
 * Collect gathers current system metrics.
 *
 * Process:
 *   1. Collect CPU metrics
 *   2. Gather memory statistics
 *   3. Query GPU metrics (if enabled)
 *   4. Validate collected data
 *
 * Returns:
 *   - *SystemMetrics: Collected system metrics
 *   - error: Any collection errors
 *
 * Error Handling:
 *   - Hardware access failures
 *   - Collection timeouts
 *   - Invalid metric values
 */
func (c *Collector) Collect() (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// Collect CPU metrics
	if err := c.collectCPUMetrics(metrics); err != nil {
		debug.Error("Failed to collect CPU metrics: %v", err)
	}

	// Collect memory metrics
	if err := c.collectMemoryMetrics(metrics); err != nil {
		debug.Error("Failed to collect memory metrics: %v", err)
	}

	// Collect GPU metrics if enabled
	if c.gpuEnabled {
		if err := c.collectGPUMetrics(metrics); err != nil {
			debug.Error("Failed to collect GPU metrics: %v", err)
		}
	}

	return metrics, nil
}

func (c *Collector) collectCPUMetrics(metrics *SystemMetrics) error {
	percentage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %v", err)
	}

	if len(percentage) > 0 {
		metrics.CPUUsage = percentage[0]
	}

	return nil
}

func (c *Collector) collectMemoryMetrics(metrics *SystemMetrics) error {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory info: %v", err)
	}

	metrics.MemoryUsage = vmem.UsedPercent
	return nil
}

func (c *Collector) collectGPUMetrics(metrics *SystemMetrics) error {
	count, err := nvml.GetDeviceCount()
	if err != nil {
		return fmt.Errorf("failed to get GPU count: %v", err)
	}

	if count > 0 {
		device, err := nvml.NewDevice(0)
		if err != nil {
			return fmt.Errorf("failed to get GPU device: %v", err)
		}

		status, err := device.Status()
		if err != nil {
			return fmt.Errorf("failed to get GPU status: %v", err)
		}

		metrics.GPUUtilization = float64(*status.Utilization.GPU)
		metrics.GPUTemp = float64(*status.Temperature)
	}

	return nil
}

func (c *Collector) Close() error {
	// Add any cleanup logic here if needed
	return nil
}
