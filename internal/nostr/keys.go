package nostrclient

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip19"
)

var hexPubkeyRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ResolvePubkey accepte une clé publique Nostr en hex (64 car.) ou en npub
// (bech32) et renvoie toujours la forme hex minuscule utilisée en interne.
func ResolvePubkey(s string) (string, error) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	if hexPubkeyRe.MatchString(lower) {
		return lower, nil
	}

	if strings.HasPrefix(lower, "npub1") {
		prefix, value, err := nip19.Decode(s)
		if err != nil {
			return "", fmt.Errorf("npub invalide: %w", err)
		}
		if prefix != "npub" {
			return "", fmt.Errorf("attendu un npub, reçu prefix %q", prefix)
		}
		hex, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("décodage npub inattendu")
		}
		return hex, nil
	}

	return "", fmt.Errorf("clé publique invalide (attendu hex 64 caractères ou npub1...): %q", s)
}

// EncodeNpub encode une pubkey hex en npub (bech32) pour l'affichage. C'est la
// forme qui doit servir à identifier un compte publicateur/suivi — un alias
// local n'est qu'une commodité de saisie, jamais l'identité de référence.
// En cas de hex invalide (ne devrait pas arriver, la valeur vient de la base
// ou d'un événement dont la signature a été vérifiée), renvoie le hex tel quel.
func EncodeNpub(hexPubkey string) string {
	npub, err := nip19.EncodePublicKey(hexPubkey)
	if err != nil {
		return hexPubkey
	}
	return npub
}
