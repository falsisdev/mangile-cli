package app

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mangile-cli/internal/cfg"
	"mangile-cli/internal/sanity"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

var (
	bolumReWide = regexp.MustCompile(`(?i)b[oö]l[uü]m\s*(\d+(?:[.,]\d+)?)`)
	bolumRe     = regexp.MustCompile(`(?i)chapter\s*(\d+(?:[.,]\d+)?)`)
	ciltRe      = regexp.MustCompile(`(?i)cilt\s*(\d+)`)
	firstNumRe  = regexp.MustCompile(`\d+(?:[.,]\d+)?`)
)

type App struct {
	Cfg     cfg.Config
	Client  *sanity.Client
	uploads string
}

func New(c cfg.Config) *App {
	return &App{
		Cfg:     c,
		Client:  sanity.New(c.Token, c.ProjectID, c.Dataset, c.APIVersion),
		uploads: c.UploadsPath(),
	}
}

func (a *App) requireToken() error {
	if a.Cfg.HasToken() {
		return nil
	}
	return errors.New("SANITY_TOKEN ortam değişkeni tanımlı değil")
}

func (a *App) isDry() bool {
	return a.Cfg.DryRun
}

func (a *App) uploadsDir() string {
	return a.uploads
}

type sanitySeries struct {
	ID           string   `json:"_id"`
	Rev          string   `json:"_rev"`
	Type         string   `json:"_type"`
	Title        string   `json:"title"`
	MalID        int      `json:"myAnimeListId"`
	UploadStatus string   `json:"uploadStatus"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
}

func (a *App) findSanitySeries(ctx context.Context, malID int, seriesType string) (*sanitySeries, error) {
	if malID <= 0 {
		return nil, nil
	}
	var res []sanitySeries
	groq := `*[_type == $type && myAnimeListId == $malId][0..0]{
      _id, _rev, _type, title, myAnimeListId, uploadStatus,
      "slug": slug.current, description, tags
    }`
	if err := a.Client.Query(ctx, groq, map[string]any{"type": seriesType, "malId": malID}, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	return &res[0], nil
}

func (a *App) fetchSeriesByID(ctx context.Context, id string) (*sanitySeries, error) {
	if id == "" {
		return nil, nil
	}
	var res []sanitySeries
	groq := `*[_id == $id][0..0]{
      _id, _rev, _type, title, myAnimeListId, uploadStatus,
      "slug": slug.current, description, tags
    }`
	if err := a.Client.Query(ctx, groq, map[string]any{"id": id}, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	return &res[0], nil
}

type scanGroup struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func (a *App) fetchScans(ctx context.Context) ([]scanGroup, error) {
	var scans []scanGroup
	groq := `*[_type == "scan"]{_id, name} | order(name asc)`
	if err := a.Client.Query(ctx, groq, nil, &scans); err != nil {
		return nil, err
	}
	return scans, nil
}

func (a *App) pickSeries(ctx context.Context) (*uploads.Series, error) {
	series, err := uploads.DiscoverSeries(a.uploadsDir())
	if err != nil {
		return nil, err
	}
	if len(series) == 0 {
		return nil, errors.New("uploads dizininde seri yok")
	}

	type option struct {
		series uploads.Series
		label  string
	}
	var opts []option
	for _, s := range series {
		kind := "manga"
		if s.IsNovel() {
			kind = "lightNovel"
		}
		tag := ""
		if s.IsConfigured() {
			ss, err := a.fetchSeriesByID(ctx, s.Config.SanityID)
			if err == nil && ss == nil && s.Config.MalID > 0 {
				ss, _ = a.findSanitySeries(ctx, s.Config.MalID, kind)
			}
			if ss != nil {
				tag = " ✓ " + ss.ID
			} else {
				tag = " ? Sanity'de bulunamadı"
			}
		} else {
			tag = " (ayarlanmamış)"
		}
		opts = append(opts, option{series: s, label: fmt.Sprintf("%s [%s]%s", s.Name(), kind, tag)})
	}

	var labels []string
	for _, o := range opts {
		labels = append(labels, o.label)
	}
	var selected string
	if err := tui.SelectOne("Seri seçin", labels, &selected); err != nil {
		return nil, err
	}
	for _, o := range opts {
		if o.label == selected {
			return &o.series, nil
		}
	}
	return nil, errors.New("geçersiz seçim")
}

func chapterID(prefix string, malID int, volume int, number string) string {
	return prefix + "-" + strconv.Itoa(malID) + "-" + strconv.Itoa(volume) + "-" + normalizeChapterNum(number)
}

func normalizeChapterNum(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	s = strings.TrimSuffix(s, ".0")
	if s == "" {
		s = "0"
	}
	return s
}

func parseChapterNumber(s string) (float64, bool) {
	m := bolumReWide.FindStringSubmatch(s)
	if m == nil {
		m = bolumRe.FindStringSubmatch(s)
	}
	if m == nil {
		m = firstNumRe.FindStringSubmatch(s)
	}
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseVolume(s string) (int, bool) {
	m := ciltRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.Atoi(m[1])
	return v, err == nil
}

func cleanChapterTitle(folder string) string {
	for _, re := range []*regexp.Regexp{bolumReWide, bolumRe} {
		s := re.Split(folder, 2)
		if len(s) == 2 {
			rest := strings.TrimSpace(s[1])
			rest = strings.TrimLeft(rest, " :-_")
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func ref(id string) map[string]any {
	return map[string]any{"_type": "reference", "_ref": id}
}
