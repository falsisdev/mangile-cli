package formats

import (
	"encoding/xml"
	"strings"
)

type comicInfo struct {
	XMLName   xml.Name `xml:"ComicInfo"`
	Series    string   `xml:"Series"`
	Number    string   `xml:"Number"`
	Volume    string   `xml:"Volume"`
	Title     string   `xml:"Title"`
	Publisher string   `xml:"Publisher"`
	PageCount string   `xml:"PageCount"`
}

type ComicInfoData struct {
	Series    string
	Number    string
	Volume    string
	Title     string
	Publisher string
	PageCount string
}

func ParseComicInfo(content []byte) (ComicInfoData, bool) {
	var ci comicInfo
	if err := xml.Unmarshal(content, &ci); err != nil {
		return ComicInfoData{}, false
	}
	return ComicInfoData{
		Series:    strings.TrimSpace(ci.Series),
		Number:    strings.TrimSpace(ci.Number),
		Volume:    strings.TrimSpace(ci.Volume),
		Title:     strings.TrimSpace(ci.Title),
		Publisher: strings.TrimSpace(ci.Publisher),
		PageCount: strings.TrimSpace(ci.PageCount),
	}, true
}
