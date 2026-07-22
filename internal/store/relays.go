package store

import "fmt"

// Relay est un relai Nostr (généralement gratuit/public) interrogé pour
// récupérer les torrents des comptes suivis.
type Relay struct {
	URL     string
	Enabled bool
}

// AddRelay ajoute (ou réactive) un relai.
func (s *Store) AddRelay(url string) error {
	_, err := s.db.Exec(`
		INSERT INTO relays (url, enabled, created_at) VALUES (?, 1, ?)
		ON CONFLICT(url) DO UPDATE SET enabled=1
	`, url, now())
	if err != nil {
		return fmt.Errorf("ajout relai: %w", err)
	}
	return nil
}

// RemoveRelay retire un relai de la configuration.
func (s *Store) RemoveRelay(url string) error {
	res, err := s.db.Exec(`DELETE FROM relays WHERE url = ?`, url)
	if err != nil {
		return fmt.Errorf("suppression relai: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRelays renvoie tous les relais configurés.
func (s *Store) ListRelays() ([]Relay, error) {
	rows, err := s.db.Query(`SELECT url, enabled FROM relays ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("liste relais: %w", err)
	}
	defer rows.Close()

	var out []Relay
	for rows.Next() {
		var r Relay
		var enabled int
		if err := rows.Scan(&r.URL, &enabled); err != nil {
			return nil, err
		}
		r.Enabled = enabled != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListEnabledRelayURLs renvoie uniquement les URLs des relais actifs.
func (s *Store) ListEnabledRelayURLs() ([]string, error) {
	rows, err := s.db.Query(`SELECT url FROM relays WHERE enabled = 1 ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("liste relais actifs: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		out = append(out, url)
	}
	return out, rows.Err()
}
