# Où sont stockées les données

Tout vit à côté du binaire, dans le dossier où vous avez cloné le dépôt. Il
n'y a pas de dossier caché ailleurs sur le système, sauf mention contraire
ci-dessous.

| Fichier | Contenu |
|---|---|
| `config.yaml` | Configuration de l'instance (adresse d'écoute, intervalles, etc.). Créé à partir de `config.example.yaml` lors de l'installation. |
| `revolution.db` | Le catalogue complet : torrents, fichiers, profils Nostr mis en cache, comptes suivis, relais, statistiques de seeders/leechers. Fichier SQLite unique. |
| `revolution.db-wal`, `revolution.db-shm` | Fichiers annexes de SQLite (mode WAL), utilisés pendant que le serveur tourne. Ne pas les supprimer manuellement pendant que Revolution est démarré. |
| `revolution.log` | Journal du serveur quand il tourne en arrière-plan (`revolution start` sans `--foreground`). |
| `revolution.pid` | Numéro de processus du serveur en cours, utilisé par `revolution stop`. |
| `bin/revolution` | Le binaire compilé. Régénéré automatiquement par `./revolution` si le code source a changé. |

Ces fichiers (sauf le code source) ne sont volontairement pas suivis par git
(voir `.gitignore`) : chaque instance a son propre catalogue, sa propre
configuration.

## Contenu exact du catalogue (`revolution.db`)

- **torrents** : un torrent par événement Nostr reçu (titre, description,
  infohash, lien magnet reconstruit, tracker, catégorie, taille, date de
  publication).
- **torrent_files** : la liste des fichiers de chaque torrent, si l'événement
  Nostr les décrit.
- **torrent_stats** : dernier résultat connu de scrape (nombre de seeders et
  leechers), par torrent. Absent tant qu'aucun scrape n'a réussi pour ce
  torrent.
- **profiles** : cache des profils Nostr (kind 0 : nom, avatar...) des
  comptes suivis, pour ne pas les redemander à chaque affichage.
- **followed** : la liste des comptes suivis (voir
  [relais-nostr.md](relais-nostr.md)).
- **relays** : la liste des relais configurés.
- **settings** : réglages internes (par exemple l'état du mode maintenance).

## Le catalogue ne perd jamais rien tout seul

Nostr ne garantit aucune rétention : un relai peut oublier un événement à
tout moment. Revolution ne supprime donc **jamais** une entrée du catalogue
en synchronisant. Un cycle de synchronisation ne fait qu'ajouter ou mettre à
jour, jamais supprimer. Seules des commandes explicites suppriment quelque
chose :

- `revolution purge --user <npub>` : supprime tous les torrents d'un compte.
- `revolution torrent rm <id>` : supprime un torrent précis.

## Sauvegarder le catalogue

```bash
revolution backup /chemin/vers/copie.db
```

Cette commande produit une copie complète et cohérente du fichier
`revolution.db`, même pendant que le serveur tourne. C'est la seule vraie
copie de sûreté : contrairement aux relais Nostr, rien ne garantit qu'un
événement reste disponible indéfiniment ailleurs que dans votre catalogue
local.
