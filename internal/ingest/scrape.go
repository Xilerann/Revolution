package ingest

import (
	"log"
	"time"

	"revolution/internal/scrape"
	"revolution/internal/store"
)

// ScrapeOnce interroge les trackers UDP (BEP 15) des torrents dus — jamais
// scrapés, ou scrapés il y a plus de `maxAge` — et enregistre les compteurs
// seeders/leechers renvoyés. Les torrents sans tracker, ou avec un tracker
// non-udp://, n'ont simplement pas de statistiques (voir internal/scrape).
//
// Les requêtes sont groupées par tracker (par lots d'au plus
// scrape.MaxInfoHashesPerRequest infohash) : un aller-retour UDP par lot,
// donc un coût réseau/CPU négligeable même pour un catalogue de plusieurs
// centaines de torrents.
func ScrapeOnce(st *store.Store, maxAge, timeout time.Duration) {
	targets, err := st.TorrentsDueForScrape(time.Now().Add(-maxAge).Unix())
	if err != nil {
		log.Printf("ingest: scrape: %v", err)
		return
	}
	if len(targets) == 0 {
		return
	}

	byTracker := make(map[string][]store.ScrapeTarget)
	for _, t := range targets {
		byTracker[t.Tracker] = append(byTracker[t.Tracker], t)
	}

	now := time.Now().Unix()
	scraped := 0
	for tracker, group := range byTracker {
		for i := 0; i < len(group); i += scrape.MaxInfoHashesPerRequest {
			batch := group[i:min(i+scrape.MaxInfoHashesPerRequest, len(group))]

			infoHashes := make([]string, len(batch))
			torrentIDByHash := make(map[string]string, len(batch))
			for j, t := range batch {
				infoHashes[j] = t.InfoHash
				torrentIDByHash[t.InfoHash] = t.TorrentID
			}

			stats, err := scrape.Scrape(tracker, infoHashes, timeout)
			if err != nil {
				log.Printf("ingest: scrape %s: %v", tracker, err)
				continue
			}
			for ih, s := range stats {
				torrentID, ok := torrentIDByHash[ih]
				if !ok {
					continue
				}
				if err := st.UpsertTorrentStats(torrentID, s.Seeders, s.Leechers, now); err != nil {
					log.Printf("ingest: scrape: %v", err)
					continue
				}
				scraped++
			}
		}
	}
	if scraped > 0 {
		log.Printf("ingest: scrape: %d torrent(s) mis à jour", scraped)
	}
}
