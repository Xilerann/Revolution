// Package store gère la persistance SQLite du catalogue de torrents.
//
// Choix : modernc.org/sqlite (implémentation pure Go, sans CGO) pour garder un
// binaire statique facile à cross-compiler et à déployer sur une petite machine.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store encapsule la connexion SQLite et les opérations du catalogue.
type Store struct {
	db *sql.DB
}

// Open ouvre (et crée si besoin) la base SQLite au chemin donné, active le
// mode WAL (lectures concurrentes pendant l'ingestion) et applique le schéma.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("ouverture sqlite: %w", err)
	}
	// Le catalogue est un usage mono-écrivain (ingestion) + multi-lecteurs (web) ;
	// une seule connexion d'écriture évite les erreurs SQLITE_BUSY sous charge.
	db.SetMaxOpenConns(4)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close ferme la base après un dernier checkpoint WAL.
func (s *Store) Close() error {
	_, _ = s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return s.db.Close()
}

const schema = `
CREATE TABLE IF NOT EXISTS relays (
	url TEXT PRIMARY KEY,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS followed (
	pubkey TEXT PRIMARY KEY,
	alias TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	last_synced_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS profiles (
	pubkey TEXT PRIMARY KEY,
	name TEXT NOT NULL DEFAULT '',
	display_name TEXT NOT NULL DEFAULT '',
	picture TEXT NOT NULL DEFAULT '',
	nip05 TEXT NOT NULL DEFAULT '',
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS torrents (
	id TEXT PRIMARY KEY,
	pubkey TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	infohash TEXT NOT NULL,
	magnet TEXT NOT NULL,
	tracker TEXT NOT NULL DEFAULT '',
	image_url TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT '',
	refs TEXT NOT NULL DEFAULT '[]',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	published_at INTEGER NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_torrents_pubkey ON torrents(pubkey);
CREATE INDEX IF NOT EXISTS idx_torrents_published_at ON torrents(published_at DESC);

CREATE TABLE IF NOT EXISTS torrent_files (
	torrent_id TEXT NOT NULL REFERENCES torrents(id) ON DELETE CASCADE,
	path TEXT NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_torrent_files_torrent_id ON torrent_files(torrent_id);

-- Statistiques de tracker (BEP 15, scrape UDP), mises à jour périodiquement
-- en arrière-plan. Absent de cette table = jamais scrapé (pas de tracker udp://
-- connu, ou pas encore atteint son tour) ; le site n'affiche alors rien plutôt
-- qu'un zéro trompeur.
CREATE TABLE IF NOT EXISTS torrent_stats (
	torrent_id TEXT PRIMARY KEY REFERENCES torrents(id) ON DELETE CASCADE,
	seeders INTEGER NOT NULL DEFAULT 0,
	leechers INTEGER NOT NULL DEFAULT 0,
	scraped_at INTEGER NOT NULL DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS torrents_fts USING fts5(
	title, description, category,
	content='torrents', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS torrents_ai AFTER INSERT ON torrents BEGIN
	INSERT INTO torrents_fts(rowid, title, description, category)
	VALUES (new.rowid, new.title, new.description, new.category);
END;

CREATE TRIGGER IF NOT EXISTS torrents_ad AFTER DELETE ON torrents BEGIN
	INSERT INTO torrents_fts(torrents_fts, rowid, title, description, category)
	VALUES('delete', old.rowid, old.title, old.description, old.category);
END;

CREATE TRIGGER IF NOT EXISTS torrents_au AFTER UPDATE ON torrents BEGIN
	INSERT INTO torrents_fts(torrents_fts, rowid, title, description, category)
	VALUES('delete', old.rowid, old.title, old.description, old.category);
	INSERT INTO torrents_fts(rowid, title, description, category)
	VALUES (new.rowid, new.title, new.description, new.category);
END;

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migration schéma: %w", err)
	}
	return nil
}

func now() int64 { return time.Now().Unix() }
