package app

import (
	"context"
	"fmt"
	"path/filepath"

	"mangile-cli/internal/uploads"
)

type chapterRange struct {
	first  int
	last   int
	volume int
}

var batchMangaVolumes = map[string][]chapterRange{
	"20th Century Boys": {
		{33, 43, 4}, {44, 54, 5}, {55, 65, 6}, {66, 76, 7}, {77, 87, 8}, {88, 98, 9}, {99, 109, 10}, {110, 121, 11},
		{122, 133, 12}, {134, 145, 13}, {146, 157, 14}, {158, 170, 15}, {171, 181, 16}, {182, 192, 17}, {193, 203, 18}, {204, 214, 19},
		{215, 225, 20}, {226, 236, 21}, {237, 249, 22},
	},
	"Chainsaw Man": {
		{1, 7, 1}, {8, 16, 2}, {17, 25, 3}, {26, 34, 4}, {35, 43, 5}, {44, 52, 6}, {53, 61, 7}, {62, 70, 8}, {71, 79, 9}, {80, 88, 10},
		{89, 97, 11}, {98, 103, 12}, {104, 112, 13}, {113, 122, 14}, {123, 133, 15}, {134, 143, 16}, {144, 153, 17}, {154, 164, 18},
		{165, 175, 19}, {176, 186, 20}, {187, 198, 21}, {199, 210, 22}, {211, 222, 23}, {223, 229, 24},
	},
}

var batchMangaTitles = map[string]map[int]string{
	"20th Century Boys": {
		34: "Kaçış", 35: "Bağlantı", 36: "Aşk ve Barış", 37: "Gökkuşağı Çocuk", 38: "Işık", 39: "Patlama", 40: "Gizli Karargah", 41: "Robot Konferansı", 42: "Yeraltı Kralı", 43: "Karanlığın Ötesi",
	},
}

func (a *App) BatchMangaUpload(ctx context.Context, name string) error {
	if err := a.requireToken(); err != nil {
		return err
	}
	series, err := uploads.FindSeries(a.uploadsDir(), name)
	if err != nil {
		return err
	}
	if !series.IsManga() {
		return fmt.Errorf("seri manga değil: %s", series.Name())
	}
	seriesKey := filepath.Base(series.Dir)
	ranges, ok := batchMangaVolumes[seriesKey]
	if !ok {
		return fmt.Errorf("toplu cilt haritası yok: %s", series.Name())
	}
	sanityID, err := a.resolveSeriesID(ctx, &series, "manga")
	if err != nil {
		return err
	}
	chapters, err := a.scanMangaChapters(series.Dir)
	if err != nil {
		return err
	}
	existing, err := a.existingMangaNumbers(ctx, sanityID)
	if err != nil {
		return err
	}
	journal := uploads.NewJournal(sanityID, "manga")
	if !a.isDry() {
		if err := journal.Save(a.uploadsDir()); err != nil {
			return err
		}
	}
	created := 0
	for _, chapter := range skipEmptyChapters(chapters) {
		number := int(chapter.Number)
		if !chapter.NumberP || chapter.Number != float64(number) {
			return fmt.Errorf("geçersiz bölüm numarası: %s", chapter.Display)
		}
		if existing[number] {
			continue
		}
		if number == 146 {
			fmt.Printf("Atlanıyor: bölüm 146 (bozuk JPEG)\n")
			continue
		}
		volume, ok := batchMangaVolume(ranges, number)
		if !ok {
			return fmt.Errorf("cilt eşleşmesi yok: bölüm %d", number)
		}
		chapter.Volume = volume
		chapter.Title = batchMangaTitles[seriesKey][number]
		if a.isDry() {
			continue
		}
		if _, ok, err := a.uploadMangaChapter(ctx, journal, &series, sanityID, chapter); err != nil {
			_ = journal.Save(a.uploadsDir())
			return fmt.Errorf("bölüm %d: %w", number, err)
		} else if ok {
			created++
			if err := journal.Save(a.uploadsDir()); err != nil {
				return err
			}
		}
	}
	fmt.Printf("%s: %d yeni bölüm taslak olarak kaydedildi.\n", series.Name(), created)
	return nil
}

func (a *App) existingMangaNumbers(ctx context.Context, seriesID string) (map[int]bool, error) {
	var chapters []struct {
		Number float64 `json:"chapterNumber"`
	}
	if err := a.Client.Query(ctx, `*[_type == "mangaChapter" && manga._ref == $seriesID]{chapterNumber}`, map[string]any{"seriesID": seriesID}, &chapters); err != nil {
		return nil, err
	}
	existing := make(map[int]bool, len(chapters))
	for _, chapter := range chapters {
		if chapter.Number == float64(int(chapter.Number)) {
			existing[int(chapter.Number)] = true
		}
	}
	return existing, nil
}

func batchMangaVolume(ranges []chapterRange, number int) (int, bool) {
	for _, r := range ranges {
		if number >= r.first && number <= r.last {
			return r.volume, true
		}
	}
	return 0, false
}
