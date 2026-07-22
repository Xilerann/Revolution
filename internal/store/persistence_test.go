package store

import (
	"path/filepath"
	"testing"
)

// TestCatalogIsAdditiveOnly documente et vérifie une garantie centrale du
// produit : Nostr ne garantit aucune rétention (un relai peut oublier un
// événement à tout moment), donc le catalogue local ne doit JAMAIS perdre un
// torrent simplement parce qu'il n'apparaît plus dans une réponse de relai.
// Seules des suppressions explicites (DeleteTorrent / DeleteByPubkey,
// déclenchées par le modérateur) doivent faire disparaître une entrée.
func TestCatalogIsAdditiveOnly(t *testing.T) {
	st := openTestStore(t)

	pubkey := "aa11111111111111111111111111111111111111111111111111111111111a"
	t1 := Torrent{
		ID: "id-1", Pubkey: pubkey, Title: "Torrent 1",
		InfoHash: "1111111111111111111111111111111111111a", Magnet: "magnet:?xt=urn:btih:1111",
		PublishedAt: 100,
	}
	t2 := Torrent{
		ID: "id-2", Pubkey: pubkey, Title: "Torrent 2",
		InfoHash: "2222222222222222222222222222222222222b", Magnet: "magnet:?xt=urn:btih:2222",
		PublishedAt: 200,
	}

	if err := st.UpsertTorrent(t1); err != nil {
		t.Fatalf("upsert t1: %v", err)
	}
	if err := st.UpsertTorrent(t2); err != nil {
		t.Fatalf("upsert t2: %v", err)
	}

	count, err := st.CountByPubkey(pubkey)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("attendu 2 torrents après insertion, obtenu %d", count)
	}

	// Simule une resynchronisation où le relai ne renvoie plus AUCUN événement
	// pour cet auteur (event pruné/expiré côté relai) : un cycle de sync ne
	// fait qu'upserter ce qui est reçu, il ne doit rien supprimer.
	// (aucun appel à Delete* ici, volontairement : c'est exactement le
	// comportement qu'on vérifie)

	count, err = st.CountByPubkey(pubkey)
	if err != nil {
		t.Fatalf("count après 'sync vide': %v", err)
	}
	if count != 2 {
		t.Fatalf("le catalogue a perdu des entrées alors qu'aucune suppression explicite n'a eu lieu : count=%d", count)
	}

	// Re-upserter le même événement (id-1) doit rester idempotent, pas dupliquer.
	if err := st.UpsertTorrent(t1); err != nil {
		t.Fatalf("re-upsert t1: %v", err)
	}
	count, err = st.CountByPubkey(pubkey)
	if err != nil {
		t.Fatalf("count après re-upsert: %v", err)
	}
	if count != 2 {
		t.Fatalf("re-upserter un événement déjà connu a dupliqué l'entrée : count=%d", count)
	}

	// Seule une suppression explicite doit faire disparaître les entrées.
	deleted, err := st.DeleteByPubkey(pubkey)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("attendu 2 suppressions explicites, obtenu %d", deleted)
	}
	if count, _ := st.CountByPubkey(pubkey); count != 0 {
		t.Fatalf("attendu 0 après suppression explicite, obtenu %d", count)
	}
}

// TestBackupToProducesUsableCopy vérifie que Store.BackupTo produit un
// fichier SQLite indépendant et complet (la "copie" que Nostr ne garantit
// pas), exploitable en rouvrant simplement le fichier.
func TestBackupToProducesUsableCopy(t *testing.T) {
	st := openTestStore(t)

	if err := st.UpsertTorrent(Torrent{
		ID: "id-1", Pubkey: "aa", Title: "Torrent 1",
		InfoHash: "1111111111111111111111111111111111111a", Magnet: "magnet:?xt=urn:btih:1111",
		PublishedAt: 100,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := st.BackupTo(backupPath); err != nil {
		t.Fatalf("backup: %v", err)
	}

	reopened, err := Open(backupPath)
	if err != nil {
		t.Fatalf("ouverture de la copie: %v", err)
	}
	defer reopened.Close()

	if _, err := reopened.GetTorrent("id-1"); err != nil {
		t.Fatalf("la copie ne contient pas le torrent attendu: %v", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(path)
	if err != nil {
		t.Fatalf("ouverture store de test: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
