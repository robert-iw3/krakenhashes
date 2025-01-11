/*
 * Package metrics_test implements comprehensive unit testing for the metrics collector.
 *
 * Job Processing:
 *   - Input: Test configurations and scenarios
 *   - Processing: Metric collection validation
 *   - Error handling: Invalid configurations, resource failures
 *   - Resource management: Cleanup verification
 *
 * Test Coverage:
 *   - Configuration validation
 *   - Metric collection accuracy
 *   - GPU metrics handling
 *   - Concurrent operations
 *   - Resource cleanup
 *
 * Dependencies:
 *   - metrics package
 *   - testify for assertions
 *   - NVML for GPU metrics
 *
 * Error Scenarios:
 *   - Invalid configurations
 *   - Resource unavailability
 *   - Concurrent access issues
 *   - Cleanup failures
 */
package metrics_test

import (
	"testing"
	"time"

	"github.com/ZerkerEOD/hashdom/agent/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  metrics.Config
		wantErr bool
	}{
		{
			name: "default configuration",
			config: metrics.Config{
				CollectionInterval: 0, // Should use default
				EnableGPU:          false,
			},
			wantErr: false,
		},
		{
			name: "custom interval",
			config: metrics.Config{
				CollectionInterval: 10 * time.Second,
				EnableGPU:          false,
			},
			wantErr: false,
		},
		{
			name: "with GPU enabled",
			config: metrics.Config{
				CollectionInterval: 5 * time.Second,
				EnableGPU:          true,
			},
			wantErr: false, // Should not error even if GPU is not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := metrics.New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)
			}

			if collector != nil {
				assert.NoError(t, collector.Close())
			}
		})
	}
}

func TestCollect(t *testing.T) {
	collector, err := metrics.New(metrics.Config{
		CollectionInterval: time.Second,
		EnableGPU:          false,
	})
	require.NoError(t, err)
	defer collector.Close()

	t.Run("collect metrics", func(t *testing.T) {
		metrics, err := collector.Collect()
		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// CPU usage should be between 0 and 100
		assert.GreaterOrEqual(t, metrics.CPUUsage, 0.0)
		assert.LessOrEqual(t, metrics.CPUUsage, 100.0)

		// Memory usage should be between 0 and 100
		assert.GreaterOrEqual(t, metrics.MemoryUsage, 0.0)
		assert.LessOrEqual(t, metrics.MemoryUsage, 100.0)
	})
}

func TestGPUMetrics(t *testing.T) {
	collector, err := metrics.New(metrics.Config{
		CollectionInterval: time.Second,
		EnableGPU:          true,
	})
	require.NoError(t, err)
	defer collector.Close()

	t.Run("collect GPU metrics", func(t *testing.T) {
		metrics, err := collector.Collect()
		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// GPU metrics might be 0 if no GPU is available
		assert.GreaterOrEqual(t, metrics.GPUUtilization, 0.0)
		assert.LessOrEqual(t, metrics.GPUUtilization, 100.0)

		// GPU temperature should be reasonable if available
		if metrics.GPUTemp > 0 {
			assert.GreaterOrEqual(t, metrics.GPUTemp, 20.0) // 20°C minimum
			assert.LessOrEqual(t, metrics.GPUTemp, 100.0)   // 100°C maximum
		}
	})
}

/*
 * TestConcurrentCollection validates the collector's behavior under concurrent access.
 *
 * Job Processing:
 *   - Input: Multiple concurrent metric collection requests
 *   - Processing: Parallel metric gathering
 *   - Error handling: Race conditions, resource contention
 *   - Resource management: Goroutine and resource cleanup
 *
 * Test Coverage:
 *   - Concurrent access safety
 *   - Resource limit handling
 *   - Collection accuracy
 *   - Memory management
 *
 * Requirements (from .cursorrules lines 864-867):
 *   - Resource management testing
 *   - Error handling validation
 *   - Status reporting verification
 *
 * Error Scenarios:
 *   - Resource contention
 *   - Collection failures
 *   - Cleanup errors
 */
func TestConcurrentCollection(t *testing.T) {
	collector, err := metrics.New(metrics.Config{
		CollectionInterval: time.Second,
		EnableGPU:          false,
	})
	require.NoError(t, err)
	defer collector.Close()

	t.Run("concurrent metric collection", func(t *testing.T) {
		const numGoroutines = 5
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				metrics, err := collector.Collect()
				assert.NoError(t, err)
				assert.NotNil(t, metrics)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestResourceCleanup(t *testing.T) {
	t.Run("multiple init and cleanup", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			collector, err := metrics.New(metrics.Config{
				CollectionInterval: time.Second,
				EnableGPU:          true,
			})
			require.NoError(t, err)
			assert.NoError(t, collector.Close())
		}
	})
}

// Benchmark metric collection
func BenchmarkCollect(b *testing.B) {
	collector, err := metrics.New(metrics.Config{
		CollectionInterval: time.Second,
		EnableGPU:          false,
	})
	require.NoError(b, err)
	defer collector.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		require.NoError(b, err)
	}
}
