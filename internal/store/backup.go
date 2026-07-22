package store

import "fmt"

// BackupTo écrit une copie complète et cohérente du catalogue vers path, sans
// interrompre le service (VACUUM INTO prend un instantané cohérent même en
// mode WAL, y compris si le serveur tourne en même temps).
//
// Nostr ne garantit aucune rétention : un relai peut oublier un événement à
// tout moment (limite de stockage, politique de rétention, panne...). Le
// catalogue local est la copie de référence — voir aussi la garantie
// « ingestion additive uniquement » documentée sur Store.UpsertTorrent et
// Store.DeleteByPubkey/DeleteTorrent (les seuls points d'entrée en
// suppression, tous deux déclenchés explicitement par le modérateur).
func (s *Store) BackupTo(path string) error {
	if path == "" {
		return fmt.Errorf("chemin de sauvegarde vide")
	}
	if _, err := s.db.Exec(`VACUUM INTO ?`, path); err != nil {
		return fmt.Errorf("sauvegarde vers %s: %w", path, err)
	}
	return nil
}
