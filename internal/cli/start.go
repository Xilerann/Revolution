package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func cmdStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	foreground := fs.Bool("foreground", false, "reste au premier plan (utilisé par systemd ou en interne par le mode arrière-plan)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if *foreground {
		return runServer(cfg)
	}

	if pid, alive := readPid(DefaultPidPath); alive {
		return fmt.Errorf("revolution est déjà lancé (pid %d) ; utilisez `revolution stop` d'abord", pid)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("localisation du binaire: %w", err)
	}

	logFile, err := os.OpenFile(DefaultLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("ouverture log %s: %w", DefaultLogPath, err)
	}
	defer logFile.Close()

	child := exec.Command(exe, "start", "--foreground")
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := child.Start(); err != nil {
		return fmt.Errorf("démarrage: %w", err)
	}
	if err := os.WriteFile(DefaultPidPath, []byte(strconv.Itoa(child.Process.Pid)), 0o644); err != nil {
		return fmt.Errorf("écriture pidfile: %w", err)
	}

	fmt.Printf("revolution démarré (pid %d) sur http://%s — logs: %s\n", child.Process.Pid, cfg.ListenAddr, DefaultLogPath)
	return nil
}
