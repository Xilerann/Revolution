package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// TorrentStats est le dernier résultat de scrape (BEP 15) connu pour un torrent.
type TorrentStats struct {
	Seeders   int
	Leechers  int
	ScrapedAt int64
}

// UpsertTorrentStats enregistre le résultat d'un scrape pour un torrent.
func (s *Store) UpsertTorrentStats(torrentID string, seeders, leechers int, scrapedAt int64) error {
	_, err := s.db.Exec(`
		INSERT INTO torrent_stats (torrent_id, seeders, leechers, scraped_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(torrent_id) DO UPDATE SET
			seeders=excluded.seeders, leechers=excluded.leechers, scraped_at=excluded.scraped_at
	`, torrentID, seeders, leechers, scrapedAt)
	if err != nil {
		return fmt.Errorf("maj stats torrent %s: %w", torrentID, err)
	}
	return nil
}

// GetTorrentStats lit le dernier résultat de scrape connu pour un torrent.
// Renvoie ErrNotFound si ce torrent n'a jamais été scrapé (pas de tracker
// udp:// connu, ou pas encore atteint son tour).
func (s *Store) GetTorrentStats(torrentID string) (TorrentStats, error) {
	var st TorrentStats
	row := s.db.QueryRow(`SELECT seeders, leechers, scraped_at FROM torrent_stats WHERE torrent_id = ?`, torrentID)
	if err := row.Scan(&st.Seeders, &st.Leechers, &st.ScrapedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return st, ErrNotFound
		}
		return st, fmt.Errorf("lecture stats torrent %s: %w", torrentID, err)
	}
	return st, nil
}

// ScrapeTarget est un torrent candidat à un scrape de tracker.
type ScrapeTarget struct {
	TorrentID string
	InfoHash  string
	Tracker   string
}

// TorrentsDueForScrape renvoie les torrents ayant un tracker connu et jamais
// scrapés, ou scrapés avant le timestamp `olderThan`.
func (s *Store) TorrentsDueForScrape(olderThan int64) ([]ScrapeTarget, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.infohash, t.tracker
		FROM torrents t
		LEFT JOIN torrent_stats ts ON ts.torrent_id = t.id
		WHERE t.tracker != '' AND (ts.scraped_at IS NULL OR ts.scraped_at < ?)
	`, olderThan)
	if err != nil {
		return nil, fmt.Errorf("liste torrents à scraper: %w", err)
	}
	defer rows.Close()

	var out []ScrapeTarget
	for rows.Next() {
		var t ScrapeTarget
		if err := rows.Scan(&t.TorrentID, &t.InfoHash, &t.Tracker); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
