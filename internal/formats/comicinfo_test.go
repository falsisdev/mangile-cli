package formats

import "testing"

func TestParseComicInfo(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Title>Bölüm 45</Title>
  <Series>Chainsaw Man</Series>
  <Number>45</Number>
  <Volume>1</Volume>
  <PageCount>20</PageCount>
</ComicInfo>`
	data, ok := ParseComicInfo([]byte(xml))
	if !ok {
		t.Fatal("parsing error")
	}
	if data.Series != "Chainsaw Man" || data.Number != "45" || data.Volume != "1" || data.Title != "Bölüm 45" {
		t.Errorf("beklenmeyen sonuç: %+v", data)
	}
}

func TestParseComicInfoInvalid(t *testing.T) {
	if _, ok := ParseComicInfo([]byte("not xml")); ok {
		t.Fatal("geçersiz xml kabul edildi")
	}
}
