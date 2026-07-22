package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Followed est un compte Nostr suivi par ce modérateur : ses torrents (kind 2003)
// sont récupérés automatiquement et ajoutés au catalogue local.
type Followed struct {
	Pubkey       string
	Alias        string
	Enabled      bool
	LastSyncedAt int64
	CreatedAt    int64
}

// AddFollowed ajoute (ou réactive) un compte suivi.
func (s *Store) AddFollowed(pubkey, alias string) error {
	_, err := s.db.Exec(`
		INSERT INTO followed (pubkey, alias, enabled, last_synced_at, created_at)
		VALUES (?, ?, 1, 0, ?)
		ON CONFLICT(pubkey) DO UPDATE SET alias=excluded.alias, enabled=1
	`, pubkey, alias, now())
	if err != nil {
		return fmt.Errorf("ajout compte suivi: %w", err)
	}
	return nil
}

// RemoveFollowed retire un compte de la liste de suivi (ne supprime pas ses
// torrents déjà catalogués — voir Store.DeleteByPubkey pour ça).
func (s *Store) RemoveFollowed(pubkey string) error {
	res, err := s.db.Exec(`DELETE FROM followed WHERE pubkey = ?`, pubkey)
	if err != nil {
		return fmt.Errorf("suppression compte suivi: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListFollowed renvoie tous les comptes suivis (actifs et inactifs).
func (s *Store) ListFollowed() ([]Followed, error) {
	rows, err := s.db.Query(`SELECT pubkey, alias, enabled, last_synced_at, created_at FROM followed ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("liste comptes suivis: %w", err)
	}
	defer rows.Close()

	var out []Followed
	for rows.Next() {
		var f Followed
		var enabled int
		if err := rows.Scan(&f.Pubkey, &f.Alias, &enabled, &f.LastSyncedAt, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Enabled = enabled != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

// FindFollowed résout un alias ou une pubkey hex vers l'entrée Followed correspondante.
func (s *Store) FindFollowed(aliasOrPubkey string) (Followed, error) {
	var f Followed
	var enabled int
	row := s.db.QueryRow(`SELECT pubkey, alias, enabled, last_synced_at, created_at
		FROM followed WHERE pubkey = ? OR alias = ?`, aliasOrPubkey, aliasOrPubkey)
	if err := row.Scan(&f.Pubkey, &f.Alias, &enabled, &f.LastSyncedAt, &f.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return f, ErrNotFound
		}
		return f, fmt.Errorf("résolution compte suivi: %w", err)
	}
	f.Enabled = enabled != 0
	return f, nil
}

// UpdateLastSynced enregistre le timestamp du dernier événement vu pour ce
// compte, utilisé comme borne `since` lors du prochain fetch incrémental.
// N'avance jamais en arrière : la synchronisation périodique (par lot) et la
// souscription live tournent en parallèle et peuvent toutes deux appeler
// cette méthode pour le même compte, dans un ordre non garanti.
func (s *Store) UpdateLastSynced(pubkey string, ts int64) error {
	_, err := s.db.Exec(`UPDATE followed SET last_synced_at = MAX(last_synced_at, ?) WHERE pubkey = ?`, ts, pubkey)
	if err != nil {
		return fmt.Errorf("maj last_synced_at: %w", err)
	}
	return nil
}
