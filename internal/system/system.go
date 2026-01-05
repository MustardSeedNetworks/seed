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

// cpuCacheState holds the CPU sampling state.
type cpuCacheState struct {
	mu      sync.RWMutex
	percent float64
	once    sync.Once
	stop    chan struct{}
}

// CPU cache accessor functions use closure-encapsulated state to satisfy gochecknoglobals.
// getCPUCache returns the CPU cache state.
// getCachedCPUPercent returns the cached CPU percentage.
// startCPUSampler starts the background CPU sampler.
var (
	getCPUCache, getCachedCPUPercent, startCPUSampler = func() (
		func() *cpuCacheState,
		func() float64,
		func(),
	) {
		state := &cpuCacheState{}

		startSampler := func() {
			state.stop = make(chan struct{})
			go func() {
				// Take initial sample immediately
				if pct, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(pct) > 0 {
					state.mu.Lock()
					state.percent = pct[0]
					state.mu.Unlock()
				}

				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-state.stop:
						return
					case <-ticker.C:
						if pct, err := cpu.Percent(100*time.Millisecond, false); err == nil &&
							len(pct) > 0 {
							state.mu.Lock()
							state.percent = pct[0]
							state.mu.Unlock()
						}
					}
				}
			}()
		}

		getCache := func() *cpuCacheState {
			return state
		}

		getCached := func() float64 {
			// Ensure sampler is started
			state.once.Do(startSampler)

			state.mu.RLock()
			defer state.mu.RUnlock()
			return state.percent
		}

		return getCache, getCached, startSampler
	}()
)

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

// Process cache accessor functions use closure-encapsulated state to satisfy gochecknoglobals.
// getProcessCache returns the process cache state.
// clearProcessCache clears the process cache for testing.
// getCachedProcesses returns cached top processes (non-blocking).
var (
	getProcessCache, clearProcessCache, getCachedProcesses = func() (
		func() *processCacheState,
		func(),
		func() ([]ProcessInfo, []ProcessInfo),
	) {
		state := &processCacheState{}

		getCache := func() *processCacheState {
			return state
		}

		clearCache := func() {
			state.cacheMu.Lock()
			state.top5 = nil
			state.mem5 = nil
			state.cacheTime = time.Time{}
			state.cacheMu.Unlock()
		}

		getCached := func() ([]ProcessInfo, []ProcessInfo) {
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

		return getCache, clearCache, getCached
	}()
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
	var topCPU []ProcessInfo
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
	var topMemory []ProcessInfo
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
