package collector

import (
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryInfo holds the collected memory data.
type MemoryInfo struct {
	TotalMB int64
	UsedMB  int64
	Percent float64
}

// CollectMemory returns current memory usage.
func CollectMemory() (*MemoryInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	totalMB := int64(v.Total / 1024 / 1024)
	usedMB := int64(v.Used / 1024 / 1024)

	return &MemoryInfo{
		TotalMB: totalMB,
		UsedMB:  usedMB,
		Percent: v.UsedPercent,
	}, nil
}
