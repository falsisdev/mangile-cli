package sanity

import "testing"

func TestPortableTextFromText(t *testing.T) {
	text := `# Birinci Bölüm

Paragraf bir satır ve ardından diğeri.

## Alt Başlık

- Madde 1
- Madde 2

> Alıntı`

	blocks := PortableTextFromText(text)
	if len(blocks) != 6 {
		t.Fatalf("beklenen 6 blok, alındı %d", len(blocks))
	}
	if blocks[0]["style"] != "h1" {
		t.Errorf("ilk blok h1 olmalı: %v", blocks[0]["style"])
	}
	if blocks[1]["style"] != "normal" {
		t.Errorf("ikinci blok normal olmalı")
	}
	if blocks[2]["style"] != "h2" {
		t.Errorf("üçüncü blok h2 olmalı")
	}
	if blocks[4]["style"] != "bullet" {
		t.Errorf("dört blok bullet olmalı")
	}
	if blocks[5]["style"] != "blockquote" {
		t.Errorf("blokquote bekleniyor")
	}
}

func TestPortableTextLinkAnnotation(t *testing.T) {
	text := `Metin içinde [link](https://ornek.com) var.`
	blocks := PortableTextFromText(text)
	if len(blocks) == 0 {
		t.Fatal("blok bekleniyor")
	}
	children := blocks[0]["children"].([]map[string]any)
	var linked bool
	for _, c := range children {
		if c["_type"] == "link" {
			linked = true
			if c["href"] != "https://ornek.com" {
				t.Errorf("href yanlış: %v", c["href"])
			}
		}
	}
	if !linked {
		t.Error("link annotation bulunamadı")
	}
}

func TestParagraphBlockEmpty(t *testing.T) {
	if b := ParagraphBlock(""); b != nil {
		t.Fatal("boş paragraf nil olmalı")
	}
}

func TestPortableTextHeadingDepthOrder(t *testing.T) {
	text := `#### Dördüncü
### Üçüncü
## İkinci
# Birinci`
	blocks := PortableTextFromText(text)
	if len(blocks) != 4 {
		t.Fatalf("4 blok beklenir, %d alındı", len(blocks))
	}
	expected := []string{"h4", "h3", "h2", "h1"}
	for i, want := range expected {
		if blocks[i]["style"] != want {
			t.Errorf("blok %d: style=%v, istediğim %s", i, blocks[i]["style"], want)
		}
	}
}

func TestPortableTextInlineBoldItalic(t *testing.T) {
	text := `**kalın** ve _italik_`
	blocks := PortableTextFromText(text)
	if len(blocks) != 1 {
		t.Fatalf("1 blok beklenir, %d alındı", len(blocks))
	}
	children := blocks[0]["children"].([]map[string]any)
	full := ""
	for _, c := range children {
		if s, ok := c["text"].(string); ok {
			full += s
		}
	}
	if full != "**kalın** ve _italik_" {
		t.Errorf("ham metin korunmalı: %q", full)
	}
}
