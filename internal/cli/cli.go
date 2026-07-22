// Package cli implémente les sous-commandes du binaire revolution : gestion
// du cycle de vie du serveur (start/stop/maintenance) et modération du
// catalogue (fetch/sync/purge/follow/relay/torrent). Ce sont les seules
// portes d'entrée en écriture de l'application — il n'y a pas d'API web
// d'administration.
package cli

import (
	"fmt"
	"os"

	"revolution/internal/config"
)

func loadConfig() (config.Config, error) {
	return config.Load(DefaultConfigPath)
}

// Run exécute la sous-commande demandée et renvoie le code de sortie du processus.
func Run(args []string) int {
	if len(args) < 1 {
		printUsage()
		return 2
	}

	cmd, rest := args[0], args[1:]
	var err error

	switch cmd {
	case "start":
		err = cmdStart(rest)
	case "stop":
		err = cmdStop(rest)
	case "maintenance":
		err = cmdMaintenance(rest)
	case "sync":
		err = cmdSync(rest)
	case "fetch":
		err = cmdFetch(rest)
	case "purge":
		err = cmdPurge(rest)
	case "follow":
		err = cmdFollow(rest)
	case "relay":
		err = cmdRelay(rest)
	case "torrent":
		err = cmdTorrent(rest)
	case "backup":
		err = cmdBackup(rest)
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "commande inconnue : %s\n\n", cmd)
		printUsage()
		return 2
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		return 1
	}
	return 0
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Revolution — indexeur de torrents public sur Nostr (NIP-35)

Usage: revolution <commande> [options]

Cycle de vie du serveur :
  start [--foreground]                Compile si besoin puis démarre (arrière-plan par défaut)
  stop                                 Arrête proprement le serveur en cours
  maintenance on|off                   Bascule le mode maintenance du site public

Synchronisation Nostr :
  sync                                 Resynchronise tous les comptes suivis
  fetch --user <npub|hex|alias>        Resynchronise un seul compte suivi
  purge --user <npub|hex|alias> [--yes]  Supprime du catalogue tous les torrents de ce compte

Configuration du suivi :
  follow add <npub|hex> [--alias nom]  Suit un nouveau compte Nostr
  follow rm <npub|hex|alias>           Ne suit plus ce compte
  follow list                          Liste les comptes suivis

  relay add <wss://...>                Ajoute un relai
  relay rm <wss://...>                 Retire un relai
  relay list                           Liste les relais configurés

Modération manuelle :
  torrent rm <id>                              Supprime un torrent précis
  torrent edit <id> --title T --category C     Corrige titre/catégorie localement

Sauvegarde :
  backup <chemin.db>                   Copie cohérente du catalogue (Nostr ne garantit
                                        aucune rétention, le catalogue local fait foi)
`)
}
