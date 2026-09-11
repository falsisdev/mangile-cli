package formats

import "unicode"

func naturalChunks(s string) []string {
	var chunks []string
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		j := i
		isDigit := unicode.IsDigit(runes[i])
		for j < len(runes) && unicode.IsDigit(runes[j]) == isDigit {
			j++
		}
		chunks = append(chunks, string(runes[i:j]))
		i = j
	}
	return chunks
}

func NaturalLess(a, b string) bool {
	ca := naturalChunks(a)
	cb := naturalChunks(b)
	for i := 0; i < len(ca) && i < len(cb); i++ {
		da, ea := digitChunk(ca[i])
		db, eb := digitChunk(cb[i])
		if da && db {
			na := trimLeadingZeros(ca[i])
			nb := trimLeadingZeros(cb[i])
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			_ = ea
			_ = eb
			continue
		}
		if da != db {
			return da
		}
		if ca[i] != cb[i] {
			return ca[i] < cb[i]
		}
	}
	return len(ca) < len(cb)
}

func digitChunk(s string) (bool, string) {
	if s == "" {
		return false, ""
	}
	return unicode.IsDigit([]rune(s)[0]), s
}

func trimLeadingZeros(s string) string {
	i := 0
	for i < len(s)-1 && s[i] == '0' {
		i++
	}
	return s[i:]
}
