// Package system provides system health metrics collection.
package system

import (
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

const (
	cpuSampleInterval = 100 * time.Millisecond
	cpuTickerInterval = 2 * time.Second
)

// Memory conversion constants.
const (
	// bytesPerKilobyte is the number of bytes in one kilobyte.
	bytesPerKilobyte = 1024
	// bytesPerMegabyte is the number of bytes in one megabyte.
	bytesPerMegabyte = bytesPerKilobyte * bytesPerKilobyte
)

// topProcessCount is the number of top processes to return.
const topProcessCount = 5

// cpuCacheState holds the CPU sampling state.
type cpuCacheState struct {
	mu      sync.RWMutex
	percent float64
	once    sync.Once
	stop    chan struct{}
}

// cpuCacheSingleton holds the singleton CPU cache state using [sync.OnceValue].
// This pattern satisfies gochecknoglobals by using a function rather than a variable.
func cpuCacheSingleton() *cpuCacheState {
	return sync.OnceValue(func() *cpuCacheState {
		return &cpuCacheState{}
	})()
}

// getCachedCPUPercent returns the cached CPU percentage.
func getCachedCPUPercent() float64 {
	state := cpuCacheSingleton()

	startSampler := func() {
		state.stop = make(chan struct{})
		go func() {
			// Take initial sample immediately
			if pct, err := cpu.Percent(cpuSampleInterval, false); err == nil && len(pct) > 0 {
				state.mu.Lock()
				state.percent = pct[0]
				state.mu.Unlock()
			}

			ticker := time.NewTicker(cpuTickerInterval)
			defer ticker.Stop()

			for {
				select {
				case <-state.stop:
					return
				case <-ticker.C:
					if pct, err := cpu.Percent(cpuSampleInterval, false); err == nil &&
						len(pct) > 0 {
						state.mu.Lock()
						state.percent = pct[0]
						state.mu.Unlock()
					}
				}
			}
		}()
	}

	// Ensure sampler is started
	state.once.Do(startSampler)

	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.percent
}

// processCacheTTL is how long process info remains valid.
const processCacheTTL = 5 * time.Second

// processCacheState holds the process cache state.
type processCacheState struct {
	cacheMu     sync.RWMutex
	top5        []ProcessInfo
	mem5        []ProcessInfo
	cacheTime   time.Time
	updateMu    sync.Mutex
	updateInFly bool
}

// processCacheSingleton holds the singleton process cache state using [sync.OnceValue].
func processCacheSingleton() *processCacheState {
	return sync.OnceValue(func() *processCacheState {
		return &processCacheState{}
	})()
}

// getCachedProcesses returns cached top processes (non-blocking).
func getCachedProcesses() ([]ProcessInfo, []ProcessInfo) {
	state := processCacheSingleton()

	state.cacheMu.RLock()
	cacheAge := time.Since(state.cacheTime)
	topCPU := state.top5
	topMemory := state.mem5
	state.cacheMu.RUnlock()

	// If cache is stale, trigger background update (non-blocking)
	if cacheAge > processCacheTTL {
		state.updateMu.Lock()
		if !state.updateInFly {
			state.updateInFly = true
			go func() {
				defer func() {
					state.updateMu.Lock()
					state.updateInFly = false
					state.updateMu.Unlock()
				}()

				cpuProcs, memProcs := getTopProcessesInternal()
				state.cacheMu.Lock()
				state.top5 = cpuProcs
				state.mem5 = memProcs
				state.cacheTime = time.Now()
				state.cacheMu.Unlock()
			}()
		}
		state.updateMu.Unlock()
	}

	return topCPU, topMemory
}

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

// getTopProcessesInternal collects information about top resource-consuming processes.
// Returns top 5 processes by CPU and memory usage.
// This is the internal (slow) version - use getCachedProcesses() for non-blocking access.
func getTopProcessesInternal() ([]ProcessInfo, []ProcessInfo) {
	procs, err := process.Processes()
	if err != nil {
		return nil, nil
	}

	var processes []ProcessInfo
	for _, p := range procs {
		// Get process name
		name, nameErr := p.Name()
		if nameErr != nil {
			continue
		}

		// Get CPU percent (over a short interval)
		cpuPercent, cpuErr := p.CPUPercent()
		if cpuErr != nil {
			cpuPercent = 0
		}

		// Get memory info
		memInfo, memErr := p.MemoryInfo()
		if memErr != nil {
			continue
		}
		memoryMB := float64(memInfo.RSS) / bytesPerMegabyte

		processes = append(processes, ProcessInfo{
			Name:       name,
			PID:        int(p.Pid),
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
		})
	}

	// Sort by CPU and get top processes
	cpuSorted := make([]ProcessInfo, len(processes))
	copy(cpuSorted, processes)
	sort.Slice(cpuSorted, func(i, j int) bool {
		return cpuSorted[i].CPUPercent > cpuSorted[j].CPUPercent
	})
	var topCPU []ProcessInfo
	if len(cpuSorted) > topProcessCount {
		topCPU = cpuSorted[:topProcessCount]
	} else {
		topCPU = cpuSorted
	}

	// Sort by memory and get top processes
	memSorted := make([]ProcessInfo, len(processes))
	copy(memSorted, processes)
	sort.Slice(memSorted, func(i, j int) bool {
		return memSorted[i].MemoryMB > memSorted[j].MemoryMB
	})
	var topMemory []ProcessInfo
	if len(memSorted) > topProcessCount {
		topMemory = memSorted[:topProcessCount]
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

	// CPU percentage (from background sampler - non-blocking)
	h.CPUPercent = getCachedCPUPercent()

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
	// Uses cached data for fast response (non-blocking)
	const warningThreshold = 75.0
	if h.CPUPercent >= warningThreshold || h.MemoryPercent >= warningThreshold {
		topCPU, topMemory := getCachedProcesses()
		if h.CPUPercent >= warningThreshold {
			h.TopCPUProcesses = topCPU
		}
		if h.MemoryPercent >= warningThreshold {
			h.TopMemoryProcesses = topMemory
		}
	}

	return h, nil
}
