package sanity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

func randKey() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func ParagraphBlock(text string) map[string]any {
	if text == "" {
		return nil
	}
	children := parseInline(text)
	return map[string]any{
		"_type":    "block",
		"_key":     randKey(),
		"style":    "normal",
		"children": children,
	}
}

func HeadingBlock(text string, level int) map[string]any {
	if text == "" {
		return nil
	}
	return map[string]any{
		"_type":    "block",
		"_key":     randKey(),
		"style":    fmt.Sprintf("h%d", levelClamp(level)),
		"children": parseInline(text),
	}
}

func BulletBlock(text string) map[string]any {
	if text == "" {
		return nil
	}
	return map[string]any{
		"_type":    "block",
		"_key":     randKey(),
		"style":    "bullet",
		"children": parseInline(text),
	}
}

func BlockquoteBlock(text string) map[string]any {
	if text == "" {
		return nil
	}
	return map[string]any{
		"_type":    "block",
		"_key":     randKey(),
		"style":    "blockquote",
		"children": parseInline(text),
	}
}

func ImageBlock(assetID string) map[string]any {
	return map[string]any{
		"_type": "image",
		"_key":  randKey(),
		"asset": map[string]any{
			"_type": "reference",
			"_ref":  assetID,
		},
	}
}

func levelClamp(l int) int {
	if l < 1 {
		return 1
	}
	if l > 4 {
		return 4
	}
	return l
}

var linkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)

func parseInline(text string) []map[string]any {
	items, _ := splitLinks(text)
	return items
}

func splitLinks(text string) ([]map[string]any, bool) {
	loc := linkRe.FindStringIndex(text)
	if loc == nil {
		return []map[string]any{{"_type": "span", "marks": []string{}, "text": text}}, false
	}
	before := text[:loc[0]]
	match := text[loc[0]:loc[1]]
	after := text[loc[1]:]

	mu := linkRe.FindStringSubmatch(match)
	var children []map[string]any
	if before != "" {
		children = append(children, map[string]any{"_type": "span", "marks": []string{}, "text": before})
	}
	if len(mu) == 3 {
		children = append(children, map[string]any{
			"_type": "span",
			"marks": []string{"link"},
			"text":  mu[1],
		})
		children = append(children, map[string]any{
			"_type": "link",
			"_key":  randKey(),
			"href":  mu[2],
		})
	}
	rest, _ := splitLinks(after)
	children = append(children, rest...)
	return children, true
}

func PortableTextFromText(content string) []map[string]any {
	var blocks []map[string]any
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var para strings.Builder
	flush := func() {
		if para.Len() > 0 {
			if b := ParagraphBlock(strings.TrimSpace(para.String())); b != nil {
				blocks = append(blocks, b)
			}
			para.Reset()
		}
	}
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flush()
		case strings.HasPrefix(trimmed, "####"):
			flush()
			if b := HeadingBlock(strings.TrimSpace(trimmed[4:]), 4); b != nil {
				blocks = append(blocks, b)
			}
		case strings.HasPrefix(trimmed, "###"):
			flush()
			if b := HeadingBlock(strings.TrimSpace(trimmed[3:]), 3); b != nil {
				blocks = append(blocks, b)
			}
		case strings.HasPrefix(trimmed, "##"):
			flush()
			if b := HeadingBlock(strings.TrimSpace(trimmed[2:]), 2); b != nil {
				blocks = append(blocks, b)
			}
		case strings.HasPrefix(trimmed, "#"):
			flush()
			if b := HeadingBlock(strings.TrimSpace(trimmed[1:]), 1); b != nil {
				blocks = append(blocks, b)
			}
		case strings.HasPrefix(trimmed, "> "):
			flush()
			if b := BlockquoteBlock(strings.TrimSpace(trimmed[2:])); b != nil {
				blocks = append(blocks, b)
			}
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			flush()
			if b := BulletBlock(strings.TrimSpace(trimmed[2:])); b != nil {
				blocks = append(blocks, b)
			}
		default:
			para.WriteString(line)
			para.WriteString("\n")
		}
	}
	flush()
	return blocks
}

func TextFromPortableText(blocks []map[string]any) string {
	var lines []string
	for _, b := range blocks {
		line, ok := blockToText(b)
		if !ok {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n\n")
}

func blockToText(b map[string]any) (string, bool) {
	if b["_type"] != "block" {
		return "", false
	}
	style, _ := b["style"].(string)
	text := spansToText(b)
	switch {
	case strings.HasPrefix(style, "h") && len(style) == 2 && style[1] >= '1' && style[1] <= '6':
		level := int(style[1] - '0')
		return strings.Repeat("#", level) + " " + text, true
	case style == "blockquote":
		return "> " + text, true
	case style == "bullet":
		return "- " + text, true
	default:
		return text, true
	}
}

func childMaps(b map[string]any, key string) []map[string]any {
	if raw, ok := b[key].([]any); ok {
		var out []map[string]any
		for _, c := range raw {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	if maps, ok := b[key].([]map[string]any); ok {
		return maps
	}
	return nil
}

func spansToText(b map[string]any) string {
	hrefs := map[string]string{}
	for _, d := range childMaps(b, "markDefs") {
		if d["_type"] != "link" {
			continue
		}
		key, _ := d["_key"].(string)
		href, _ := d["href"].(string)
		if key != "" && href != "" {
			hrefs[key] = href
		}
	}
	var sb strings.Builder
	children := childMaps(b, "children")
	var inlineHrefs []string
	for _, m := range children {
		if m["_type"] == "link" {
			if href, _ := m["href"].(string); href != "" {
				inlineHrefs = append(inlineHrefs, href)
			}
		}
	}
	for _, span := range children {
		if span["_type"] != "span" {
			continue
		}
		text, _ := span["text"].(string)
		for _, key := range spanMarks(span) {
			href := hrefs[key]
			if href == "" && (key == "link" && len(inlineHrefs) > 0) {
				href, inlineHrefs = inlineHrefs[0], inlineHrefs[1:]
			}
			if href != "" {
				text = "[" + text + "](" + href + ")"
				break
			}
		}
		sb.WriteString(text)
	}
	return sb.String()
}

func spanMarks(span map[string]any) []string {
	if marks, ok := span["marks"].([]any); ok {
		var out []string
		for _, m := range marks {
			if key, ok := m.(string); ok {
				out = append(out, key)
			}
		}
		return out
	}
	if marks, ok := span["marks"].([]string); ok {
		return marks
	}
	return nil
}
