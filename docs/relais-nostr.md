# Relais Nostr et filtrage

## Qu'est-ce qu'un relai ?

Un relai Nostr est un petit serveur qui reçoit, stocke et redistribue des
messages ("événements"). Nostr n'a pas de serveur central : n'importe qui
peut faire tourner un relai, et la plupart sont publics et gratuits.

Revolution ne publie rien : il se contente d'écouter les relais que vous avez
configurés, pour y récupérer les torrents publiés par les comptes que vous
suivez.

## Comment Revolution filtre ce qu'il récupère

Revolution ne récupère jamais "tout Nostr". Deux listes contrôlent
exactement ce qui entre dans le catalogue :

1. **La liste des relais** (`revolution relay list`) : les serveurs
   interrogés.
2. **La liste des comptes suivis** (`revolution follow list`) : les seules
   clés publiques (npub) dont les torrents sont récupérés.

Un événement qui n'est ni sur un relai configuré, ni publié par un compte
suivi, n'apparaît jamais dans le catalogue. Le filtrage se fait donc
entièrement par ces deux listes, il n'y a pas de mot-clé ou de catégorie à
configurer en plus.

## Ajouter et retirer des relais

```bash
revolution relay add wss://relay.damus.io
revolution relay list
revolution relay rm wss://relay.damus.io
```

Un relai retiré n'est plus interrogé, mais les torrents déjà catalogués
restent en place (ils viennent des comptes suivis, pas du relai lui-même).

## Ajouter et retirer des comptes suivis

```bash
revolution follow add --alias monalias npub1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
revolution follow list
revolution follow rm npub1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

L'alias n'est qu'une commodité de saisie locale : l'identité de référence
d'un compte est toujours son npub (visible dans `follow list`).

Retirer un compte du suivi n'efface pas ses torrents déjà catalogués. Pour ça,
utilisez `revolution purge --user <npub|alias>`.

Après un `follow add`, pensez à lancer `revolution fetch --user <alias>` pour
récupérer l'historique de ce compte (le suivi ne récupère que les nouveaux
torrents publiés à partir de maintenant).

## Trouver des relais publics

- [nostr.watch](https://nostr.watch/) liste des relais publics et leur état
  (en ligne, latence...).

Choisissez plusieurs relais différents, pas un seul : un même compte peut
publier sur des relais différents, et un relai peut tomber en panne.

## Faire tourner son propre relai

Ce n'est pas quelque chose que Revolution fait à votre place. Un relai est
un logiciel séparé. Si vous voulez héberger le vôtre plutôt que de dépendre de
relais publics :

- [nodetec/relayrunner](https://github.com/nodetec/relayrunner) : guide pas à
  pas pour installer différentes implémentations de relais.
- [scsibug/nostr-rs-relay](https://github.com/scsibug/nostr-rs-relay) : relai
  simple en Rust, avec image Docker, base SQLite intégrée (c'est celui qu'on a
  utilisé pour tester Revolution pendant son développement).

Une fois votre relai en ligne, ajoutez son adresse avec `revolution relay add`
comme n'importe quel autre relai.
