package cli

import (
	"flag"
	"fmt"

	nostrclient "revolution/internal/nostr"
	"revolution/internal/store"
)

func cmdFollow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: revolution follow add|rm|list ...")
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

	switch args[0] {
	case "add":
		fs := flag.NewFlagSet("follow add", flag.ContinueOnError)
		alias := fs.String("alias", "", "alias pratique pour désigner ce compte dans les autres commandes")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: revolution follow add <npub|hex> [--alias nom]")
		}
		hex, err := nostrclient.ResolvePubkey(fs.Arg(0))
		if err != nil {
			return err
		}
		if err := st.AddFollowed(hex, *alias); err != nil {
			return err
		}
		fmt.Printf("compte suivi : %s\n", hex)
		return nil

	case "rm":
		if len(args) != 2 {
			return fmt.Errorf("usage: revolution follow rm <npub|hex|alias>")
		}
		f, err := resolveFollowed(st, args[1])
		if err != nil {
			return err
		}
		if err := st.RemoveFollowed(f.Pubkey); err != nil {
			return err
		}
		fmt.Printf("compte retiré du suivi : %s (ses torrents restent catalogués — voir `revolution purge`)\n", displayFollowed(f))
		return nil

	case "list":
		list, err := st.ListFollowed()
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Println("aucun compte suivi")
			return nil
		}
		for _, f := range list {
			status := "actif"
			if !f.Enabled {
				status = "inactif"
			}
			npub := nostrclient.EncodeNpub(f.Pubkey)
			alias := f.Alias
			if alias == "" {
				alias = "—"
			}
			fmt.Printf("%s  %-16s  %s\n", npub, alias, status)
		}
		return nil

	default:
		return fmt.Errorf("sous-commande follow inconnue : %s", args[0])
	}
}
