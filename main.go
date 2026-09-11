package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"mangile-cli/internal/app"
	"mangile-cli/internal/cfg"
	"mangile-cli/internal/tui"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	dryRun := false
	fetch := false
	versionRequested := false
	var positional []string
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--fetch":
			fetch = true
		case "--version", "-v":
			versionRequested = true
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintln(os.Stderr, "Bilinmeyen bayrak:", a)
				return 1
			}
			positional = append(positional, a)
		}
	}

	if versionRequested {
		fmt.Printf("mangile %s\n", version)
		return 0
	}

	cmd := "run"
	if len(positional) > 0 {
		cmd = positional[0]
	}

	switch cmd {
	case "version":
		fmt.Printf("mangile %s\n", version)
		return 0
	case "help", "--help", "-h":
		printHelp()
		return 0
	}

	c := cfg.Load()
	c.DryRun = dryRun
	a := app.New(c)

	if !cfg.HasUserConfig() && cmd != "init" {
		if !dryRun {
			if err := a.RunSetupWizard(); err != nil {
				tui.PrintError("Kurulum sihirbazı başarısız: %v", err)
				return 1
			}
		}
	}

	ctx := context.Background()
	switch cmd {
	case "run":
		return errCode(a.Run(ctx))
	case "init":
		return errCode(a.Init())
	case "publish":
		return errCode(a.PublishAll(ctx))
	case "rollback":
		return errCode(a.Rollback(ctx))
	case "web":
		return errCode(a.WebServe(ctx))
	case "chapter":
		sub := ""
		if len(positional) > 1 {
			sub = positional[1]
		}
		return errCode(a.ChapterManage(ctx, sub))
	case "create":
		return errCode(a.CreateSeries(ctx, fetch))
	case "update":
		return errCode(a.UpdateSeries(ctx, fetch))
	case "doctor":
		return errCode(a.Doctor(ctx))
	case "import":
		sub := ""
		if len(positional) > 1 {
			sub = positional[1]
		}
		switch sub {
		case "csv":
			path, name := "", ""
			if len(positional) > 2 {
				path = positional[2]
			}
			if len(positional) > 3 {
				name = positional[3]
			}
			return errCode(a.ImportCSV(ctx, path, name))
		case "migrate":
			return errCode(a.ImportMigrate(ctx))
		default:
			fmt.Fprintln(os.Stderr, "Kullanım: mangile import [csv <dosya> [seri] | migrate]")
			return 1
		}
	default:
		fmt.Fprintln(os.Stderr, "Bilinmeyen komut:", cmd)
		printHelp()
		return 1
	}
}

func errCode(err error) int {
	if err != nil {
		tui.PrintError("%v", err)
		return 1
	}
	return 0
}

func printHelp() {
	fmt.Println("Mangile CLI — Mangile içerik yükleyici")
	fmt.Println()
	fmt.Printf("Sürüm: %s\n", version)
	fmt.Println()
	fmt.Println("Kullanım: mangile [global] <komut>")
	fmt.Println()
	fmt.Println("Komutlar:")
	fmt.Println("  run          İnteraktif ana menü (varsayılan)")
	fmt.Println("  init         uploads/ dizinini ve yapılandırmayı hazırlar")
	fmt.Println("  publish      Tüm taslakları (drafts.**) yayınlar")
	fmt.Println("  rollback     İşlem günlüklerinden geri alma")
	fmt.Println("  web          Yerel sürükle-bırak yükleme sunucusu (localhost:8787)")
	fmt.Println("  chapter      Bölüm listele / düzenle / sil (alt komut: list, edit, delete)")
	fmt.Println("  create       Seri oluştur [--fetch ile Jikan'dan bilgi çek]")
	fmt.Println("  update       Seri güncelle [--fetch ile eksikleri doldur]")
	fmt.Println("  doctor       Tutarlılık taraması")
	fmt.Println("  import       İçe aktar (alt komut: csv <dosya> [seri], migrate)")
	fmt.Println("  version      Sürüm bilgisini gösterir")
	fmt.Println("  help         Bu yardımı gösterir")
	fmt.Println()
	fmt.Println("Global bayraklar:")
	fmt.Println("  --dry-run    Hiçbir şey yazılmaz, yalnızca plan gösterilir")
	fmt.Println("  --fetch      create/update ile Jikan'dan bilgi çeker")
	fmt.Println()
	fmt.Println("Ortam değişkenleri:")
	fmt.Println("  SANITY_TOKEN                Zorunlu (yükleme/yayın işlemleri için)")
	fmt.Println("  MANGILE_PROJECT_ID          Varsayılan: 1yge7tlr")
	fmt.Println("  MANGILE_DATASET             Varsayılan: production")
	fmt.Println("  MANGILE_UPLOADS             Varsayılan: kullanıcı config'i > ./uploads")
	fmt.Println("  MANGILE_WEB_PORT            Varsayılan: 8787")
	fmt.Println()
	fmt.Println("Kullanıcı yapılandırması:")
	fmt.Println("  os.UserConfigDir()/mangile/config.yaml (macOS: ~/Library/Application Support/mangile/)")
	fmt.Println("  İlk çalıştırmada otomatik kurulum sihirbazı rehberlik eder.")
	fmt.Println()
	fmt.Println("Dizin yapısı: uploads/<Seri>/Bölüm NN/ ... (manga: görseller; novel: text.txt/baslik.txt/data.txt/image_N.txt veya tek .txt/.md)")
}
