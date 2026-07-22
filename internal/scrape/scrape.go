// Package scrape implémente le protocole de "scrape" UDP des trackers
// BitTorrent (BEP 15) : une requête très légère (deux allers-retours UDP,
// quelques dizaines d'octets) qui renvoie le nombre de seeders/leechers pour
// une liste d'infohash, sans avoir à rejoindre le swarm ni le DHT. C'est
// délibérément le seul mécanisme utilisé pour ces statistiques : contacter le
// DHT ou faire du peer wire protocol coûterait bien plus cher en ressources
// et en complexité pour un serveur qui ne fait sinon aucun réseau BitTorrent.
//
// Seuls les trackers udp:// sont interrogés : c'est le seul protocole de
// scrape normalisé et universellement supporté par les trackers publics ; il
// n'existe pas de convention fiable de scrape pour les trackers http(s).
package scrape

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"time"
)

// Stats est le résultat de scrape pour un infohash donné.
type Stats struct {
	Seeders   int
	Completed int
	Leechers  int
}

const (
	protocolMagic = 0x41727101980
	actionConnect = 0
	actionScrape  = 2

	// Limite conventionnelle du nombre d'infohash par requête de scrape UDP.
	MaxInfoHashesPerRequest = 74
)

// Scrape interroge un tracker UDP pour les infohash donnés (hex, 40
// caractères chacun). Renvoie une entrée par infohash effectivement présente
// dans la réponse du tracker (un tracker peut ne pas connaître un infohash).
func Scrape(trackerURL string, infoHashesHex []string, timeout time.Duration) (map[string]Stats, error) {
	u, err := url.Parse(trackerURL)
	if err != nil || u.Scheme != "udp" || u.Host == "" {
		return nil, fmt.Errorf("scrape: tracker non-udp ignoré : %q", trackerURL)
	}
	if len(infoHashesHex) == 0 {
		return nil, nil
	}
	if len(infoHashesHex) > MaxInfoHashesPerRequest {
		infoHashesHex = infoHashesHex[:MaxInfoHashesPerRequest]
	}

	addr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("résolution %s: %w", u.Host, err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("connexion udp %s: %w", u.Host, err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("réglage délai: %w", err)
	}

	connID, err := connect(conn)
	if err != nil {
		return nil, fmt.Errorf("connect BEP15 %s: %w", u.Host, err)
	}

	return doScrape(conn, connID, infoHashesHex)
}

func connect(conn *net.UDPConn) (uint64, error) {
	txID := rand.Uint32()

	req := make([]byte, 16)
	binary.BigEndian.PutUint64(req[0:8], protocolMagic)
	binary.BigEndian.PutUint32(req[8:12], actionConnect)
	binary.BigEndian.PutUint32(req[12:16], txID)

	if _, err := conn.Write(req); err != nil {
		return 0, fmt.Errorf("envoi: %w", err)
	}

	resp := make([]byte, 16)
	n, err := conn.Read(resp)
	if err != nil {
		return 0, fmt.Errorf("réception: %w", err)
	}
	if n < 16 {
		return 0, fmt.Errorf("réponse trop courte (%d octets)", n)
	}
	if action := binary.BigEndian.Uint32(resp[0:4]); action != actionConnect {
		return 0, fmt.Errorf("action inattendue %d", action)
	}
	if gotTxID := binary.BigEndian.Uint32(resp[4:8]); gotTxID != txID {
		return 0, fmt.Errorf("transaction_id ne correspond pas")
	}

	return binary.BigEndian.Uint64(resp[8:16]), nil
}

func doScrape(conn *net.UDPConn, connID uint64, infoHashesHex []string) (map[string]Stats, error) {
	txID := rand.Uint32()

	req := make([]byte, 16+20*len(infoHashesHex))
	binary.BigEndian.PutUint64(req[0:8], connID)
	binary.BigEndian.PutUint32(req[8:12], actionScrape)
	binary.BigEndian.PutUint32(req[12:16], txID)

	for i, ihHex := range infoHashesHex {
		raw, err := hex.DecodeString(ihHex)
		if err != nil || len(raw) != 20 {
			return nil, fmt.Errorf("infohash invalide : %q", ihHex)
		}
		copy(req[16+20*i:16+20*(i+1)], raw)
	}

	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("envoi scrape: %w", err)
	}

	resp := make([]byte, 8+12*len(infoHashesHex))
	n, err := conn.Read(resp)
	if err != nil {
		return nil, fmt.Errorf("réception scrape: %w", err)
	}
	if n < 8 {
		return nil, fmt.Errorf("réponse scrape trop courte (%d octets)", n)
	}
	if action := binary.BigEndian.Uint32(resp[0:4]); action != actionScrape {
		return nil, fmt.Errorf("action inattendue %d", action)
	}
	if gotTxID := binary.BigEndian.Uint32(resp[4:8]); gotTxID != txID {
		return nil, fmt.Errorf("transaction_id ne correspond pas")
	}

	out := make(map[string]Stats, len(infoHashesHex))
	for i, ihHex := range infoHashesHex {
		offset := 8 + 12*i
		if offset+12 > n {
			break // le tracker peut renvoyer moins d'entrées que demandé
		}
		out[ihHex] = Stats{
			Seeders:   int(binary.BigEndian.Uint32(resp[offset : offset+4])),
			Completed: int(binary.BigEndian.Uint32(resp[offset+4 : offset+8])),
			Leechers:  int(binary.BigEndian.Uint32(resp[offset+8 : offset+12])),
		}
	}
	return out, nil
}
