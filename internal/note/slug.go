package note

import (
	"slices"
	"strings"
)

const maxSlugLen = 60

// translit maps common Latin letters with diacritics to ASCII.
// Other non-ASCII runes are treated as separators.
var translit = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'æ': "ae",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'œ': "oe",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ý': "y", 'ÿ': "y",
	'ß': "ss", 'š': "s", 'ž': "z", 'č': "c", 'ł': "l",
}

// Slugify turns a title into a filesystem- and URL-safe id. It never returns
// an empty string; titles without any usable characters become "note".
func Slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		var s string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			s = string(r)
		default:
			s = translit[r]
		}
		if s == "" {
			dash = b.Len() > 0
			continue
		}
		if dash {
			b.WriteByte('-')
			dash = false
		}
		b.WriteString(s)
	}
	slug := b.String()
	if len(slug) > maxSlugLen {
		slug = strings.TrimRight(slug[:maxSlugLen], "-")
	}
	if slug == "" {
		return "note"
	}
	return slug
}

// NormalizeTags trims, lowercases, drops empty, dedupes and sorts tags.
// The result is never nil.
func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			out = append(out, t)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// EditTags returns tags plus add minus remove, normalized. A tag in both add
// and remove is removed.
func EditTags(tags, add, remove []string) []string {
	rm := NormalizeTags(remove)
	return slices.DeleteFunc(NormalizeTags(append(slices.Clone(tags), add...)), func(t string) bool {
		_, found := slices.BinarySearch(rm, t)
		return found
	})
}
