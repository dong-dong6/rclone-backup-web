// +build windows

package services

// Windows implementations - return stub values since Windows agents
// are not the primary target platform

func readCPUTimes() (idle, total uint64, err error) {
	// Windows: return zero values, CPU monitoring not implemented
	return 0, 0, nil
}

func readMemInfo() (total, used uint64, usage float64, swapTotal, swapUsed uint64, err error) {
	// Windows: return zero values, memory monitoring not implemented
	return 0, 0, 0, 0, 0, nil
}

func getDiskUsage(path string) (total, used uint64, usage float64, err error) {
	// Windows: return zero values, disk monitoring not implemented
	return 0, 0, 0, nil
}

func readNetIO() (rx, tx uint64, err error) {
	// Windows: return zero values, network monitoring not implemented
	return 0, 0, nil
}

func countConnections(path string) (int, error) {
	// Windows: return zero, connection counting not implemented
	return 0, nil
}

func countProcesses() int {
	// Windows: return zero, process counting not implemented
	return 0
}
