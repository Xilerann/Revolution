# Installation et démarrage

Ce guide part de zéro : une machine Linux vide, jusqu'à un site accessible.

## Prérequis

- Un serveur Linux (Ubuntu/Debian recommandé pour ce guide).
- 2 vCPU et 4 Go de RAM suffisent largement.
- Un accès shell (SSH) avec les droits pour installer un paquet système (Go).
- Aucune base de données à installer : Revolution utilise SQLite, intégré au binaire.

## Étape 1 : récupérer le code

```bash
git clone <url-du-dépôt> revolution
cd revolution
```

Remplacez `<url-du-dépôt>` par l'URL de ce dépôt Gitea.

## Étape 2 : lancer le script d'installation

```bash
./scripts/install.sh
```

Ce script fait trois choses, dans l'ordre :

1. Il installe Go via `apt` s'il n'est pas déjà présent.
2. Il crée le fichier `config.yaml` à partir de `config.example.yaml`, s'il n'existe pas encore.
3. Il compile le binaire une première fois.

Rien d'autre n'est installé. Pas de base de données externe, pas de serveur web
séparé, pas de gestionnaire de paquets Node ou Python.

## Étape 3 : ajouter au moins un relai Nostr

Revolution ne fonctionne pas sans relai : c'est par là que passent les
torrents publiés par les comptes que vous suivez.

```bash
./revolution relay add wss://relay.damus.io
./revolution relay add wss://nos.lol
```

Deux relais suffisent pour commencer. Voir
[relais-nostr.md](relais-nostr.md) pour choisir les vôtres.

## Étape 4 : suivre au moins un compte

```bash
./revolution follow add --alias monalias npub1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Remplacez le `npub1...` par la clé publique Nostr d'un compte qui publie des
torrents (voir [publier-nip35.md](publier-nip35.md) si c'est vous qui allez
publier). L'option `--alias` doit être placée avant le npub, pas après.

## Étape 5 : démarrer

```bash
./revolution start
```

Cette commande compile le binaire si le code source a changé depuis la
dernière fois, puis démarre le serveur en arrière-plan. Le site est alors
accessible à l'adresse indiquée dans le terminal (par défaut
`http://127.0.0.1:8420`).

Pour vérifier que ça tourne :

```bash
curl -s http://127.0.0.1:8420/
```

Pour arrêter proprement :

```bash
./revolution stop
```

## Rendre le site accessible depuis l'extérieur

Par défaut, Revolution écoute sur `127.0.0.1` : seule la machine elle-même
peut s'y connecter. Deux façons de le rendre accessible depuis internet :

1. **Recommandé, un reverse proxy** (nginx, Caddy...) devant Revolution, qui
   gère le nom de domaine et le certificat TLS. Revolution ne fait ni l'un ni
   l'autre lui-même.
2. **Sans proxy** : changez `listen_addr` dans `config.yaml` en
   `"0.0.0.0:8420"`, puis relancez (`./revolution stop && ./revolution start`).
   Le site sera alors accessible directement sur le port choisi, sans nom de
   domaine ni TLS.

## Démarrage automatique au redémarrage du serveur (systemd)

Pour que Revolution redémarre tout seul après un reboot du serveur, créez un
service systemd :

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

Remplacez `/chemin/vers/revolution` par le chemin réel où vous avez cloné le
dépôt. Puis :

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now revolution
```

Si vous utilisez systemd, n'utilisez plus `./revolution start`/`stop`
directement : passez par `systemctl start|stop|restart revolution`.

## Vérifier que tout va bien

```bash
./revolution follow list      # comptes suivis
./revolution relay list       # relais configurés
./revolution sync             # force une resynchronisation complète
```

Si `follow list` ou `relay list` affichent une liste vide, c'est normal juste
après l'installation : ajoutez-en avec les commandes de l'étape 3 et 4.
