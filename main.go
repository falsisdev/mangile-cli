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

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	dryRun := false
	var positional []string
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			fmt.Fprintln(os.Stderr, "Bilinmeyen bayrak:", a)
			return 1
		}
		positional = append(positional, a)
	}

	cmd := "run"
	if len(positional) > 0 {
		cmd = positional[0]
	}

	c := cfg.Load()
	c.DryRun = dryRun
	a := app.New(c)
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
	case "create", "update", "chapter", "web", "import", "doctor":
		tui.PrintWarn("'%s' komutu Faz 2/3 kapsamında eklenecek.", cmd)
		return 0
	case "help", "--help", "-h":
		printHelp()
		return 0
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
	fmt.Println("Mangile CLI — the_mangile içerik yükleyici")
	fmt.Println()
	fmt.Println("Kullanım: mangile [global] <komut>")
	fmt.Println()
	fmt.Println("Komutlar:")
	fmt.Println("  run          İnteraktif ana menü (varsayılan)")
	fmt.Println("  init         uploads/ dizinini ve yapılandırmayı hazırlar")
	fmt.Println("  publish      Tüm taslakları (drafts.**) yayınlar")
	fmt.Println("  rollback     İşlem günlüklerinden geri alma")
	fmt.Println("  help         Bu yardımı gösterir")
	fmt.Println()
	fmt.Println("Global bayraklar:")
	fmt.Println("  --dry-run    Hiçbir şey yazılmaz, yalnızca plan gösterilir")
	fmt.Println()
	fmt.Println("Ortam değişkenleri:")
	fmt.Println("  SANITY_TOKEN                Zorunlu (yükleme/yayın işlemleri için)")
	fmt.Println("  MANGILE_PROJECT_ID          Varsayılan: 1yge7tlr")
	fmt.Println("  MANGILE_DATASET             Varsayılan: production")
	fmt.Println("  MANGILE_UPLOADS             Varsayılan: ./uploads")
	fmt.Println()
	fmt.Println("Dizin yapısı: uploads/<Seri>/Bölüm NN/ ... (manga: görseller; novel: text.txt/baslik.txt/data.txt/image_N.txt veya tek .txt/.md)")
}
