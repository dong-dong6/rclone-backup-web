// +build linux darwin

package services

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func readCPUTimes() (idle, total uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			var values []uint64
			for _, field := range fields[1:] {
				v, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					return 0, 0, err
				}
				values = append(values, v)
			}
			for _, v := range values {
				total += v
			}
			if len(values) >= 4 {
				idle = values[3]
			}
			return idle, total, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return 0, 0, fmt.Errorf("cpu stats not found")
}

func readMemInfo() (total, used uint64, usage float64, swapTotal, swapUsed uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	var memFree, buffers, cached, memAvailable uint64

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		value, parseErr := strconv.ParseUint(parts[1], 10, 64)
		if parseErr != nil {
			continue
		}
		value *= 1024
		switch key {
		case "MemTotal":
			total = value
		case "MemFree":
			memFree = value
		case "Buffers":
			buffers = value
		case "Cached":
			cached = value
		case "MemAvailable":
			memAvailable = value
		case "SwapTotal":
			swapTotal = value
		case "SwapFree":
			swapUsed = swapTotal - value
		}
	}

	if total == 0 {
		return 0, 0, 0, 0, 0, fmt.Errorf("memtotal not found")
	}

	if memAvailable > 0 {
		used = total - memAvailable
	} else {
		used = total - memFree - buffers - cached
	}

	usage = float64(used) / float64(total) * 100
	return
}

func getDiskUsage(path string) (total, used uint64, usage float64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return
	}

	total = stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used = total - free

	if total > 0 {
		usage = float64(used) / float64(total) * 100
	}
	return
}

func readNetIO() (rx, tx uint64, err error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// skip first two header lines
	for i := 0; i < 2 && scanner.Scan(); i++ {
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rxVal, err1 := strconv.ParseUint(fields[0], 10, 64)
		txVal, err2 := strconv.ParseUint(fields[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		rx += rxVal
		tx += txVal
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return rx, tx, nil
}

func countConnections(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "sl") {
			continue
		}
		count++
	}

	return count, scanner.Err()
}

func countProcesses() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err == nil {
			count++
		}
	}
	return count
}
