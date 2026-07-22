package cli

import (
	"context"
	"fmt"
	"time"

	"revolution/internal/ingest"
	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
)

func cmdSync(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout*20)
	defer cancel()

	nc := nostrclient.New(ctx)
	defer nc.Close()

	n, err := ingest.SyncAll(ctx, st, nc, timeout)
	if err != nil {
		return err
	}

	fmt.Printf("%d torrent(s) synchronisé(s) au total\n", n)
	return nil
}
