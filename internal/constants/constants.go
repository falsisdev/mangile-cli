package constants

const (
	ProjectIDDefault  = "1yge7tlr"
	DatasetDefault    = "production"
	APIVersionDefault = "v2024-01-01"
	MutateBatchSize   = 20
	PublishBatchSize  = 20
	WebServerPort     = 8787
	DocumentSizeLimit = 480_000
	MaxPreviewPages   = 3
)

var MangaTags = []string{
	"Ödüllü", "Adaptasyon", "One-Shot", "Aksiyon", "Macera",
	"Cinsellik", "Vahşet", "Doğaüstü", "Gerilim", "Korku",
	"Bilim Kurgu", "Dram", "Psikolojik", "Felsefik", "Romantik",
	"Tarihi", "Komedi", "Yaşamdan Kesit", "Gizem", "Fantezi",
}

var MangaFormats = []string{"manga", "manhwa", "manhua", "webtoon"}

var UploadStatuses = []string{"uploading", "completed", "hiatus", "cancelled"}

var ImageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

const (
	StatusUploading = "uploading"
	StatusCompleted = "completed"
	StatusHiatus    = "hiatus"
	StatusCancelled = "cancelled"
)

const (
	NovelChapterPrologue = 0
	NovelChapterEpilogue = 999
	NovelChapterExtra    = 1000
)

func IsImageExt(ext string) bool {
	for _, e := range ImageExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

func IsValidTag(tag string) bool {
	for _, t := range MangaTags {
		if t == tag {
			return true
		}
	}
	return false
}

func IsValidFormat(f string) bool {
	for _, v := range MangaFormats {
		if v == f {
			return true
		}
	}
	return false
}

func IsValidStatus(s string) bool {
	for _, v := range UploadStatuses {
		if v == s {
			return true
		}
	}
	return false
}
