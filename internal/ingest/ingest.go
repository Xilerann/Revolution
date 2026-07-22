// Package ingest orchestre la récupération des torrents Nostr des comptes
// suivis et leur écriture dans le catalogue local.
package ingest

import (
	"context"
	"fmt"
	"log"
	"time"

	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
)

// SyncUser récupère (depuis le dernier point connu) tous les torrents et le
// profil d'un compte suivi, et met à jour le catalogue. Renvoie le nombre de
// torrents reçus.
func SyncUser(ctx context.Context, st *store.Store, cl *nostrclient.Client, relays []string, f store.Followed, timeout time.Duration) (int, error) {
	if len(relays) == 0 {
		return 0, fmt.Errorf("aucun relai configuré")
	}

	torrents := cl.FetchTorrents(ctx, relays, []string{f.Pubkey}, f.LastSyncedAt, timeout)

	maxTs := f.LastSyncedAt
	for _, t := range torrents {
		if err := st.UpsertTorrent(t); err != nil {
			return 0, fmt.Errorf("enregistrement torrent %s: %w", t.ID, err)
		}
		if t.PublishedAt > maxTs {
			maxTs = t.PublishedAt
		}
	}
	if maxTs > f.LastSyncedAt {
		if err := st.UpdateLastSynced(f.Pubkey, maxTs); err != nil {
			return len(torrents), err
		}
	}

	for _, p := range cl.FetchProfiles(ctx, relays, []string{f.Pubkey}, timeout) {
		if err := st.UpsertProfile(p); err != nil {
			log.Printf("ingest: maj profil %s: %v", p.Pubkey, err)
		}
	}

	return len(torrents), nil
}

// SyncAll synchronise tous les comptes suivis actifs sur tous les relais actifs.
// Renvoie le nombre total de torrents reçus (nouveaux ou mis à jour) et continue
// sur les autres comptes si l'un d'eux échoue.
func SyncAll(ctx context.Context, st *store.Store, cl *nostrclient.Client, timeout time.Duration) (int, error) {
	relays, err := st.ListEnabledRelayURLs()
	if err != nil {
		return 0, err
	}
	if len(relays) == 0 {
		return 0, fmt.Errorf("aucun relai configuré (voir `revolution relay add`)")
	}

	followed, err := st.ListFollowed()
	if err != nil {
		return 0, err
	}

	total := 0
	for _, f := range followed {
		if !f.Enabled {
			continue
		}
		n, err := SyncUser(ctx, st, cl, relays, f, timeout)
		if err != nil {
			log.Printf("ingest: sync %s (%s): %v", f.Alias, f.Pubkey, err)
			continue
		}
		total += n
	}
	return total, nil
}
