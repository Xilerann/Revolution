// Package assets embarque les templates HTML et les fichiers statiques du
// site web public dans le binaire, pour garder un déploiement à fichier unique.
package assets

import "embed"

//go:embed templates/*.html static/*.css static/*.png
var FS embed.FS
