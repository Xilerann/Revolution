// Command revolution est le binaire unique de l'instance : serveur web
// public + ingestion Nostr + commandes de modération (voir `revolution help`).
package main

import (
	"os"

	"revolution/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
