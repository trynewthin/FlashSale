package catalog

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// SysInfo 宿主机系统资源快照。
type SysInfo struct {
	CPU    CPUInfo    `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
}

type CPUInfo struct {
	UsagePct float64 `json:"usage_pct"` // 0–100
	Cores    int     `json:"cores"`
}

type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePct       float64 `json:"usage_pct"` // 0–100
}

// ReadSysInfo 读取一次系统资源快照（Linux /proc 接口）。
func ReadSysInfo() SysInfo {
	return SysInfo{
		CPU:    readCPU(),
		Memory: readMemory(),
	}
}

// ─── CPU ───

type cpuStat struct{ idle, total uint64 }

func readCPUStat() cpuStat {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			break
		}
		var vals [10]uint64
		for i := 1; i < len(fields) && i <= 10; i++ {
			vals[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
		}
		// user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice
		idle := vals[3] + vals[4]
		total := vals[0] + vals[1] + vals[2] + vals[3] + vals[4] + vals[5] + vals[6] + vals[7]
		return cpuStat{idle: idle, total: total}
	}
	return cpuStat{}
}

// readCPU 采样两次（间隔 300ms）计算 CPU 使用率。
func readCPU() CPUInfo {
	s1 := readCPUStat()
	time.Sleep(300 * time.Millisecond)
	s2 := readCPUStat()

	totalDiff := float64(s2.total - s1.total)
	idleDiff := float64(s2.idle - s1.idle)

	var usage float64
	if totalDiff > 0 {
		usage = (1 - idleDiff/totalDiff) * 100
	}

	return CPUInfo{
		UsagePct: roundPct(usage),
		Cores:    countCores(),
	}
}

func countCores() int {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 1
	}
	defer f.Close()
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// ─── Memory ───

func readMemory() MemoryInfo {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}
	}
	defer f.Close()

	kv := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		kv[key] = val // kB
	}

	totalKB := kv["MemTotal"]
	availKB := kv["MemAvailable"]
	usedKB := totalKB - availKB

	var usagePct float64
	if totalKB > 0 {
		usagePct = float64(usedKB) / float64(totalKB) * 100
	}

	return MemoryInfo{
		TotalBytes:     totalKB * 1024,
		AvailableBytes: availKB * 1024,
		UsagePct:       roundPct(usagePct),
	}
}

func roundPct(v float64) float64 {
	f, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", v), 64)
	return f
}
