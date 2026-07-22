package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotFound est renvoyé quand une entité n'existe pas dans le catalogue.
var ErrNotFound = errors.New("introuvable")

// TorrentFile est une entrée du tag `file` NIP-35 (chemin + taille).
type TorrentFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// Torrent est un enregistrement du catalogue, dérivé d'un événement Nostr kind 2003.
type Torrent struct {
	ID          string // id de l'événement Nostr (hex)
	Pubkey      string // pubkey du publicateur (hex)
	Title       string
	Description string
	InfoHash    string
	Magnet      string
	Tracker     string
	ImageURL    string
	Category    string   // tags `t` joints par des virgules, pour affichage
	Refs        []string // tags `i` (imdb:..., tmdb:..., tcat:..., ...)
	SizeBytes   int64
	Files       []TorrentFile
	PublishedAt int64 // created_at de l'événement Nostr
}

// TorrentSummary est la vue allégée utilisée dans les résultats de recherche.
type TorrentSummary struct {
	ID            string
	Title         string
	Category      string
	SizeBytes     int64
	PublishedAt   int64
	Pubkey        string
	PublisherName string
	HasStats      bool // stats de tracker (BEP15) connues pour ce torrent
	Seeders       int
	Leechers      int
}

// UpsertTorrent insère ou met à jour un torrent et ses fichiers associés.
// Idempotent : rejouer le même événement Nostr ne duplique rien (clé = id événement).
func (s *Store) UpsertTorrent(t Torrent) error {
	refsJSON, err := json.Marshal(t.Refs)
	if err != nil {
		return fmt.Errorf("sérialisation refs: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	ts := now()
	_, err = tx.Exec(`
		INSERT INTO torrents (id, pubkey, title, description, infohash, magnet, tracker,
			image_url, category, refs, size_bytes, published_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			description=excluded.description,
			infohash=excluded.infohash,
			magnet=excluded.magnet,
			tracker=excluded.tracker,
			image_url=excluded.image_url,
			category=excluded.category,
			refs=excluded.refs,
			size_bytes=excluded.size_bytes,
			published_at=excluded.published_at,
			updated_at=excluded.updated_at
	`, t.ID, t.Pubkey, t.Title, t.Description, t.InfoHash, t.Magnet, t.Tracker,
		t.ImageURL, t.Category, string(refsJSON), t.SizeBytes, t.PublishedAt, ts, ts)
	if err != nil {
		return fmt.Errorf("upsert torrent: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM torrent_files WHERE torrent_id = ?`, t.ID); err != nil {
		return fmt.Errorf("purge fichiers: %w", err)
	}
	for _, f := range t.Files {
		if _, err := tx.Exec(`INSERT INTO torrent_files (torrent_id, path, size_bytes) VALUES (?, ?, ?)`,
			t.ID, f.Path, f.SizeBytes); err != nil {
			return fmt.Errorf("insert fichier: %w", err)
		}
	}

	return tx.Commit()
}

// GetTorrent charge un torrent complet (avec ses fichiers) par id d'événement.
func (s *Store) GetTorrent(id string) (Torrent, error) {
	var t Torrent
	var refsJSON string
	row := s.db.QueryRow(`
		SELECT id, pubkey, title, description, infohash, magnet, tracker, image_url,
			category, refs, size_bytes, published_at
		FROM torrents WHERE id = ?`, id)
	if err := row.Scan(&t.ID, &t.Pubkey, &t.Title, &t.Description, &t.InfoHash, &t.Magnet,
		&t.Tracker, &t.ImageURL, &t.Category, &refsJSON, &t.SizeBytes, &t.PublishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, ErrNotFound
		}
		return t, fmt.Errorf("lecture torrent: %w", err)
	}
	_ = json.Unmarshal([]byte(refsJSON), &t.Refs)

	rows, err := s.db.Query(`SELECT path, size_bytes FROM torrent_files WHERE torrent_id = ?`, id)
	if err != nil {
		return t, fmt.Errorf("lecture fichiers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f TorrentFile
		if err := rows.Scan(&f.Path, &f.SizeBytes); err != nil {
			return t, err
		}
		t.Files = append(t.Files, f)
	}
	return t, rows.Err()
}

// DeleteTorrent supprime un torrent unique du catalogue (modération manuelle).
func (s *Store) DeleteTorrent(id string) error {
	res, err := s.db.Exec(`DELETE FROM torrents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression torrent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateTorrentFields permet au modérateur de corriger titre/catégorie localement.
func (s *Store) UpdateTorrentFields(id, title, category string) error {
	res, err := s.db.Exec(`UPDATE torrents SET title = ?, category = ?, updated_at = ? WHERE id = ?`,
		title, category, now(), id)
	if err != nil {
		return fmt.Errorf("mise à jour torrent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountByPubkey renvoie le nombre de torrents actuellement catalogués pour un auteur.
func (s *Store) CountByPubkey(pubkey string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM torrents WHERE pubkey = ?`, pubkey).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("comptage torrents: %w", err)
	}
	return n, nil
}

// DeleteByPubkey supprime tous les torrents publiés par un auteur donné
// (ex: le modérateur ne veut plus du feed de ce compte). Renvoie le nombre supprimé.
func (s *Store) DeleteByPubkey(pubkey string) (int, error) {
	res, err := s.db.Exec(`DELETE FROM torrents WHERE pubkey = ?`, pubkey)
	if err != nil {
		return 0, fmt.Errorf("suppression torrents: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Search effectue une recherche plein texte (FTS5) sur titre/description/catégorie,
// et joint le nom du publicateur si connu en cache.
func (s *Store) Search(query string, limit, offset int) ([]TorrentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}

	var rows *sql.Rows
	var err error
	if query == "" {
		rows, err = s.db.Query(`
			SELECT t.id, t.title, t.category, t.size_bytes, t.published_at, t.pubkey,
				COALESCE(NULLIF(p.display_name, ''), NULLIF(p.name, ''), ''),
				ts.seeders, ts.leechers, ts.torrent_id IS NOT NULL
			FROM torrents t
			LEFT JOIN profiles p ON p.pubkey = t.pubkey
			LEFT JOIN torrent_stats ts ON ts.torrent_id = t.id
			ORDER BY t.published_at DESC
			LIMIT ? OFFSET ?`, limit, offset)
	} else {
		rows, err = s.db.Query(`
			SELECT t.id, t.title, t.category, t.size_bytes, t.published_at, t.pubkey,
				COALESCE(NULLIF(p.display_name, ''), NULLIF(p.name, ''), ''),
				ts.seeders, ts.leechers, ts.torrent_id IS NOT NULL
			FROM torrents_fts f
			JOIN torrents t ON t.rowid = f.rowid
			LEFT JOIN profiles p ON p.pubkey = t.pubkey
			LEFT JOIN torrent_stats ts ON ts.torrent_id = t.id
			WHERE torrents_fts MATCH ?
			ORDER BY t.published_at DESC
			LIMIT ? OFFSET ?`, ftsQuery(query), limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("recherche: %w", err)
	}
	defer rows.Close()

	var out []TorrentSummary
	for rows.Next() {
		var ts TorrentSummary
		var seeders, leechers sql.NullInt64
		if err := rows.Scan(&ts.ID, &ts.Title, &ts.Category, &ts.SizeBytes, &ts.PublishedAt,
			&ts.Pubkey, &ts.PublisherName, &seeders, &leechers, &ts.HasStats); err != nil {
			return nil, err
		}
		ts.Seeders = int(seeders.Int64)
		ts.Leechers = int(leechers.Int64)
		out = append(out, ts)
	}
	return out, rows.Err()
}

// ftsQuery transforme une requête utilisateur libre en requête FTS5 sûre :
// chaque mot est mis entre guillemets et traité comme un préfixe, ce qui évite
// toute interprétation de la syntaxe FTS5 (colonnes, opérateurs) par l'utilisateur.
func ftsQuery(q string) string {
	var out string
	word := ""
	flush := func() {
		if word != "" {
			if out != "" {
				out += " "
			}
			out += `"` + word + `"*`
			word = ""
		}
	}
	for _, r := range q {
		if r == '"' {
			continue // évite d'échapper les guillemets internes
		}
		if r == ' ' || r == '\t' || r == '\n' {
			flush()
			continue
		}
		word += string(r)
	}
	flush()
	if out == "" {
		out = `""`
	}
	return out
}
