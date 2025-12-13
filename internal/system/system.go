// Package system provides system health metrics collection.
package system

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// Health contains system health metrics.
type Health struct {
	// CPU usage percentage (0-100)
	CPUPercent float64 `json:"cpuPercent"`
	// Memory usage percentage (0-100)
	MemoryPercent float64 `json:"memoryPercent"`
	// Memory used in bytes
	MemoryUsed uint64 `json:"memoryUsed"`
	// Memory total in bytes
	MemoryTotal uint64 `json:"memoryTotal"`
	// Disk usage percentage (0-100)
	DiskPercent float64 `json:"diskPercent"`
	// Disk used in bytes
	DiskUsed uint64 `json:"diskUsed"`
	// Disk total in bytes
	DiskTotal uint64 `json:"diskTotal"`
	// System uptime in seconds
	Uptime uint64 `json:"uptime"`
	// Load averages (1, 5, 15 minutes)
	LoadAvg1  float64 `json:"loadAvg1"`
	LoadAvg5  float64 `json:"loadAvg5"`
	LoadAvg15 float64 `json:"loadAvg15"`
	// Number of running goroutines
	Goroutines int `json:"goroutines"`
	// Process memory usage
	ProcessMemory uint64 `json:"processMemory"`
	// Hostname
	Hostname string `json:"hostname"`
	// Operating system
	OS string `json:"os"`
	// Architecture
	Arch string `json:"arch"`
	// Number of CPUs
	NumCPU int `json:"numCpu"`
}

// GetHealth collects current system health metrics.
func GetHealth() (*Health, error) {
	h := &Health{
		Goroutines: runtime.NumGoroutine(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		h.Hostname = hostname
	}

	// CPU percentage (average over a short interval)
	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		h.CPUPercent = cpuPercent[0]
	}

	// Memory stats
	if vmStat, err := mem.VirtualMemory(); err == nil {
		h.MemoryPercent = vmStat.UsedPercent
		h.MemoryUsed = vmStat.Used
		h.MemoryTotal = vmStat.Total
	}

	// Disk stats (root filesystem)
	if diskStat, err := disk.Usage("/"); err == nil {
		h.DiskPercent = diskStat.UsedPercent
		h.DiskUsed = diskStat.Used
		h.DiskTotal = diskStat.Total
	}

	// System uptime
	if uptimeInfo, err := host.Uptime(); err == nil {
		h.Uptime = uptimeInfo
	}

	// Load averages
	if loadStat, err := load.Avg(); err == nil {
		h.LoadAvg1 = loadStat.Load1
		h.LoadAvg5 = loadStat.Load5
		h.LoadAvg15 = loadStat.Load15
	}

	// Process memory (from Go runtime)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	h.ProcessMemory = memStats.Alloc

	return h, nil
}
