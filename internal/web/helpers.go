package web

import (
	"fmt"
	"time"
)

// humanSize formate un nombre d'octets en unités lisibles (Ko, Mo, Go, ...).
func humanSize(n int64) string {
	if n <= 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d o", n)
	}
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit {
		div *= unit
		exp++
	}
	units := []string{"Ko", "Mo", "Go", "To", "Po"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}

// fmtTime formate un timestamp unix en date lisible (heure locale du serveur).
func fmtTime(ts int64) string {
	if ts <= 0 {
		return "—"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04")
}

// fmtAge formate l'ancienneté d'un timestamp unix ("il y a 12 min").
func fmtAge(ts int64) string {
	if ts <= 0 {
		return "—"
	}
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "à l'instant"
	case d < time.Hour:
		return fmt.Sprintf("il y a %d min", int(d.Minutes()))
	default:
		return fmt.Sprintf("il y a %dh", int(d.Hours()))
	}
}
