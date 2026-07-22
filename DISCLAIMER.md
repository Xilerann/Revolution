# Disclaimer / Avertissement

## Français

Revolution est un logiciel d'indexation de métadonnées de torrents,
distribué librement sous licence AGPL-3.0 (voir `LICENSE` et `NOTICE`).

Chaque instance de Revolution est déployée, opérée et modérée de manière
totalement **indépendante** par la personne ou l'entité qui l'héberge
("l'opérateur d'instance"). En particulier :

- Le choix des **relais Nostr** interrogés est configuré librement par
  l'opérateur de chaque instance.
- Le choix des **comptes Nostr suivis**, dont les publications (kind 2003,
  NIP-35) sont automatiquement récupérées et ajoutées au catalogue local,
  relève **exclusivement** de l'opérateur de cette instance.
- Le contenu effectivement catalogué, affiché, et rendu accessible par
  une instance donnée est donc sous la **seule responsabilité** de son
  opérateur, pas de l'auteur du logiciel Revolution.

L'auteur du logiciel Revolution :

- ne contrôle, n'héberge, ne modère et n'a connaissance d'aucun contenu
  indexé par une instance tierce ;
- ne fournit aucune infrastructure (relais, hébergement, nom de domaine,
  fichiers de torrents ou de données) pour les instances tierces ;
- décline toute responsabilité quant à l'usage fait de ce logiciel, à la
  légalité du contenu indexé par une instance donnée, ou aux conséquences
  de son exploitation par un opérateur d'instance.

Revolution ne stocke, ne télécharge et ne diffuse lui-même aucun fichier :
il se limite à indexer des métadonnées (titre, description, liens magnet)
publiées par des tiers sur le protocole Nostr. Le protocole BitTorrent
(DHT/PEX) est mis en œuvre uniquement par le client torrent de l'utilisateur
final, jamais par le serveur.

## English

Revolution is torrent-metadata indexing software, freely distributed under
the AGPL-3.0 license (see `LICENSE` and `NOTICE`).

Each Revolution instance is deployed, operated, and moderated **fully
independently** by whoever hosts it (the "instance operator"). In particular:

- The choice of which **Nostr relays** are queried is configured freely by
  each instance's operator.
- The choice of which **Nostr accounts are followed**, whose posts (kind
  2003, NIP-35) are automatically fetched and added to the local catalog,
  is **entirely** the instance operator's decision.
- The content actually cataloged, displayed, and made accessible by a given
  instance is therefore the **sole responsibility** of its operator, not
  of the Revolution software's author.

The author of the Revolution software:

- does not control, host, moderate, or have knowledge of any content
  indexed by a third-party instance;
- provides no infrastructure (relays, hosting, domain names, torrent or
  data files) for third-party instances;
- disclaims any responsibility for how this software is used, for the
  legality of content indexed by any given instance, or for the
  consequences of its operation by an instance operator.

Revolution itself never stores, downloads, or distributes any file: it only
indexes metadata (title, description, magnet links) published by third
parties over the Nostr protocol. The BitTorrent protocol (DHT/PEX) is
handled solely by the end user's own torrent client, never by the server.
