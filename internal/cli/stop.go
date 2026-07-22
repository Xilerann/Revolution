package cli

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func cmdStop(args []string) error {
	pid, alive := readPid(DefaultPidPath)
	if !alive {
		return fmt.Errorf("revolution n'est pas lancé (ou le pidfile %s est absent/obsolète)", DefaultPidPath)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("processus %d introuvable: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("envoi SIGTERM à %d: %w", pid, err)
	}

	if !waitForExit(pid, 10*time.Second) {
		return fmt.Errorf("le processus %d ne s'est pas arrêté à temps (10s)", pid)
	}

	_ = os.Remove(DefaultPidPath)
	fmt.Println("revolution arrêté")
	return nil
}
