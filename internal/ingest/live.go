package ingest

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
)

// LiveSync maintient une souscription Nostr permanente sur les comptes
// suivis actifs : c'est le mécanisme principal de mise à jour du catalogue.
// Une souscription persistante (le relai pousse lui-même les nouveaux
// événements) est nettement plus légère pour un relai public/gratuit qu'un
// sondage périodique fréquent qui réémettrait sans cesse les mêmes requêtes.
// SyncAll (voir ingest.go), appelé à intervalle plus large par l'appelant,
// reste un filet de sécurité pour rattraper ce qui aurait été manqué pendant
// une coupure de connexion.
//
// La liste des comptes suivis / relais est relue en base toutes les
// `refreshEvery` : si elle a changé depuis (follow/relay add|rm exécutés
// pendant que le serveur tourne), la souscription est fermée et rouverte
// avec le nouveau filtre. Bloque jusqu'à annulation de ctx.
func LiveSync(ctx context.Context, st *store.Store, cl *nostrclient.Client, refreshEvery time.Duration) {
	var subCancel context.CancelFunc
	var lastKey string
	defer func() {
		if subCancel != nil {
			subCancel()
		}
	}()

	reload := func() {
		relays, err := st.ListEnabledRelayURLs()
		if err != nil {
			log.Printf("ingest: live: %v", err)
			return
		}
		followedList, err := st.ListFollowed()
		if err != nil {
			log.Printf("ingest: live: %v", err)
			return
		}

		var authors []string
		for _, f := range followedList {
			if f.Enabled {
				authors = append(authors, f.Pubkey)
			}
		}
		sort.Strings(authors)
		sort.Strings(relays)
		key := strings.Join(relays, ",") + "|" + strings.Join(authors, ",")
		if key == lastKey {
			return
		}
		lastKey = key

		if subCancel != nil {
			subCancel()
			subCancel = nil
		}
		if len(relays) == 0 || len(authors) == 0 {
			return
		}

		var subCtx context.Context
		subCtx, subCancel = context.WithCancel(ctx)
		go consumeLive(subCtx, st, cl, relays, authors)
	}

	reload()

	ticker := time.NewTicker(refreshEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reload()
		}
	}
}

func consumeLive(ctx context.Context, st *store.Store, cl *nostrclient.Client, relays, authors []string) {
	for ev := range cl.Live(ctx, relays, authors) {
		switch {
		case ev.Torrent != nil:
			if err := st.UpsertTorrent(*ev.Torrent); err != nil {
				log.Printf("ingest: live: upsert torrent: %v", err)
				continue
			}
			if err := st.UpdateLastSynced(ev.Torrent.Pubkey, ev.Torrent.PublishedAt); err != nil {
				log.Printf("ingest: live: maj last_synced_at: %v", err)
			}
			log.Printf("ingest: live: nouveau torrent %q (%s)", ev.Torrent.Title, ev.Torrent.ID)

		case ev.Profile != nil:
			if err := st.UpsertProfile(*ev.Profile); err != nil {
				log.Printf("ingest: live: maj profil: %v", err)
			}
		}
	}
}
