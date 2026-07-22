package web

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"revolution/internal/store"
)

// rssFeed, rssChannel, rssItem, rssGUID modélisent juste assez de RSS 2.0
// pour un flux de lecture seule (titre, lien, date, résumé court) — pas
// besoin de plus pour "les derniers torrents publiés".
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Description string  `xml:"description"`
	Category    string  `xml:"category,omitempty"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// handleRSS sert le flux RSS des derniers torrents (/rss.xml). Désactivable
// via rss_enabled dans la config (renvoie alors 404), et protégé par une
// limite de requêtes par IP indépendante de celle du reste du site — un flux
// qui liste plusieurs dizaines d'entrées coûte plus cher qu'une page simple.
func (s *Server) handleRSS(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.RSSEnabled {
		http.NotFound(w, r)
		return
	}
	if !s.rssLim.allow(clientIP(r)) {
		http.Error(w, "trop de requêtes, réessayez plus tard", http.StatusTooManyRequests)
		return
	}

	maxItems := s.cfg.RSSMaxItems
	if maxItems <= 0 {
		maxItems = 50
	}

	results, err := s.store.Search("", maxItems, 0)
	if err != nil {
		log.Printf("web: rss: %v", err)
		http.Error(w, "erreur", http.StatusInternalServerError)
		return
	}

	base := requestBaseURL(r)
	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         s.cfg.SiteName + " — derniers torrents",
			Link:          base + "/",
			Description:   "Derniers torrents indexés par " + s.cfg.SiteName,
			Language:      "fr",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
		},
	}

	for _, t := range results {
		link := base + "/t/" + t.ID
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       t.Title,
			Link:        link,
			GUID:        rssGUID{IsPermaLink: "true", Value: link},
			PubDate:     time.Unix(t.PublishedAt, 0).Format(time.RFC1123Z),
			Description: rssItemSummary(t),
			Category:    t.Category,
		})
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		log.Printf("web: rss: encodage: %v", err)
	}
}

func rssItemSummary(t store.TorrentSummary) string {
	parts := []string{humanSize(t.SizeBytes)}
	if t.PublisherName != "" {
		parts = append(parts, "par "+t.PublisherName)
	}
	if t.HasStats {
		parts = append(parts, fmt.Sprintf("%d seeders / %d leechers", t.Seeders, t.Leechers))
	}
	return strings.Join(parts, " · ")
}

// requestBaseURL reconstruit l'URL de base à partir de la requête entrante
// (Host + schéma), pour que les liens du flux fonctionnent aussi bien en
// accès direct que derrière un reverse proxy.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
