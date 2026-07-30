package host

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Capacity represents the physical host VM capacity.
type Capacity struct {
	TotalRAMBytes int64 `json:"total_ram_bytes"`
	TotalCPUCores int   `json:"total_cpu_cores"`
}

// GetCapacity returns the host VM capacity. It reads /proc/meminfo on Linux.
func GetCapacity() Capacity {
	cap := Capacity{
		TotalCPUCores: runtime.NumCPU(),
	}

	// Read /proc/meminfo for Total RAM (works on Linux, which is our target for MyPaas)
	f, err := os.Open("/proc/meminfo")
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						cap.TotalRAMBytes = kb * 1024
					}
				}
				break
			}
		}
	}

	// Fallback if not on Linux or reading failed (e.g. developing on macOS/Windows)
	if cap.TotalRAMBytes == 0 {
		// Just a dummy 8GB fallback for local dev
		cap.TotalRAMBytes = 8 * 1024 * 1024 * 1024 
	}

	return cap
}
