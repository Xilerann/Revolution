# Publier un torrent pour que Revolution l'indexe

Ce guide s'adresse à vous si vous voulez publier un torrent sur Nostr, pour
qu'une instance Revolution (la vôtre ou celle de quelqu'un d'autre) le
récupère et l'affiche correctement.

L'idée est simple. Vous publiez un événement Nostr signé avec votre clé. Cet
événement contient le titre, le hash du torrent, éventuellement une image et
une description. Revolution lit ça et construit lui-même le lien magnet, la
fiche et le reste.

Rien de tout ça n'est spécifique à Revolution : c'est le format
[NIP-35](https://github.com/nostr-protocol/nips/blob/master/35.md), ouvert et
utilisable par n'importe quel outil compatible.

## Ce dont vous avez besoin

- Une clé Nostr (une paire de clés `nsec`/`npub`). Si vous avez déjà un
  compte Nostr, c'est la même clé que pour vos notes habituelles.
- Un relai à qui envoyer l'événement (voir
  [relais-nostr.md](relais-nostr.md) pour en choisir un ou deux).
- Un moyen d'envoyer un événement signé avec des tags personnalisés. La
  plupart des clients Nostr grand public ne savent poster que des notes
  simples, il vous faut donc un outil qui laisse construire l'événement à la
  main. Le plus simple : [nak](https://github.com/fiatjaf/nak), un petit
  programme en ligne de commande fait pour ça.

## Publier avec nak

Installez `nak` (voir son README pour l'installation selon votre système),
puis construisez votre événement en une seule commande. Chaque information
va dans un tag `--tag clef=valeur`, et le relai se met à la fin.

```bash
nak event \
  --sec nsec1votrecledepublicationprivee \
  --kind 2003 \
  --tag title="Torrent de démonstration Revolution" \
  --tag x=65b9af20ea1ba57a0400f2dd6511d093162c91c2 \
  --tag "file=track_1.bin;2228224" \
  --tag "file=README.md;11650" \
  --tag tracker=udp://tracker.opentrackr.org:1337/announce \
  --tag t=demo \
  --tag image=https://picsum.photos/id/1011/1200/500 \
  --content "Votre description en Markdown ici" \
  wss://relay.damus.io
```

Remplacez `--sec` par votre propre clé privée (`nsec...` ou en hexadécimal),
et le contenu des tags par les vôtres. Vous pouvez donner plusieurs relais à
la suite si vous voulez publier au même endroit partout.

## Ce qui compte dans l'événement

- **`title`** : le titre affiché. Sans lui, Revolution ignore l'événement.
- **`x`** : le hash BitTorrent, exactement 40 caractères hexadécimaux.
  Sans lui non plus, l'événement est ignoré. Ce hash doit être le vrai hash
  de votre fichier `.torrent` (calculé par `mktorrent` ou votre client
  torrent habituel), pas une valeur inventée : sinon personne ne pourra
  jamais télécharger quoi que ce soit avec le lien magnet généré.
- **`file`** : un fichier de l'archive, chemin puis taille en octets.
  Répétez le tag autant de fois que vous avez de fichiers.
- **`tracker`** : voir la section suivante, c'est important.
- **`t`** : une catégorie libre, un mot par tag. Répétez pour plusieurs.
- **`image`** : l'image de couverture, en `http://` ou `https://`.

Le lien magnet ne se met pas dans l'événement : chaque instance Revolution
le reconstruit elle-même à partir du hash, du titre et du tracker, pour ne
jamais afficher un lien fourni tel quel sans vérification.

## Ajoutez toujours un tracker public

Sans tracker, votre torrent ne peut compter que sur le DHT pour trouver des
pairs, ce qui marche moins bien et rend impossible l'affichage du nombre de
seeders/leechers sur Revolution. Ajoutez toujours au moins un tracker
`udp://` avec le tag `tracker`.

Quelques trackers publics qui fonctionnent bien, testés pendant le
développement de Revolution :

- `udp://tracker.opentrackr.org:1337/announce`
- `udp://open.stealth.si:80/announce`
- `udp://tracker.torrent.eu.org:451/announce`

Les trackers publics changent avec le temps (certains disparaissent,
d'autres apparaissent). Pour une liste à jour, régulièrement mise à jour :
[ngosang/trackerslist](https://github.com/ngosang/trackerslist).

Vous pouvez répéter le tag `tracker` plusieurs fois pour en indiquer
plusieurs d'un coup.

## Composer une belle description

Le champ `content` accepte du Markdown, et Revolution le transforme en page
propre (titres, gras, italique, listes, citations, tableaux, images). Une
description bien composée fait toute la différence entre une fiche torrent
qui donne envie et une simple ligne de texte brut.

Quelques principes simples :

- Commencez par une image si vous en avez une, ça donne tout de suite le ton.
- Utilisez un ou deux titres (`##`, `###`) pour séparer les sections plutôt
  qu'un mur de texte.
- Une courte citation (`>`) fonctionne bien pour une accroche ou une note.
- Une liste à puces se lit plus vite qu'une phrase qui énumère tout à la
  suite.

Pour insérer une image dans le texte, la syntaxe est la même que partout en
Markdown :

```markdown
![Texte alternatif](https://example.com/image.jpg)
```

## Exemple complet et bien structuré

Plutôt qu'un exemple inventé, voici l'événement exact utilisé pour publier
le torrent de démonstration de cette instance Revolution. Le hash est réel,
les fichiers existent vraiment, et ce torrent est réellement seedé : vous
pouvez le retrouver tel quel sur le site pour voir le résultat.

Le format JSON brut d'un événement Nostr oblige à mettre tout le `content`
sur une seule ligne (avec des `\n` pour les retours à la ligne), ce qui n'est
pas très lisible. Voici donc d'abord la description telle qu'elle a été
écrite, en Markdown normal :

```markdown
## Lorem Ipsum

Lorem ipsum dolor sit amet, **consectetur adipiscing elit**. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat.

> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
> dolore eu fugiat nulla pariatur.

### Sed ut perspiciatis unde omnis

- Nemo enim ipsam voluptatem quia voluptas sit aspernatur
- Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet
- Ut enim ad minima veniam, quis nostrum exercitationem ullam

Lorem ipsum dolor sit amet, consectetur adipiscing elit. *Sed do eiusmod
tempor incididunt* ut labore et dolore magna aliqua, ut enim ad minim veniam.

![Illustration](https://picsum.photos/id/1015/1200/500)

Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut
aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in
voluptate velit esse cillum dolore eu fugiat nulla pariatur.
```

Et voici les tags qui l'accompagnent (avec `nak`, c'est ce texte tel quel
qu'on donne à `--content`, il s'occupe lui-même de le mettre au bon format) :

```json
[
  ["title", "Torrent de démonstration Revolution"],
  ["x", "65b9af20ea1ba57a0400f2dd6511d093162c91c2"],
  ["file", "README.md", "11650"],
  ["file", "track_1.bin", "2228224"],
  ["file", "track_2.bin", "2359296"],
  ["file", "track_3.bin", "2490368"],
  ["file", "track_4.bin", "2621440"],
  ["file", "track_5.bin", "2752512"],
  ["tracker", "udp://tracker.opentrackr.org:1337/announce"],
  ["t", "demo"],
  ["t", "test"],
  ["image", "https://picsum.photos/id/1011/1200/500"]
]
```

Ensemble, ce texte et ces tags forment l'événement `kind: 2003` complet.
Remarquez la structure de la description une fois qu'on la relit avec les
principes du dessus : un titre pour ouvrir, un paragraphe d'intro en gras,
une citation, un sous-titre, une liste, un peu d'italique, une image au
milieu du texte, puis une conclusion. C'est exactement ce qui donne, une
fois affiché sur le site, une fiche torrent qui se lit bien plutôt qu'un
bloc de texte plat.

Et pour référence, voici à quoi ressemble le même événement une fois
assemblé en JSON brut, le format réellement envoyé au relai (le `content`
est obligatoirement sur une seule ligne, avec des `\n` à la place des
retours à la ligne) :

```json
{
  "kind": 2003,
  "content": "## Lorem Ipsum\n\nLorem ipsum dolor sit amet, **consectetur adipiscing elit**. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.\n\n> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.\n\n### Sed ut perspiciatis unde omnis\n\n- Nemo enim ipsam voluptatem quia voluptas sit aspernatur\n- Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet\n- Ut enim ad minima veniam, quis nostrum exercitationem ullam\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit. *Sed do eiusmod tempor incididunt* ut labore et dolore magna aliqua, ut enim ad minim veniam.\n\n![Illustration](https://picsum.photos/id/1015/1200/500)\n\nUt enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
  "tags": [
    ["title", "Torrent de démonstration Revolution"],
    ["x", "65b9af20ea1ba57a0400f2dd6511d093162c91c2"],
    ["file", "README.md", "11650"],
    ["file", "track_1.bin", "2228224"],
    ["file", "track_2.bin", "2359296"],
    ["file", "track_3.bin", "2490368"],
    ["file", "track_4.bin", "2621440"],
    ["file", "track_5.bin", "2752512"],
    ["tracker", "udp://tracker.opentrackr.org:1337/announce"],
    ["t", "demo"],
    ["t", "test"],
    ["image", "https://picsum.photos/id/1011/1200/500"]
  ]
}
```

C'est ce deuxième bloc qui part réellement sur le relai une fois signé (avec
en plus `id`, `pubkey`, `created_at` et `sig` ajoutés par votre outil de
publication). Le premier bloc, en Markdown lisible, sert juste à comprendre
comment il a été composé.

## Vérifier que ça marche

Une fois publié, deux cas de figure :

- Si l'instance Revolution vous suit déjà, elle le récupère toute seule en
  quelques secondes, grâce à la souscription en direct.
- Sinon, demandez à l'opérateur de vous suivre (`revolution follow add`),
  puis de lancer `revolution fetch --user vous` pour récupérer aussi ce que
  vous aviez déjà publié avant.
