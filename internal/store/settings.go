package store

import (
	"database/sql"
	"errors"
	"fmt"
)

const settingMaintenance = "maintenance"

// GetSetting lit une valeur libre de la table settings.
func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("lecture setting %s: %w", key, err)
	}
	return v, nil
}

// SetSetting écrit une valeur libre dans la table settings.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("écriture setting %s: %w", key, err)
	}
	return nil
}

// IsMaintenance indique si le site public doit actuellement servir la page de maintenance.
func (s *Store) IsMaintenance() bool {
	v, err := s.GetSetting(settingMaintenance)
	if err != nil {
		return false
	}
	return v == "1"
}

// SetMaintenance active/désactive le mode maintenance du site public.
// L'ingestion Nostr continue de fonctionner indépendamment de ce flag.
func (s *Store) SetMaintenance(on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	return s.SetSetting(settingMaintenance, v)
}
