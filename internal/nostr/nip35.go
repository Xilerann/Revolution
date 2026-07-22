// Package nostrclient implémente la lecture NIP-35 (Torrents) sur Nostr :
// connexion aux relais, extraction des torrents (kind 2003) et des profils
// (kind 0) des comptes suivis, construction du lien magnet.
package nostrclient

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"

	"revolution/internal/store"
)

const (
	// KindTorrent est le kind NIP-35 pour un index de torrent.
	KindTorrent = 2003
	// KindTorrentComment est le kind NIP-35 pour un commentaire de torrent.
	KindTorrentComment = 2004
	// KindProfile est le kind NIP-01 standard pour les métadonnées de profil.
	KindProfile = 0
)

var infoHashRe = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// ParseTorrent convertit un événement Nostr kind 2003 (NIP-35) en store.Torrent.
//
// On refuse volontairement tout événement sans titre ou sans infohash BitTorrent v1
// valide (40 caractères hexadécimaux) : un enregistrement de catalogue sans ces deux
// champs ne peut pas produire de lien magnet exploitable, et accepter un infohash mal
// formé ouvrirait la porte à de l'injection dans le lien magnet généré côté web.
func ParseTorrent(evt *nostr.Event) (store.Torrent, error) {
	var t store.Torrent
	t.ID = evt.ID
	t.Pubkey = evt.PubKey
	t.Description = evt.Content
	t.PublishedAt = int64(evt.CreatedAt)

	if title := evt.Tags.Find("title"); title != nil {
		t.Title = strings.TrimSpace(title[1])
	}
	if t.Title == "" {
		return t, fmt.Errorf("événement %s: tag title manquant", evt.ID)
	}

	if x := evt.Tags.Find("x"); x != nil {
		t.InfoHash = strings.ToLower(strings.TrimSpace(x[1]))
	}
	if !infoHashRe.MatchString(t.InfoHash) {
		return t, fmt.Errorf("événement %s: infohash manquant ou invalide", evt.ID)
	}

	if tr := evt.Tags.Find("tracker"); tr != nil && isTrackerURL(tr[1]) {
		t.Tracker = strings.TrimSpace(tr[1])
	}

	for tag := range evt.Tags.FindAll("file") {
		f := store.TorrentFile{Path: tag[1]}
		if len(tag) >= 3 {
			if n, err := strconv.ParseInt(tag[2], 10, 64); err == nil && n >= 0 {
				f.SizeBytes = n
			}
		}
		t.Files = append(t.Files, f)
		t.SizeBytes += f.SizeBytes
	}

	var categories []string
	for tag := range evt.Tags.FindAll("t") {
		if v := strings.TrimSpace(tag[1]); v != "" {
			categories = append(categories, v)
		}
	}
	t.Category = strings.Join(categories, ", ")

	for tag := range evt.Tags.FindAll("i") {
		t.Refs = append(t.Refs, tag[1])
	}

	t.ImageURL = extractImage(evt.Tags)
	t.Magnet = BuildMagnet(t.InfoHash, t.Title, t.Tracker)

	return t, nil
}

// extractImage cherche une image d'illustration en best-effort : NIP-35 ne
// définit pas de tag image officiel, donc on accepte soit un tag `image`
// simple, soit un tag NIP-92 `imeta` contenant un champ `url` pointant vers
// une image, sans jamais faire échouer le parsing NIP-35 si absent.
func extractImage(tags nostr.Tags) string {
	if img := tags.Find("image"); img != nil && isHTTPURL(img[1]) {
		return img[1]
	}
	for tag := range tags.FindAll("imeta") {
		var candidate string
		for _, field := range tag[1:] {
			parts := strings.SplitN(field, " ", 2)
			if len(parts) == 2 && parts[0] == "url" && isHTTPURL(parts[1]) {
				candidate = parts[1]
				break
			}
		}
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// BuildMagnet reconstruit un lien magnet standard à partir de champs validés
// localement (jamais une chaîne magnet fournie brute par l'événement Nostr).
func BuildMagnet(infoHash, title, tracker string) string {
	v := url.Values{}
	v.Set("dn", title)
	if tracker != "" {
		v.Set("tr", tracker)
	}
	// url.Values.Encode() encode les espaces en "+" (forme application/x-www-form-urlencoded).
	// On les remplace par %20, plus largement accepté par les clients torrent qui parsent
	// les liens magnet (certains ne traitent pas "+" comme un espace).
	encoded := strings.ReplaceAll(v.Encode(), "+", "%20")
	return "magnet:?xt=urn:btih:" + strings.ToUpper(infoHash) + "&" + encoded
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isTrackerURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	switch u.Scheme {
	case "http", "https", "udp", "wss", "ws":
		return true
	default:
		return false
	}
}

// ProfileMeta est le contenu JSON (NIP-01) d'un événement kind 0.
type ProfileMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Picture     string `json:"picture"`
	NIP05       string `json:"nip05"`
}
