// Package system provides system health metrics collection.
package system

import (
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
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

// cpuCache holds the most recent CPU sample, written by the Collector's
// background sampler and read by Health.
type cpuCache struct {
	mu      sync.RWMutex
	percent float64
}

// cpuPercent returns the most recent sample. Zero until the sampler has taken
// its first reading, which happens within cpuSampleInterval of NewCollector.
func (c *cpuCache) cpuPercent() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.percent
}

// sample takes one reading and stores it. A failed sample leaves the previous
// value in place, which is a better answer than reporting zero load.
func (c *cpuCache) sample() {
	pct, err := cpu.Percent(cpuSampleInterval, false)
	if err != nil || len(pct) == 0 {
		return
	}
	c.mu.Lock()
	c.percent = pct[0]
	c.mu.Unlock()
}

// processCacheTTL is how long process info remains valid.
const processCacheTTL = 5 * time.Second

// processCache holds the last process snapshot plus the single-flight flag that
// keeps concurrent readers from each launching their own enumeration.
type processCache struct {
	cacheMu     sync.RWMutex
	top5        []ProcessInfo
	mem5        []ProcessInfo
	cacheTime   time.Time
	updateMu    sync.Mutex
	updateInFly bool
}

// processes returns the cached snapshot without blocking, refreshing it in the
// background when stale. Enumerating processes costs hundreds of milliseconds
// and pins an OS thread in cgo, so at most one refresh runs at a time —
// updateInFly is what enforces that, and it only works because the cache is
// owned by a Collector rather than rebuilt on every call.
func (p *processCache) processes() ([]ProcessInfo, []ProcessInfo) {
	p.cacheMu.RLock()
	cacheAge := time.Since(p.cacheTime)
	topCPU := p.top5
	topMemory := p.mem5
	p.cacheMu.RUnlock()

	if cacheAge > processCacheTTL {
		p.updateMu.Lock()
		if !p.updateInFly {
			p.updateInFly = true
			go func() {
				defer func() {
					p.updateMu.Lock()
					p.updateInFly = false
					p.updateMu.Unlock()
				}()

				cpuProcs, memProcs := getTopProcessesInternal()
				p.cacheMu.Lock()
				p.top5 = cpuProcs
				p.mem5 = memProcs
				p.cacheTime = time.Now()
				p.cacheMu.Unlock()
			}()
		}
		p.updateMu.Unlock()
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

// Collector samples host health. It owns the CPU sampler and the process-list
// cache, so construct one and keep it for the process lifetime — a Collector
// per request would start a sampler per request and never reuse a cache.
type Collector struct {
	cpu  cpuCache
	proc processCache
	stop chan struct{}
	done chan struct{}
}

// NewCollector starts the background CPU sampler and returns a ready Collector.
// Close stops the sampler.
func NewCollector() *Collector {
	c := &Collector{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go c.sampleCPU()
	return c
}

// Close stops the background sampler and waits for it to exit.
func (c *Collector) Close() error {
	close(c.stop)
	<-c.done
	return nil
}

// Health collects current system health metrics.
func (c *Collector) Health() (*Health, error) {
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
	h.CPUPercent = c.cpu.cpuPercent()

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
		topCPU, topMemory := c.proc.processes()
		if h.CPUPercent >= warningThreshold {
			h.TopCPUProcesses = topCPU
		}
		if h.MemoryPercent >= warningThreshold {
			h.TopMemoryProcesses = topMemory
		}
	}

	return h, nil
}

// sampleCPU takes a reading immediately and then every cpuTickerInterval until
// Close. cpu.Percent blocks for cpuSampleInterval, so this runs off the request
// path and Health only ever reads the last stored value.
func (c *Collector) sampleCPU() {
	defer close(c.done)

	c.cpu.sample()

	ticker := time.NewTicker(cpuTickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.cpu.sample()
		}
	}
}
