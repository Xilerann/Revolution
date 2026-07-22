package cli

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"revolution/internal/store"
)

func cmdPurge(args []string) error {
	fs := flag.NewFlagSet("purge", flag.ContinueOnError)
	user := fs.String("user", "", "npub, hex ou alias du compte dont purger tous les torrents")
	yes := fs.Bool("yes", false, "ne pas demander confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *user == "" {
		return fmt.Errorf("usage: revolution purge --user <npub|hex|alias> [--yes]")
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

	count, err := st.CountByPubkey(f.Pubkey)
	if err != nil {
		return err
	}
	if count == 0 {
		fmt.Printf("aucun torrent catalogué pour %s\n", displayFollowed(f))
		return nil
	}

	if !*yes {
		fmt.Printf("%d torrent(s) de %s vont être supprimés du catalogue. Confirmer ? [y/N] ", count, displayFollowed(f))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(line)) != "y" {
			fmt.Println("annulé")
			return nil
		}
	}

	n, err := st.DeleteByPubkey(f.Pubkey)
	if err != nil {
		return err
	}
	fmt.Printf("%d torrent(s) supprimé(s) pour %s\n", n, displayFollowed(f))
	return nil
}
