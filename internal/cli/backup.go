package cli

import (
	"fmt"
	"time"

	"revolution/internal/store"
)

func cmdBackup(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: revolution backup <chemin-destination.db>")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	start := time.Now()
	if err := st.BackupTo(args[0]); err != nil {
		return err
	}
	fmt.Printf("catalogue sauvegardé vers %s (%s)\n", args[0], time.Since(start).Round(time.Millisecond))
	return nil
}
