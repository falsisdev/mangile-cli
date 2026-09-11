package formats

import (
	"regexp"
	"strings"
)

var turkishMap = map[rune]string{
	'ç': "c", 'Ç': "c", 'ğ': "g", 'Ğ': "g", 'ı': "i", 'İ': "i",
	'ö': "o", 'Ö': "o", 'ş': "s", 'Ş': "s", 'ü': "u", 'Ü': "u",
}

var invalidRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if repl, ok := turkishMap[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteRune(r)
	}
	slug := invalidRe.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "seri"
	}
	return slug
}
