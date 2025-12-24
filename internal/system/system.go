// Package system provides system health metrics collection.
package system

import (
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo contains information about a single process.
type ProcessInfo struct {
	Name       string  `json:"name"`
	PID        int     `json:"pid"`
	CPUPercent float64 `json:"cpuPercent"`
	MemoryMB   float64 `json:"memoryMb"`
}

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
	// Top CPU consuming processes (only populated when CPU > 75%)
	TopCPUProcesses []ProcessInfo `json:"topCpuProcesses,omitempty"`
	// Top memory consuming processes (only populated when memory > 75%)
	TopMemoryProcesses []ProcessInfo `json:"topMemoryProcesses,omitempty"`
}

// getTopProcesses collects information about top resource-consuming processes.
// Returns top 5 processes by CPU and memory usage.
func getTopProcesses() (topCPU, topMemory []ProcessInfo) {
	procs, err := process.Processes()
	if err != nil {
		return nil, nil
	}

	var processes []ProcessInfo
	for _, p := range procs {
		// Get process name
		name, err := p.Name()
		if err != nil {
			continue
		}

		// Get CPU percent (over a short interval)
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		// Get memory info
		memInfo, err := p.MemoryInfo()
		if err != nil {
			continue
		}
		memoryMB := float64(memInfo.RSS) / (1024 * 1024)

		processes = append(processes, ProcessInfo{
			Name:       name,
			PID:        int(p.Pid),
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
		})
	}

	// Sort by CPU and get top 5
	cpuSorted := make([]ProcessInfo, len(processes))
	copy(cpuSorted, processes)
	sort.Slice(cpuSorted, func(i, j int) bool {
		return cpuSorted[i].CPUPercent > cpuSorted[j].CPUPercent
	})
	if len(cpuSorted) > 5 {
		topCPU = cpuSorted[:5]
	} else {
		topCPU = cpuSorted
	}

	// Sort by memory and get top 5
	memSorted := make([]ProcessInfo, len(processes))
	copy(memSorted, processes)
	sort.Slice(memSorted, func(i, j int) bool {
		return memSorted[i].MemoryMB > memSorted[j].MemoryMB
	})
	if len(memSorted) > 5 {
		topMemory = memSorted[:5]
	} else {
		topMemory = memSorted
	}

	return topCPU, topMemory
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

	// Collect top processes only when thresholds exceeded (75%)
	const warningThreshold = 75.0
	if h.CPUPercent >= warningThreshold || h.MemoryPercent >= warningThreshold {
		topCPU, topMemory := getTopProcesses()
		if h.CPUPercent >= warningThreshold {
			h.TopCPUProcesses = topCPU
		}
		if h.MemoryPercent >= warningThreshold {
			h.TopMemoryProcesses = topMemory
		}
	}

	return h, nil
}
