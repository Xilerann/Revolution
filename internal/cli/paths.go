package cli

// Chemins par défaut, résolus depuis le répertoire courant : Revolution est
// pensé pour être lancé depuis son propre dossier d'installation (voir README),
// donc pas de découverte compliquée de répertoire "home" ou XDG.
const (
	DefaultConfigPath = "config.yaml"
	DefaultPidPath    = "revolution.pid"
	DefaultLogPath    = "revolution.log"
)
