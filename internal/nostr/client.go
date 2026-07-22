package nostrclient

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"

	"revolution/internal/store"
)

// Client est un pool de connexions aux relais Nostr configurés par le modérateur.
type Client struct {
	pool *nostr.SimplePool
}

// New crée un client Nostr lié au contexte donné (fermé quand ctx est annulé).
func New(ctx context.Context) *Client {
	return &Client{pool: nostr.NewSimplePool(ctx)}
}

// Close ferme toutes les connexions relais du pool.
func (c *Client) Close() {
	c.pool.Close("arrêt")
}

// FetchTorrents récupère, sur tous les relais donnés, les événements kind 2003
// des auteurs donnés publiés après `since` (0 = depuis toujours), valide leur
// signature et renvoie les torrents parsés avec succès.
func (c *Client) FetchTorrents(ctx context.Context, relays, authors []string, since int64, timeout time.Duration) []store.Torrent {
	if len(relays) == 0 || len(authors) == 0 {
		return nil
	}

	fctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	filter := nostr.Filter{Kinds: []int{KindTorrent}, Authors: authors}
	if since > 0 {
		ts := nostr.Timestamp(since)
		filter.Since = &ts
	}

	seen := make(map[string]bool)
	var out []store.Torrent
	for re := range c.pool.FetchMany(fctx, relays, filter) {
		if seen[re.Event.ID] {
			continue
		}
		seen[re.Event.ID] = true

		ok, err := re.Event.CheckSignature()
		if err != nil || !ok {
			log.Printf("nostr: signature invalide pour événement %s, ignoré", re.Event.ID)
			continue
		}

		t, err := ParseTorrent(re.Event)
		if err != nil {
			log.Printf("nostr: %v", err)
			continue
		}
		out = append(out, t)
	}
	return out
}

// LiveEvent est soit un torrent soit un profil reçu au fil de l'eau par Live.
// Exactement un des deux champs est non-nil.
type LiveEvent struct {
	Torrent *store.Torrent
	Profile *store.Profile
}

// Live ouvre une souscription Nostr persistante (pas de EOSE : reste ouverte
// tant que ctx n'est pas annulé) sur les événements kind 2003 (torrent) et
// kind 0 (profil) des auteurs donnés, et pousse chaque événement valide sur
// le channel renvoyé. Une souscription persistante pousse les nouveaux
// événements dès leur publication, sans que le client ait besoin de
// ré-interroger le relai : c'est plus léger pour le relai qu'un sondage
// périodique fréquent. Le channel se ferme quand ctx est annulé.
func (c *Client) Live(ctx context.Context, relays, authors []string) <-chan LiveEvent {
	out := make(chan LiveEvent)
	if len(relays) == 0 || len(authors) == 0 {
		close(out)
		return out
	}

	// Léger recouvrement en arrière pour ne pas perdre d'événement publié
	// pile au moment où une souscription précédente vient de se fermer
	// (ex: reconfiguration du filtre après un `follow add`).
	since := nostr.Timestamp(time.Now().Add(-2 * time.Minute).Unix())
	filter := nostr.Filter{Kinds: []int{KindTorrent, KindProfile}, Authors: authors, Since: &since}

	go func() {
		defer close(out)
		for re := range c.pool.SubscribeMany(ctx, relays, filter) {
			ok, err := re.Event.CheckSignature()
			if err != nil || !ok {
				continue
			}

			switch re.Event.Kind {
			case KindTorrent:
				t, err := ParseTorrent(re.Event)
				if err != nil {
					log.Printf("nostr: live: %v", err)
					continue
				}
				select {
				case out <- LiveEvent{Torrent: &t}:
				case <-ctx.Done():
					return
				}

			case KindProfile:
				var meta ProfileMeta
				if err := json.Unmarshal([]byte(re.Event.Content), &meta); err != nil {
					continue
				}
				p := store.Profile{
					Pubkey:      re.Event.PubKey,
					Name:        meta.Name,
					DisplayName: meta.DisplayName,
					Picture:     meta.Picture,
					NIP05:       meta.NIP05,
					UpdatedAt:   int64(re.Event.CreatedAt),
				}
				select {
				case out <- LiveEvent{Profile: &p}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}

// FetchProfiles récupère les métadonnées de profil (kind 0, la version la plus
// récente par auteur) pour les pubkeys données.
//
// Note : on utilise volontairement FetchMany (brut) plutôt que
// SimplePool.FetchManyReplaceable — cette dernière, dans go-nostr v0.52.3,
// écarte silencieusement l'événement le plus récent au lieu des plus anciens
// (le filtre de duplication interne à la lib a une logique inversée pour les
// événements "replaceable" simples comme le kind 0). On déduplique donc
// nous-mêmes en gardant, par pubkey, l'événement au created_at le plus grand ;
// Store.UpsertProfile refait de toute façon cette même vérification côté SQL.
func (c *Client) FetchProfiles(ctx context.Context, relays, authors []string, timeout time.Duration) []store.Profile {
	if len(relays) == 0 || len(authors) == 0 {
		return nil
	}

	fctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	filter := nostr.Filter{Kinds: []int{KindProfile}, Authors: authors}

	latest := make(map[string]*nostr.Event)
	for re := range c.pool.FetchMany(fctx, relays, filter) {
		ok, err := re.Event.CheckSignature()
		if err != nil || !ok {
			continue
		}
		if cur, seen := latest[re.Event.PubKey]; !seen || re.Event.CreatedAt > cur.CreatedAt {
			latest[re.Event.PubKey] = re.Event
		}
	}

	out := make([]store.Profile, 0, len(latest))
	for _, evt := range latest {
		var meta ProfileMeta
		if err := json.Unmarshal([]byte(evt.Content), &meta); err != nil {
			continue
		}
		out = append(out, store.Profile{
			Pubkey:      evt.PubKey,
			Name:        meta.Name,
			DisplayName: meta.DisplayName,
			Picture:     meta.Picture,
			NIP05:       meta.NIP05,
			UpdatedAt:   int64(evt.CreatedAt),
		})
	}
	return out
}
