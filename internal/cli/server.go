package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"revolution/internal/config"
	"revolution/internal/ingest"
	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
	"revolution/internal/web"
)

// runServer démarre le worker d'ingestion Nostr et le serveur web public dans
// le même processus, et bloque jusqu'à réception de SIGINT/SIGTERM.
func runServer(cfg config.Config) error {
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("ouverture base %s: %w", cfg.DBPath, err)
	}
	defer st.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	nc := nostrclient.New(ctx)
	defer nc.Close()

	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second

	liveRefresh := time.Duration(cfg.LiveRefreshMinutes) * time.Minute
	if liveRefresh <= 0 {
		liveRefresh = 5 * time.Minute
	}
	reconcileInterval := time.Duration(cfg.ReconcileIntervalMinutes) * time.Minute
	if reconcileInterval <= 0 {
		reconcileInterval = 30 * time.Minute
	}

	// Mécanisme principal : souscription Nostr permanente (le relai pousse les
	// nouveaux événements dès leur publication, pas de sondage répété).
	go ingest.LiveSync(ctx, st, nc, liveRefresh)

	// Filet de sécurité : resynchronisation complète à intervalle large, pour
	// rattraper ce que le live aurait manqué (coupure, relai indisponible...).
	go func() {
		syncOnce(ctx, st, nc, timeout)
		ticker := time.NewTicker(reconcileInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncOnce(ctx, st, nc, timeout)
			}
		}
	}()

	if cfg.ScrapeEnabled {
		scrapeInterval := time.Duration(cfg.ScrapeIntervalMinutes) * time.Minute
		if scrapeInterval <= 0 {
			scrapeInterval = 30 * time.Minute
		}
		scrapeTimeout := time.Duration(cfg.ScrapeTimeoutSeconds) * time.Second
		go func() {
			ticker := time.NewTicker(scrapeInterval)
			defer ticker.Stop()
			ingest.ScrapeOnce(st, scrapeInterval, scrapeTimeout)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					ingest.ScrapeOnce(st, scrapeInterval, scrapeTimeout)
				}
			}
		}()
	}

	srv := web.New(cfg, st)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("revolution: web public sur http://%s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("revolution: signal reçu, arrêt en cours…")
	case err := <-errCh:
		log.Printf("revolution: erreur serveur web: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("revolution: arrêt serveur web: %v", err)
	}
	return nil
}

func syncOnce(ctx context.Context, st *store.Store, nc *nostrclient.Client, timeout time.Duration) {
	n, err := ingest.SyncAll(ctx, st, nc, timeout)
	if err != nil {
		log.Printf("revolution: ingestion: %v", err)
		return
	}
	log.Printf("revolution: ingestion terminée, %d torrent(s) reçu(s)", n)
}
