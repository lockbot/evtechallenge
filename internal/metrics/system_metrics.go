package metrics

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// MetricsManager is a singleton that manages all Prometheus metrics
type MetricsManager struct {
	// System metrics
	systemCPUUsage    *prometheus.GaugeVec
	systemMemoryUsage *prometheus.GaugeVec

	// Go runtime metrics
	goGoroutines    prometheus.Gauge
	goThreads       prometheus.Gauge
	goHeapAlloc     prometheus.Gauge
	goHeapSys       prometheus.Gauge
	goGCPauseNs     prometheus.Histogram
	goGCCPUFraction prometheus.Gauge

	// Process metrics
	processOpenFDs   prometheus.Gauge
	processStartTime prometheus.Gauge

	// Registry for manual control
	registry *prometheus.Registry

	// Singleton control
	initialized bool
	mu          sync.RWMutex
}

var (
	instance *MetricsManager
	once     sync.Once
)

// GetInstance returns the singleton instance of MetricsManager
func GetInstance() *MetricsManager {
	once.Do(func() {
		instance = &MetricsManager{
			registry: prometheus.NewRegistry(),
		}
	})
	return instance
}

// InitializeMetrics initializes all Prometheus metrics (thread-safe)
func (mm *MetricsManager) InitializeMetrics() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.initialized {
		return
	}

	// System metrics
	mm.systemCPUUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		},
		[]string{"core"},
	)

	mm.systemMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		[]string{"type"},
	)

	// Go runtime metrics
	mm.goGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines that currently exist",
		},
	)

	mm.goThreads = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_threads",
			Help: "Number of OS threads created",
		},
	)

	mm.goHeapAlloc = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_heap_alloc_bytes",
			Help: "Heap memory usage in bytes",
		},
	)

	mm.goHeapSys = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_heap_sys_bytes",
			Help: "Heap memory reserved in bytes",
		},
	)

	mm.goGCPauseNs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "go_gc_pause_nanoseconds",
			Help:    "GC pause time in nanoseconds",
			Buckets: prometheus.ExponentialBuckets(1000, 2, 20),
		},
	)

	mm.goGCCPUFraction = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_gc_cpu_fraction",
			Help: "Fraction of CPU time used by GC",
		},
	)

	// Process metrics
	mm.processOpenFDs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "process_open_fds",
			Help: "Number of open file descriptors",
		},
	)

	mm.processStartTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "process_start_time_seconds",
			Help: "Start time of the process since unix epoch in seconds",
		},
	)

	// Register all metrics with our custom registry
	mm.registry.MustRegister(
		mm.systemCPUUsage,
		mm.systemMemoryUsage,
		mm.goGoroutines,
		mm.goThreads,
		mm.goHeapAlloc,
		mm.goHeapSys,
		mm.goGCPauseNs,
		mm.goGCCPUFraction,
		mm.processOpenFDs,
		mm.processStartTime,
	)

	mm.initialized = true
}

// GetRegistry returns the Prometheus registry (only if metrics are enabled)
func GetRegistry() *prometheus.Registry {
	if os.Getenv("ENABLE_SYSTEM_METRICS") != "true" {
		return nil
	}

	mm := GetInstance()
	if !mm.initialized {
		return nil
	}

	return mm.registry
}

// StartSystemMetrics starts collecting system metrics (thread-safe)
func StartSystemMetrics(interval time.Duration) {
	// Check if system metrics are enabled
	if os.Getenv("ENABLE_SYSTEM_METRICS") != "true" {
		return
	}

	mm := GetInstance()
	mm.InitializeMetrics()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			mm.collectSystemMetrics()
			mm.collectGoRuntimeMetrics()
		}
	}()
}

// collectSystemMetrics collects system-level metrics
func (mm *MetricsManager) collectSystemMetrics() {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if !mm.initialized {
		return
	}

	// CPU usage
	if cpuPercentages, err := cpu.Percent(0, true); err == nil {
		for i, percentage := range cpuPercentages {
			mm.systemCPUUsage.WithLabelValues(fmt.Sprintf("cpu%d", i)).Set(percentage)
		}
	}

	// Memory usage
	if vmstat, err := mem.VirtualMemory(); err == nil {
		mm.systemMemoryUsage.WithLabelValues("total").Set(float64(vmstat.Total))
		mm.systemMemoryUsage.WithLabelValues("available").Set(float64(vmstat.Available))
		mm.systemMemoryUsage.WithLabelValues("used").Set(float64(vmstat.Used))
		mm.systemMemoryUsage.WithLabelValues("free").Set(float64(vmstat.Free))
	}
}

// collectGoRuntimeMetrics collects Go runtime metrics
func (mm *MetricsManager) collectGoRuntimeMetrics() {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if !mm.initialized {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm.goGoroutines.Set(float64(runtime.NumGoroutine()))
	mm.goThreads.Set(float64(runtime.GOMAXPROCS(0)))
	mm.goHeapAlloc.Set(float64(m.HeapAlloc))
	mm.goHeapSys.Set(float64(m.HeapSys))
	mm.goGCPauseNs.Observe(float64(m.PauseNs[(m.NumGC+255)%256]))
	mm.goGCCPUFraction.Set(m.GCCPUFraction)
}
