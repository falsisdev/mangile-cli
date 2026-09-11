package formats

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Entry struct {
	Name     string
	Path     string
	Index    int
	HasIndex bool
	Tier     int
}

type SortResult struct {
	Entries  []Entry
	Warnings []string
}

var digitRunRe = regexp.MustCompile(`(\d+)`)

func DetectIndex(name string) (index int, tier int, ok bool) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		return 0, 0, false
	}
	digits := digitRunRe.FindAllString(trimmed, -1)
	if len(digits) == 0 {
		return 0, 0, false
	}
	if digitRunRe.MatchString(trimmed) && strings.Trim(trimmed, "0123456789") == "" {
		v, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, 0, false
		}
		return v, 1, true
	}
	last := digits[len(digits)-1]
	lastVal, err := strconv.Atoi(last)
	if err != nil {
		return 0, 0, false
	}
	tier = 3
	tail := strings.TrimSuffix(trimmed, last)
	tail = strings.TrimRight(tail, " _.-_()[]{}#,#")
	if len(digits) == 1 && strings.HasSuffix(base, last) {
		tier = 2
	}
	tailNonNumeric := false
	for _, r := range tail {
		if r != '-' && r != '_' && r != ' ' && r != '.' && r != '(' && r != ')' {
			tailNonNumeric = true
			break
		}
	}
	if len(digits) == 1 && tailNonNumeric {
		tier = 2
	}
	return lastVal, tier, true
}

func SortPages(entries []Entry) SortResult {
	var warnings []string
	for i := range entries {
		base := strings.ToLower(entries[i].Name)
		idx, tier, ok := DetectIndex(base)
		entries[i].Index = idx
		entries[i].Tier = tier
		entries[i].HasIndex = ok
	}

	seen := map[int]int{}
	for _, e := range entries {
		if e.HasIndex {
			seen[e.Index]++
		}
	}

	for idx, count := range seen {
		if count > 1 {
			warnings = append(warnings, "duplike sayfa indeksi "+strconv.Itoa(idx))
		}
	}

	var unnumbered []Entry
	var numbered []Entry
	for _, e := range entries {
		if e.HasIndex {
			numbered = append(numbered, e)
		} else {
			unnumbered = append(unnumbered, e)
		}
	}

	numbered = sortEntriesByIndex(numbered)
	unnumbered = sortEntriesByName(unnumbered)
	for _, e := range unnumbered {
		warnings = append(warnings, "numara içermiyor, ada göre dizildi: "+e.Name)
	}

	result := append(numbered, unnumbered...)
	return SortResult{Entries: result, Warnings: warnings}
}

func sortEntriesByIndex(entries []Entry) []Entry {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Index != entries[j].Index {
			return entries[i].Index < entries[j].Index
		}
		return NaturalLess(entries[i].Name, entries[j].Name)
	})
	return entries
}

func sortEntriesByName(entries []Entry) []Entry {
	sort.SliceStable(entries, func(i, j int) bool {
		return NaturalLess(entries[i].Name, entries[j].Name)
	})
	return entries
}
