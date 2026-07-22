package cli

import (
	"flag"
	"fmt"

	"revolution/internal/store"
)

func cmdTorrent(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: revolution torrent rm|edit ...")
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
	case "rm":
		if len(args) != 2 {
			return fmt.Errorf("usage: revolution torrent rm <id>")
		}
		if err := st.DeleteTorrent(args[1]); err != nil {
			return err
		}
		fmt.Println("torrent supprimé")
		return nil

	case "edit":
		if len(args) < 2 {
			return fmt.Errorf("usage: revolution torrent edit <id> --title T --category C")
		}
		id := args[1]

		fs := flag.NewFlagSet("torrent edit", flag.ContinueOnError)
		title := fs.String("title", "", "nouveau titre")
		category := fs.String("category", "", "nouvelle catégorie (libre)")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}

		t, err := st.GetTorrent(id)
		if err != nil {
			return err
		}
		newTitle, newCategory := t.Title, t.Category
		if *title != "" {
			newTitle = *title
		}
		if *category != "" {
			newCategory = *category
		}
		if err := st.UpdateTorrentFields(id, newTitle, newCategory); err != nil {
			return err
		}
		fmt.Println("torrent mis à jour")
		return nil

	default:
		return fmt.Errorf("sous-commande torrent inconnue : %s", args[0])
	}
}
