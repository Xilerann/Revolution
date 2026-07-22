package web

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// markdownRenderer convertit le Markdown des descriptions de torrent (NIP-35 :
// "long-description pre-formatted") en HTML. goldmark ne rend jamais le HTML
// brut présent dans la source (désactivé par défaut, on ne l'active pas
// volontairement) : c'est la première ligne de défense contre l'injection de
// contenu Nostr non fiable.
var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM), // tables, barré, autolien, listes de tâches
)

// sanitizePolicy est une seconde ligne de défense en profondeur : même si
// goldmark ne devait produire que du HTML sûr, on filtre quand même le
// résultat (liens/images limités à http(s)/mailto, pas de scripts/styles/
// gestionnaires d'événements).
var sanitizePolicy = bluemonday.UGCPolicy()

// renderDescription rend la description Markdown d'un torrent en HTML sûr à
// injecter tel quel dans le template (template.HTML : le sanitizing a déjà
// eu lieu, il ne faut pas ré-échapper).
func renderDescription(md string) template.HTML {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(md), &buf); err != nil {
		// Rendu Markdown impossible : on retombe sur du texte échappé plutôt
		// que d'afficher une erreur ou du contenu non sanitisé.
		return template.HTML(template.HTMLEscapeString(md))
	}
	safe := sanitizePolicy.SanitizeBytes(buf.Bytes())
	return template.HTML(safe)
}
