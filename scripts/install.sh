#!/usr/bin/env bash
# Installation minimale de Revolution : vérifie/installe Go si nécessaire,
# prépare config.yaml, effectue la première compilation.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
	echo "Go n'est pas installé. Installation via apt (paquet golang-go)…"
	if command -v apt-get >/dev/null 2>&1; then
		sudo apt-get update -y
		sudo apt-get install -y golang-go
	else
		echo "Gestionnaire de paquets non reconnu. Installez Go manuellement : https://go.dev/dl/" >&2
		exit 1
	fi
fi

go_version="$(go version | awk '{print $3}' | sed 's/go//')"
echo "Go détecté : $go_version"

if [[ ! -f config.yaml ]]; then
	cp config.example.yaml config.yaml
	echo "config.yaml créé à partir de config.example.yaml — pensez à l'ajuster."
fi

chmod +x ./revolution

echo "Première compilation…"
./revolution help >/dev/null

cat <<'EOF'

Installation terminée.

Prochaines étapes :
  1. Ajouter au moins un relai Nostr gratuit :
       ./revolution relay add wss://relay.damus.io
  2. Suivre un ou plusieurs comptes qui publient des torrents (NIP-35) :
       ./revolution follow add <npub...> --alias monalias
  3. Démarrer l'instance (compile si besoin, tourne en arrière-plan) :
       ./revolution start
  4. Ouvrir http://127.0.0.1:8420 (ou l'adresse configurée dans config.yaml)

Voir README.md pour le détail de toutes les commandes.
EOF
