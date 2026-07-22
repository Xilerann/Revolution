# Revolution

Indexeur de torrents public et gratuit, basé sur [Nostr](https://nostr.com/)
([NIP-35, Torrents](https://github.com/nostr-protocol/nips/blob/master/35.md)).

Chaque instance de Revolution est un binaire unique (Go), auto-suffisant,
que vous déployez et modérez vous-même : vous choisissez vos relais Nostr et
les comptes que vous suivez, l'application récupère automatiquement les
torrents que ces comptes publient et les ajoute à votre catalogue. Le site
web public est en lecture seule (recherche, fiche torrent, lien magnet) ;
toute la modération se fait en ligne de commande sur votre serveur.

Pensé pour tourner confortablement sur un petit serveur (2 vCPU / 4 Go de
RAM) : binaire Go statique, base SQLite embarquée, aucune dépendance externe
au runtime (pas de Node/Python/base de données à installer séparément).

**⚠️ Lisez [DISCLAIMER.md](DISCLAIMER.md) avant de déployer une instance
publique** : chaque opérateur d'instance est seul responsable des relais et
comptes qu'il choisit de suivre, et donc du contenu que son instance
catalogue et affiche.

## Documentation

- [Installation et démarrage](docs/installation.md)
- [Relais Nostr et filtrage](docs/relais-nostr.md)
- [Où sont stockées les données](docs/stockage-donnees.md)
- [Publier un torrent pour que Revolution l'indexe](docs/publier-nip35.md) :
  format NIP-35, exemples, liste de trackers publics

## Installation

Prérequis : Linux, [Go](https://go.dev/dl/) 1.23+ (installé automatiquement
par le script ci-dessous si absent via `apt`).

```bash
git clone <url-de-ce-dépôt> revolution
cd revolution
./scripts/install.sh
```

Le script installe Go si besoin, crée `config.yaml` à partir de
`config.example.yaml`, et effectue la première compilation.

## Démarrage rapide

```bash
# 1. Ajouter au moins un relai Nostr (gratuit/public)
./revolution relay add wss://relay.damus.io
./revolution relay add wss://nos.lol

# 2. Suivre un compte Nostr qui publie des torrents (NIP-35, kind 2003)
#    (les options doivent précéder l'argument positionnel, limitation du
#    parseur d'arguments standard de Go)
./revolution follow add --alias monalias npub1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 3. Démarrer l'instance (compile si le code a changé, puis tourne en arrière-plan)
./revolution start

# 4. Ouvrir le site
curl -s http://127.0.0.1:8420/ | head
```

`./revolution` est le point d'entrée unique : il recompile automatiquement
le binaire si le code source a changé depuis la dernière fois, puis exécute
la commande demandée. Une fois compilé, relancer une commande est quasi
instantané (aucune recompilation si rien n'a changé).

Le serveur web écoute uniquement sur `127.0.0.1` par défaut (voir
`listen_addr` dans `config.yaml`). Revolution ne gère ni domaine ni TLS,
c'est à vous de mettre un reverse proxy (nginx, Caddy, Traefik...) devant si
vous voulez l'exposer publiquement.

### Déploiement avec systemd (recommandé en production)

```ini
# /etc/systemd/system/revolution.service
[Unit]
Description=Revolution
After=network.target

[Service]
Type=simple
WorkingDirectory=/chemin/vers/revolution
ExecStart=/chemin/vers/revolution/revolution start --foreground
Restart=on-failure
User=revolution

[Install]
WantedBy=multi-user.target
```

Avec systemd, utilisez `systemctl start|stop|restart revolution` plutôt que
`./revolution start|stop` (les deux approches sont indépendantes, ne mélangez
pas `revolution start` en arrière-plan avec un service systemd sur la même
instance).

## Commandes

### Cycle de vie

| Commande | Effet |
|---|---|
| `revolution start [--foreground]` | Compile si besoin, démarre le serveur (arrière-plan par défaut) |
| `revolution stop` | Arrête proprement le serveur en cours (SIGTERM, attend la fermeture) |
| `revolution maintenance on\|off` | Bascule le mode maintenance du site public (503) ; l'ingestion Nostr continue en tâche de fond |

### Synchronisation Nostr

| Commande | Effet |
|---|---|
| `revolution sync` | Resynchronise **tous** les comptes suivis sur tous les relais actifs |
| `revolution fetch --user <npub\|hex\|alias>` | Resynchronise **un seul** compte suivi |
| `revolution purge --user <npub\|hex\|alias> [--yes]` | Supprime du catalogue **tous** les torrents publiés par ce compte |
| `revolution backup <chemin.db>` | Copie cohérente et complète du catalogue |

Le serveur en tourne ne se contente pas de sonder périodiquement : il ouvre
une **souscription Nostr permanente** (`live_refresh_minutes`, 5 min par
défaut) sur les comptes suivis. Les relais poussent les nouveaux torrents
dès leur publication, en général en quelques secondes, sans qu'on ait besoin
de les solliciter par des requêtes répétées. `revolution sync`/`fetch` et la
resynchronisation périodique de secours (`reconcile_interval_minutes`, 30 min
par défaut) ne servent qu'à rattraper ce que le live aurait manqué (coupure
réseau, relai temporairement indisponible, ou premier suivi d'un compte :
faites un `fetch --user` juste après un `follow add` pour récupérer son
historique).

**Nostr ne garantit aucune rétention** : un relai peut oublier un événement à
tout moment. Le catalogue local est donc **additif par construction**, un
cycle de synchronisation n'efface jamais une entrée déjà connue, seules les
commandes explicites (`purge`, `torrent rm`) suppriment quelque chose. Pensez
à `revolution backup` pour garder une copie du catalogue en dehors de
l'instance.

### Suivi (relais & comptes)

```bash
# Note : les options (--alias, --yes, --title, --category...) doivent être
# passées avant l'argument positionnel (npub/id), limitation du parseur
# d'arguments standard de Go.
revolution follow add [--alias nom] <npub|hex>
revolution follow rm  <npub|hex|alias>
revolution follow list

revolution relay add <wss://...>
revolution relay rm  <wss://...>
revolution relay list
```

Retirer un compte du suivi (`follow rm`) ne supprime pas ses torrents déjà
catalogués, utilisez `revolution purge` pour ça.

L'identité de référence d'un compte suivi est toujours son **npub** (affiché
par `follow list`, utilisé dans les messages de `fetch`/`purge`). L'alias
n'est qu'une commodité de saisie locale, pas l'identité elle-même : deux
opérateurs peuvent donner des alias différents au même compte, seul le npub
est sans ambiguïté.

### Modération manuelle d'un torrent

```bash
revolution torrent rm <id>
revolution torrent edit <id> --title "Nouveau titre" --category "film, 4k"
```

## Licence

AGPL-3.0, voir [LICENSE](LICENSE) et [NOTICE](NOTICE). Voir aussi
[DISCLAIMER.md](DISCLAIMER.md) pour la répartition des responsabilités entre
l'auteur du logiciel et les opérateurs d'instance.
