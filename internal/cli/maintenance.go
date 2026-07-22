package cli

import (
	"fmt"

	"revolution/internal/store"
)

func cmdMaintenance(args []string) error {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return fmt.Errorf("usage: revolution maintenance on|off")
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

	on := args[0] == "on"
	if err := st.SetMaintenance(on); err != nil {
		return err
	}

	if on {
		fmt.Println("mode maintenance activé — le site public répond désormais 503 (l'ingestion Nostr continue en arrière-plan si le serveur tourne)")
	} else {
		fmt.Println("mode maintenance désactivé")
	}
	return nil
}
