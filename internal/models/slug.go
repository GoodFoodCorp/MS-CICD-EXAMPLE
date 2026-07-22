package models

import "strings"

// GenerateSlug crée un slug à partir d'un string.
// (Vivait dans tenant.go ; les restaurants sont désormais gérés par
// franchise-service, mais les rôles utilisent toujours ce helper.)
func GenerateSlug(s string) string {
	// Convertir en minuscules
	slug := strings.ToLower(s)
	// Remplacer les espaces par des tirets
	slug = strings.ReplaceAll(slug, " ", "-")
	// Supprimer les caractères spéciaux
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	// Supprimer les tirets multiples
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Supprimer les tirets au début et à la fin
	return strings.Trim(slug, "-")
}
