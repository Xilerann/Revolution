package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"revolution/internal/ingest"
	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
)

func cmdFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	user := fs.String("user", "", "npub, hex ou alias du compte à resynchroniser")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *user == "" {
		return fmt.Errorf("usage: revolution fetch --user <npub|hex|alias>")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	f, err := resolveFollowed(st, *user)
	if err != nil {
		return err
	}

	relays, err := st.ListEnabledRelayURLs()
	if err != nil {
		return err
	}
	if len(relays) == 0 {
		return fmt.Errorf("aucun relai configuré (voir `revolution relay add`)")
	}

	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout*5)
	defer cancel()

	nc := nostrclient.New(ctx)
	defer nc.Close()

	n, err := ingest.SyncUser(ctx, st, nc, relays, f, timeout)
	if err != nil {
		return err
	}

	fmt.Printf("%d torrent(s) synchronisé(s) pour %s\n", n, displayFollowed(f))
	return nil
}

// resolveFollowed résout un alias, une pubkey hex, ou un npub vers l'entrée
// Followed correspondante en base.
func resolveFollowed(st *store.Store, aliasOrKey string) (store.Followed, error) {
	if f, err := st.FindFollowed(aliasOrKey); err == nil {
		return f, nil
	}
	hex, err := nostrclient.ResolvePubkey(aliasOrKey)
	if err != nil {
		return store.Followed{}, fmt.Errorf("compte %q introuvable (ni alias, ni pubkey suivie): %w", aliasOrKey, err)
	}
	return st.FindFollowed(hex)
}

// displayFollowed renvoie une étiquette d'affichage basée sur le npub —
// l'identité de référence d'un compte suivi est sa clé Nostr, pas son alias
// local (qui n'est qu'une commodité de saisie et peut être ambigu/réutilisé).
func displayFollowed(f store.Followed) string {
	npub := nostrclient.EncodeNpub(f.Pubkey)
	if f.Alias != "" {
		return fmt.Sprintf("%s (%s)", npub, f.Alias)
	}
	return npub
}
