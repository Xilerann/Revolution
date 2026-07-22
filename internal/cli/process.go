package cli

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// readPid lit un pidfile et vérifie que le processus est toujours vivant
// (kill -0). Un pidfile orphelin (processus mort) est traité comme absent.
func readPid(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	if !processAlive(pid) {
		return 0, false
	}
	return pid, true
}

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// waitForExit attend jusqu'à `timeout` que le processus pid ne soit plus vivant.
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !processAlive(pid)
}
