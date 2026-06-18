package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// CPUCollector collects CPU usage percentage.
type CPUCollector struct {
	previous []cpu.TimesStat
}

// NewCPUCollector creates a CPUCollector.
func NewCPUCollector() *CPUCollector {
	prev, _ := cpu.Times(false)
	return &CPUCollector{previous: prev}
}

// Percent returns the current CPU usage percentage.
// Uses gopsutil which handles /proc/stat parsing cross-platform.
func (c *CPUCollector) Percent() (float64, error) {
	// Get overall CPU percentage with a small interval
	percent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return 0, err
	}
	if len(percent) > 0 {
		return percent[0], nil
	}
	return 0, nil
}
