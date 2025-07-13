/*
 * Package metrics implements system resource monitoring and metrics collection
 * for the KrakenHashes agent.
 *
 * This is a simplified cross-platform implementation that collects basic
 * CPU and memory metrics. GPU metrics will be obtained from hashcat's
 * JSON status output during job execution.
 */
package metrics

import (
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// SystemMetrics holds system performance metrics
type SystemMetrics struct {
	CPUUsage       float64
	MemoryUsage    float64
	GPUUtilization float64
	GPUUsage       float64
	GPUTemp        float64
}

// Collector manages system metrics collection
type Collector struct {
	interval   time.Duration
	gpuEnabled bool
}

// Config defines the configuration for the metrics collector
type Config struct {
	CollectionInterval time.Duration
	EnableGPU          bool
}

// New creates a new metrics collector
func New(config Config) (*Collector, error) {
	interval := config.CollectionInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	collector := &Collector{
		interval:   interval,
		gpuEnabled: config.EnableGPU,
	}

	if config.EnableGPU {
		debug.Info("GPU metrics will be obtained from hashcat during job execution")
	}

	return collector, nil
}

// Collect gathers current system metrics
func (c *Collector) Collect() (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// Collect CPU metrics
	if err := c.collectCPUMetrics(metrics); err != nil {
		debug.Error("Failed to collect CPU metrics: %v", err)
		// Continue with other metrics even if CPU fails
	}

	// Collect memory metrics
	if err := c.collectMemoryMetrics(metrics); err != nil {
		debug.Error("Failed to collect memory metrics: %v", err)
		// Continue with other metrics even if memory fails
	}

	// GPU metrics will be populated from hashcat JSON status output
	// during job execution. For now, we leave them at zero.
	// TODO: Parse GPU metrics from hashcat status JSON when available

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

// Close cleans up any resources used by the collector
func (c *Collector) Close() error {
	// No resources to clean up in this implementation
	return nil
}

// GetInterval returns the collection interval
func (c *Collector) GetInterval() time.Duration {
	return c.interval
}

// IsGPUEnabled returns whether GPU metrics collection is enabled
func (c *Collector) IsGPUEnabled() bool {
	return c.gpuEnabled
}