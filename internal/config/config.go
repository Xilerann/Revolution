// Package config charge et sauvegarde la configuration de l'instance Revolution.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config est l'ensemble des réglages d'une instance Revolution.
type Config struct {
	// ListenAddr est l'adresse d'écoute du serveur web public (lecture seule).
	// Par défaut lié à 127.0.0.1 : c'est au modérateur de mettre un reverse proxy devant.
	ListenAddr string `yaml:"listen_addr"`

	// DBPath est le chemin du fichier SQLite du catalogue.
	DBPath string `yaml:"db_path"`

	// SiteName est affiché en en-tête du site web.
	SiteName string `yaml:"site_name"`

	// LiveRefreshMinutes est l'intervalle auquel la souscription Nostr en
	// direct (le mécanisme principal de mise à jour du catalogue) relit la
	// liste des comptes suivis / relais en base et se reconfigure si elle a
	// changé (follow/relay add|rm exécutés pendant que le serveur tourne).
	LiveRefreshMinutes int `yaml:"live_refresh_minutes"`

	// ReconcileIntervalMinutes est l'intervalle entre deux resynchronisations
	// complètes de tous les comptes suivis. Ce n'est qu'un filet de sécurité
	// pour rattraper ce que la souscription live aurait manqué (coupure de
	// connexion, relai temporairement indisponible...) — pas le mécanisme
	// principal, d'où un intervalle volontairement large plutôt qu'un
	// sondage fréquent qui solliciterait inutilement les relais.
	ReconcileIntervalMinutes int `yaml:"reconcile_interval_minutes"`

	// RequestTimeoutSeconds borne chaque requête faite à un relai Nostr.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds"`

	// ScrapeEnabled active l'affichage du nombre de seeders/leechers, obtenu
	// par scrape UDP (BEP 15) des trackers udp:// connus — aucune connexion
	// DHT ni peer wire protocol, coût réseau négligeable.
	ScrapeEnabled bool `yaml:"scrape_enabled"`

	// ScrapeIntervalMinutes est l'intervalle entre deux cycles de scrape.
	ScrapeIntervalMinutes int `yaml:"scrape_interval_minutes"`

	// ScrapeTimeoutSeconds borne chaque requête UDP de scrape.
	ScrapeTimeoutSeconds int `yaml:"scrape_timeout_seconds"`

	// RSSEnabled active le flux RSS des derniers torrents (/rss.xml) et son
	// bouton d'accès sur le site. Activé par défaut.
	RSSEnabled bool `yaml:"rss_enabled"`

	// RSSMaxItems borne le nombre de torrents inclus dans le flux.
	RSSMaxItems int `yaml:"rss_max_items"`

	// RSSRateLimitPerMinute borne, par IP, le nombre de requêtes acceptées sur
	// /rss.xml — protection minimale contre un abus de cet endpoint (réponse
	// plus coûteuse qu'une simple page) pour en faire un vecteur de déni de
	// service. Indépendant de la limite générale du site.
	RSSRateLimitPerMinute int `yaml:"rss_rate_limit_per_minute"`
}

// Default renvoie une configuration par défaut raisonnable pour un serveur
// contraint (2 vCPU / 4 Go RAM).
func Default() Config {
	return Config{
		ListenAddr:               "127.0.0.1:8420",
		DBPath:                   "revolution.db",
		SiteName:                 "Revolution",
		LiveRefreshMinutes:       5,
		ReconcileIntervalMinutes: 30,
		RequestTimeoutSeconds:    10,
		ScrapeEnabled:            true,
		ScrapeIntervalMinutes:    30,
		ScrapeTimeoutSeconds:     5,
		RSSEnabled:               true,
		RSSMaxItems:              50,
		RSSRateLimitPerMinute:    12,
	}
}

// Load lit un fichier YAML de configuration. Si le fichier n'existe pas,
// renvoie la configuration par défaut sans erreur (premier démarrage).
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("lecture config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}

// Save écrit la configuration au format YAML.
func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("sérialisation config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("écriture config %s: %w", path, err)
	}
	return nil
}
