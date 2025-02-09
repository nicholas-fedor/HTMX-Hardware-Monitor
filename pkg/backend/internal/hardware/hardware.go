package hardware

import (
	"fmt"
	"runtime"
	"sort"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// humanizeBytes converts bytes into human-readable units
func humanizeBytes(size float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", size, units[i])
}

// GetSystemAndCpuSection combines system and CPU information into one section
func GetSystemSection() (string, error) {
	runTimeOS := runtime.GOOS

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return `<div class="error">Error fetching system information: ` + err.Error() + `</div>`, err
	}

	hostStat, err := host.Info()
	if err != nil {
		return `<div class="error">Error fetching system information: ` + err.Error() + `</div>`, err
	}

	cpuStat, err := cpu.Info()
	if err != nil {
		return `<div class="error">Error fetching CPU information: ` + err.Error() + `</div>`, err
	}

	// Condensed CPU Info
	cpuInfo := fmt.Sprintf("%s x %d cores", cpuStat[0].ModelName, len(cpuStat))

	// Memory usage percentage
	usedMemoryPercent := float64(vmStat.Used) / float64(vmStat.Total) * 100.0

	output := fmt.Sprintf(`
    <div class="system-info">
        <p><strong>Hostname:</strong> %s</p>
        <p><strong>OS:</strong> %s</p>
        <p><strong>CPU:</strong> %s</p>
        <p><strong>Used Memory:</strong> %s (%.2f%%)</p>
        <p><strong>Total Memory:</strong> %s</p>
    </div>`, hostStat.Hostname, runTimeOS, cpuInfo, humanizeBytes(float64(vmStat.Used)), usedMemoryPercent, humanizeBytes(float64(vmStat.Total)))

	return output, nil
}

// GetDiskSection provides information about disk partitions, showing used space and percentage
func GetDiskSection() (string, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return `<div class="error">Error fetching disk partitions: ` + err.Error() + `</div>`, err
	}

	// Sort partitions by mount point
	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].Mountpoint < partitions[j].Mountpoint
	})

	output := `<div class="disk-info">`
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue // Skip partitions we can't get usage info for
		}
		used := usage.Total - usage.Free
		usedPercent := float64(used) / float64(usage.Total) * 100.0
		output += fmt.Sprintf(`
            <p><strong>Mount Point:</strong> %s</p>
            <p><strong>Free Disk Space:</strong> %s</p>
            <p><strong>Used Disk Space:</strong> %s (%.2f%%)</p>
            <p><strong>Total Disk Space:</strong> %s</p>
        `, partition.Mountpoint, humanizeBytes(float64(usage.Free)), humanizeBytes(float64(used)), usedPercent, humanizeBytes(float64(usage.Total)))
	}
	output += `</div>`

	return output, nil
}
