package cli

import (
	"fmt"
	"net/url"

	"revolution/internal/store"
)

func cmdRelay(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: revolution relay add|rm|list ...")
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
		if len(args) != 2 {
			return fmt.Errorf("usage: revolution relay add <wss://...>")
		}
		if err := validateRelayURL(args[1]); err != nil {
			return err
		}
		if err := st.AddRelay(args[1]); err != nil {
			return err
		}
		fmt.Printf("relai ajouté : %s\n", args[1])
		return nil

	case "rm":
		if len(args) != 2 {
			return fmt.Errorf("usage: revolution relay rm <wss://...>")
		}
		if err := st.RemoveRelay(args[1]); err != nil {
			return err
		}
		fmt.Printf("relai retiré : %s\n", args[1])
		return nil

	case "list":
		list, err := st.ListRelays()
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Println("aucun relai configuré")
			return nil
		}
		for _, r := range list {
			status := "actif"
			if !r.Enabled {
				status = "inactif"
			}
			fmt.Printf("%-50s %s\n", r.URL, status)
		}
		return nil

	default:
		return fmt.Errorf("sous-commande relay inconnue : %s", args[0])
	}
}

func validateRelayURL(s string) error {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return fmt.Errorf("URL de relai invalide : %q", s)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("un relai Nostr doit être en ws:// ou wss:// (reçu %q)", u.Scheme)
	}
	return nil
}
