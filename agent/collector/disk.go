package collector

import (
	"github.com/shirou/gopsutil/v3/disk"
)

// DiskInfo holds collected disk data for one mount.
type DiskInfo struct {
	Mount   string
	TotalGB int64
	UsedGB  int64
	Percent float64
}

// CollectDisks returns disk usage for all physical mount points.
// Excludes pseudo filesystems (tmpfs, devtmpfs, squashfs, overlay, etc.).
func CollectDisks() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var results []DiskInfo
	for _, p := range partitions {
		// Skip pseudo filesystems
		skipFSTypes := map[string]bool{
			"tmpfs": true, "devtmpfs": true, "squashfs": true,
			"overlay": true, "proc": true, "sysfs": true,
			"cgroup": true, "devpts": true, "shm": true,
			"hugetlbfs": true, "mqueue": true, "pstore": true,
			"efivarfs": true, "tracefs": true, "ramfs": true,
			"autofs": true, "configfs": true, "selinuxfs": true,
			"debugfs": true, "fusectl": true, "bpf": true,
			"securityfs": true,
		}
		if skipFSTypes[p.Fstype] {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		results = append(results, DiskInfo{
			Mount:   p.Mountpoint,
			TotalGB: int64(usage.Total / 1024 / 1024 / 1024),
			UsedGB:  int64(usage.Used / 1024 / 1024 / 1024),
			Percent: usage.UsedPercent,
		})
	}

	return results, nil
}
