// Package web sert le site public en lecture seule : recherche de torrents
// et fiche détaillée avec lien magnet. Aucune authentification, aucune action
// d'écriture n'est exposée ici — toute la modération passe par la CLI.
package web

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	nostrclient "revolution/internal/nostr"

	"revolution/internal/config"
	"revolution/internal/store"
	assets "revolution/web"
)

const pageSize = 30

// Server sert le site public.
type Server struct {
	cfg    config.Config
	store  *store.Store
	tmpl   *template.Template
	lim    *limiter
	rssLim *limiter
}

// New construit le serveur web à partir de la configuration et du catalogue.
func New(cfg config.Config, st *store.Store) *Server {
	funcs := template.FuncMap{
		"humanSize": humanSize,
		"fmtTime":   fmtTime,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(assets.FS, "templates/*.html"))

	rssRate := cfg.RSSRateLimitPerMinute
	if rssRate <= 0 {
		rssRate = 12
	}

	return &Server{
		cfg:    cfg,
		store:  st,
		tmpl:   tmpl,
		lim:    newLimiter(20, time.Second), // 20 req/s/IP, largement suffisant pour un usage humain
		rssLim: newLimiter(rssRate, time.Minute),
	}
}

// Handler construit le http.Handler complet (routes + middlewares).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleSearch)
	mux.HandleFunc("GET /t/{id}", s.handleTorrent)
	mux.HandleFunc("GET /rss.xml", s.handleRSS)
	mux.Handle("GET /static/", http.FileServerFS(assets.FS))
	return s.withMiddleware(mux)
}

// ListenAndServe démarre le serveur HTTP public (bloquant).
func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("web: écoute sur http://%s", s.cfg.ListenAddr)
	return srv.ListenAndServe()
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.store.IsMaintenance() && !strings.HasPrefix(r.URL.Path, "/static/") {
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := s.tmpl.ExecuteTemplate(w, "maintenance.html", map[string]string{"SiteName": s.cfg.SiteName}); err != nil {
				log.Printf("web: template maintenance: %v", err)
			}
			return
		}
		if !s.lim.allow(clientIP(r)) {
			http.Error(w, "trop de requêtes, réessayez plus tard", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// resultView ajoute au résumé stocké en base une étiquette de publicateur
// prête à afficher : le nom de profil Nostr si connu, sinon le npub (jamais
// le hex brut — voir nostrclient.EncodeNpub).
type resultView struct {
	store.TorrentSummary
	PublisherLabel string
}

type searchView struct {
	SiteName   string
	Query      string
	Results    []resultView
	Page       int
	PrevPage   int
	NextPage   int
	HasMore    bool
	RSSEnabled bool
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := 0
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	results, err := s.store.Search(q, pageSize, page*pageSize)
	if err != nil {
		log.Printf("web: recherche: %v", err)
		http.Error(w, "erreur de recherche", http.StatusInternalServerError)
		return
	}

	views := make([]resultView, 0, len(results))
	for _, res := range results {
		label := res.PublisherName
		if label == "" {
			label = shortenNpub(nostrclient.EncodeNpub(res.Pubkey))
		}
		views = append(views, resultView{TorrentSummary: res, PublisherLabel: label})
	}

	data := searchView{
		SiteName:   s.cfg.SiteName,
		Query:      q,
		Results:    views,
		Page:       page,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		HasMore:    len(results) == pageSize,
		RSSEnabled: s.cfg.RSSEnabled,
	}
	if err := s.tmpl.ExecuteTemplate(w, "search.html", data); err != nil {
		log.Printf("web: template search: %v", err)
	}
}

type torrentView struct {
	SiteName         string
	Torrent          store.Torrent
	DescriptionHTML  template.HTML
	MagnetURL        template.URL
	PublisherName    string
	PublisherPicture string
	PublisherNpub    string
	StatsAvailable   bool
	Seeders          int
	Leechers         int
	StatsAge         string
}

func (s *Server) handleTorrent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.store.GetTorrent(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("web: lecture torrent %s: %v", id, err)
		http.Error(w, "erreur", http.StatusInternalServerError)
		return
	}

	npub := nostrclient.EncodeNpub(t.Pubkey)
	publisherName := shortenNpub(npub)
	var picture string
	if p, err := s.store.GetProfile(t.Pubkey); err == nil {
		if p.DisplayName != "" {
			publisherName = p.DisplayName
		} else if p.Name != "" {
			publisherName = p.Name
		}
		picture = p.Picture
	}

	data := torrentView{
		SiteName: s.cfg.SiteName,
		Torrent:  t,
		// t.Magnet est reconstruit par nous (internal/nostr.BuildMagnet) à partir de
		// champs validés (infohash regex, tracker/dn encodés) : jamais de contenu Nostr
		// brut. C'est un des rares cas légitimes d'utiliser template.URL (contournement
		// volontaire du filtre de schéma de html/template, qui bloquerait "magnet:").
		MagnetURL:        template.URL(t.Magnet),
		DescriptionHTML:  renderDescription(t.Description),
		PublisherName:    publisherName,
		PublisherPicture: picture,
		PublisherNpub:    npub,
	}

	// Statistiques de tracker (BEP 15) : best-effort, obtenues en arrière-plan
	// par ingest.ScrapeOnce. Absentes pour les torrents trackerless ou pas
	// encore scrapés — on n'affiche alors rien plutôt qu'un zéro trompeur.
	if stats, err := s.store.GetTorrentStats(t.ID); err == nil {
		data.StatsAvailable = true
		data.Seeders = stats.Seeders
		data.Leechers = stats.Leechers
		data.StatsAge = fmtAge(stats.ScrapedAt)
	}
	if err := s.tmpl.ExecuteTemplate(w, "torrent.html", data); err != nil {
		log.Printf("web: template torrent: %v", err)
	}
}

func shortenNpub(npub string) string {
	if len(npub) <= 18 {
		return npub
	}
	return npub[:10] + "…" + npub[len(npub)-6:]
}
