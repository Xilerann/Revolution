package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Profile est un cache local d'un événement Nostr kind 0 (métadonnées de profil).
type Profile struct {
	Pubkey      string
	Name        string
	DisplayName string
	Picture     string
	NIP05       string
	UpdatedAt   int64
}

// UpsertProfile met à jour le cache de profil si l'événement reçu est plus récent
// que celui déjà en base (les kind 0 sont "replaceable" côté Nostr).
func (s *Store) UpsertProfile(p Profile) error {
	_, err := s.db.Exec(`
		INSERT INTO profiles (pubkey, name, display_name, picture, nip05, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			name=excluded.name,
			display_name=excluded.display_name,
			picture=excluded.picture,
			nip05=excluded.nip05,
			updated_at=excluded.updated_at
		WHERE excluded.updated_at >= profiles.updated_at
	`, p.Pubkey, p.Name, p.DisplayName, p.Picture, p.NIP05, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert profil: %w", err)
	}
	return nil
}

// GetProfile lit le profil en cache d'une pubkey donnée.
func (s *Store) GetProfile(pubkey string) (Profile, error) {
	var p Profile
	row := s.db.QueryRow(`SELECT pubkey, name, display_name, picture, nip05, updated_at
		FROM profiles WHERE pubkey = ?`, pubkey)
	if err := row.Scan(&p.Pubkey, &p.Name, &p.DisplayName, &p.Picture, &p.NIP05, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNotFound
		}
		return p, fmt.Errorf("lecture profil: %w", err)
	}
	return p, nil
}
