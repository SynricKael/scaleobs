package collector

import (
	"log"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/net"
)

// NetworkCollector tracks network I/O rates by diffing successive counter reads.
type NetworkCollector struct {
	mu         sync.Mutex
	prevSent   uint64
	prevRecv   uint64
	prevTime   time.Time
	firstRead  bool
}

// NewNetworkCollector creates a NetworkCollector. The first call to Rates
// returns 0 (no previous sample to diff against).
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{firstRead: true}
}

// Rates returns bytes/sec for upload (sent) and download (recv), summed
// across ALL network interfaces. Returns 0 on the first call or if the
// counters reset (e.g. reboot between reads).
//
// Uses gopsutil's net.IOCounters(false) which returns aggregate counters
// across all interfaces.
func (c *NetworkCollector) Rates() (sentPerSec, recvPerSec float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	counters, err := net.IOCounters(false) // false = no per-interface detail
	if err != nil {
		log.Printf("[collector] network IOCounters error: %v", err)
		return 0, 0
	}
	if len(counters) == 0 {
		return 0, 0
	}

	now := time.Now()
	totalSent := counters[0].BytesSent
	totalRecv := counters[0].BytesRecv

	if c.firstRead {
		c.prevSent = totalSent
		c.prevRecv = totalRecv
		c.prevTime = now
		c.firstRead = false
		return 0, 0
	}

	elapsed := now.Sub(c.prevTime).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}

	// Detect counter reset (reboot) — if new < old, start over
	if totalSent < c.prevSent || totalRecv < c.prevRecv {
		c.prevSent = totalSent
		c.prevRecv = totalRecv
		c.prevTime = now
		return 0, 0
	}

	sentPerSec = float64(totalSent-c.prevSent) / elapsed
	recvPerSec = float64(totalRecv-c.prevRecv) / elapsed

	c.prevSent = totalSent
	c.prevRecv = totalRecv
	c.prevTime = now

	return sentPerSec, recvPerSec
}
