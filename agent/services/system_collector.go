package services

import (
	"os"
	"runtime"
	"sync"
	"time"
)

// SystemInfo holds the data that the agent sends with each heartbeat.
type SystemInfo struct {
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	Version  string `json:"agent_version"`

	CPUUsage float64 `json:"cpu_usage"`

	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryUsage float64 `json:"memory_usage"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`

	DiskTotal uint64  `json:"disk_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskUsage float64 `json:"disk_usage"`

	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`
	NetworkRxRate  uint64 `json:"network_rx_rate"`
	NetworkTxRate  uint64 `json:"network_tx_rate"`

	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	ProcessCount int `json:"process_count"`
}

// SystemCollector collects runtime system metrics for the agent.
type SystemCollector struct {
	mu            sync.Mutex
	lastCPUTotal  uint64
	lastCPUIdle   uint64
	lastNetRx     uint64
	lastNetTx     uint64
	lastNetSample time.Time
}

// NewSystemCollector creates a new collector instance.
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

// Collect returns the current system metrics. It maintains the previous data to
// compute deltas where needed.
func (c *SystemCollector) Collect(agentVersion string) SystemInfo {
	info := SystemInfo{
		Hostname: getHostname(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Version:  agentVersion,
	}

	if idle, total, err := readCPUTimes(); err == nil {
		c.mu.Lock()
		if c.lastCPUTotal > 0 && total > c.lastCPUTotal {
			deltaTotal := total - c.lastCPUTotal
			deltaIdle := idle - c.lastCPUIdle
			if deltaTotal > 0 {
				info.CPUUsage = float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
			}
		}
		c.lastCPUIdle = idle
		c.lastCPUTotal = total
		c.mu.Unlock()
	}

	if memTotal, memUsed, memUsage, swapTotal, swapUsed, err := readMemInfo(); err == nil {
		info.MemoryTotal = memTotal
		info.MemoryUsed = memUsed
		info.MemoryUsage = memUsage
		info.SwapTotal = swapTotal
		info.SwapUsed = swapUsed
	}

	if diskTotal, diskUsed, diskUsage, err := getDiskUsage("/"); err == nil {
		info.DiskTotal = diskTotal
		info.DiskUsed = diskUsed
		info.DiskUsage = diskUsage
	}

	if tcpConns, err := countConnections("/proc/net/tcp"); err == nil {
		info.TCPConnections = tcpConns
	}

	if udpConns, err := countConnections("/proc/net/udp"); err == nil {
		info.UDPConnections = udpConns
	}

	info.ProcessCount = countProcesses()

	if rx, tx, err := readNetIO(); err == nil {
		info.NetworkRxBytes = rx
		info.NetworkTxBytes = tx

		c.mu.Lock()
		if !c.lastNetSample.IsZero() {
			interval := time.Since(c.lastNetSample).Seconds()
			if interval > 0 {
				deltaRx := uint64(0)
				deltaTx := uint64(0)
				if rx >= c.lastNetRx {
					deltaRx = rx - c.lastNetRx
				}
				if tx >= c.lastNetTx {
					deltaTx = tx - c.lastNetTx
				}
				info.NetworkRxRate = uint64(float64(deltaRx) / interval)
				info.NetworkTxRate = uint64(float64(deltaTx) / interval)
			}
		}
		c.lastNetRx = rx
		c.lastNetTx = tx
		c.lastNetSample = time.Now()
		c.mu.Unlock()
	}

	return info
}

func getHostname() string {
	if host, err := os.Hostname(); err == nil {
		return host
	}
	return "unknown"
}
