package app

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"mangile-cli/internal/constants"
	"mangile-cli/internal/tui"
	"mangile-cli/internal/uploads"
)

const webMaxUploadBytes = 64 << 20

func (a *App) WebServe(ctx context.Context) error {
	if a.isDry() {
		tui.PrintInfo("[dry-run] Yerel sunucu başlatılmayacak → http://localhost:%d (dizin: %s)", a.Cfg.WebPort, a.uploadsDir())
		return nil
	}
	mux := a.webMux()
	addr := "127.0.0.1:" + strconv.Itoa(a.Cfg.WebPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("yerel sunucu açılamadı (%s): %w", addr, err)
	}
	tui.PrintTitle("Mangile Web Yükleyici")
	tui.PrintInfo("Adres: http://localhost:%d", a.Cfg.WebPort)
	tui.PrintInfo("Dizin: %s", a.uploadsDir())
	tui.PrintDim("Durdurmak için Ctrl+C")
	srv := &http.Server{Handler: mux}
	stop, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	go func() {
		<-stop.Done()
		_ = srv.Shutdown(context.Background())
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	tui.PrintInfo("Sunucu durduruldu.")
	return nil
}

func (a *App) webMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.webIndex)
	mux.HandleFunc("/upload", a.webUpload)
	return mux
}

func (a *App) webSeries() []uploads.Series {
	series, err := uploads.DiscoverSeries(a.uploadsDir())
	if err != nil {
		return nil
	}
	sort.SliceStable(series, func(i, j int) bool { return series[i].Name() < series[j].Name() })
	return series
}

func (a *App) webIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	series := a.webSeries()
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"tr\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	b.WriteString("<title>Mangile Web Yükleyici</title>")
	b.WriteString("<style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}label{display:block;margin:1rem 0 .25rem}select,input,button{font-size:1rem;padding:.5rem}select,input{width:100%;box-sizing:border-box}#drop{border:2px dashed #7C3AED;border-radius:.75rem;padding:2rem;text-align:center;color:#555;margin-top:1rem}#drop.over{background:#f3efff}ul{padding-left:1.25rem}.ok{color:#047857}.err{color:#b91c1c}</style>")
	b.WriteString("</head><body><h1>Mangile Web Yükleyici</h1>")
	b.WriteString("<p>Sayfaları sürükleyip bırakın. Dosyalar <code>uploads/&lt;Seri&gt;/&lt;Bölüm&gt;/</code> altına yazılır, Sanity'ye dokunulmaz.</p>")
	if len(series) == 0 {
		b.WriteString("<p class=\"err\">Seri bulunamadı. Önce <code>mangile run → Yeni seri dizini oluştur</code> ile seri ekleyin.</p>")
	}
	b.WriteString("<form id=\"f\" method=\"post\" action=\"/upload\" enctype=\"multipart/form-data\">")
	b.WriteString("<label for=\"series\">Seri</label><select id=\"series\" name=\"series\" required>")
	for _, s := range series {
		b.WriteString("<option value=\"" + html.EscapeString(filepath.Base(s.Dir)) + "\">" + html.EscapeString(s.Name()) + "</option>")
	}
	b.WriteString("</select>")
	b.WriteString("<label for=\"chapter\">Bölüm klasörü (ör. Bölüm 12)</label>")
	b.WriteString("<input id=\"chapter\" name=\"chapter\" required placeholder=\"Bölüm 12\">")
	b.WriteString("<div id=\"drop\">Dosyaları buraya bırakın veya seçin<br><br><input id=\"files\" type=\"file\" name=\"files\" multiple accept=\".jpg,.jpeg,.png,.webp\"></div>")
	b.WriteString("<p><button type=\"submit\">Yükle</button></p></form>")
	b.WriteString("<ul id=\"list\"></ul>")
	b.WriteString("<script>var d=document.getElementById('drop'),f=document.getElementById('files'),l=document.getElementById('list');['dragover','dragenter'].forEach(function(e){d.addEventListener(e,function(ev){ev.preventDefault();d.classList.add('over')})});['dragleave','drop'].forEach(function(e){d.addEventListener(e,function(ev){ev.preventDefault();d.classList.remove('over')})});d.addEventListener('drop',function(ev){if(ev.dataTransfer&&ev.dataTransfer.files.length){f.files=ev.dataTransfer.files;show()}});f.addEventListener('change',show);function show(){l.innerHTML='';for(var i=0;i<f.files.length;i++){var li=document.createElement('li');li.textContent=f.files[i].name;l.appendChild(li)}}document.getElementById('f').addEventListener('submit',function(ev){if(!f.files.length){ev.preventDefault();alert('Önce dosya seçin.')}})</script>")
	b.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, b.String())
}

func (a *App) webUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Yalnızca POST desteklenir.", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, webMaxUploadBytes)
	if err := r.ParseMultipartForm(webMaxUploadBytes); err != nil {
		webResult(w, false, "Yükleme çok büyük (en fazla 64MB) veya bozuk form.", nil)
		return
	}
	seriesName := strings.TrimSpace(r.FormValue("series"))
	chapterName := strings.TrimSpace(r.FormValue("chapter"))
	target, err := a.webTargetDir(seriesName, chapterName)
	if err != nil {
		webResult(w, false, err.Error(), nil)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		webResult(w, false, "Dosya seçilmedi.", nil)
		return
	}
	var saved []string
	var skipped []string
	for _, fh := range files {
		name, ok := webSafeImageName(fh.Filename)
		if !ok {
			skipped = append(skipped, fh.Filename)
			continue
		}
		src, err := fh.Open()
		if err != nil {
			skipped = append(skipped, fh.Filename)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(src, webMaxUploadBytes))
		_ = src.Close()
		if err != nil || len(data) == 0 {
			skipped = append(skipped, fh.Filename)
			continue
		}
		dest, err := webUniquePath(target, name)
		if err != nil {
			skipped = append(skipped, fh.Filename)
			continue
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			skipped = append(skipped, fh.Filename)
			continue
		}
		saved = append(saved, filepath.Base(dest))
	}
	if len(saved) == 0 {
		webResult(w, false, "Hiçbir dosya kaydedilemedi. Yalnızca jpg, jpeg, png, webp kabul edilir.", skipped)
		return
	}
	msg := fmt.Sprintf("%d dosya kaydedildi: %s", len(saved), target)
	webResult(w, true, msg, skipped)
}

func (a *App) webTargetDir(seriesName, chapterName string) (string, error) {
	if seriesName == "" || chapterName == "" {
		return "", errors.New("Seri ve bölüm adı zorunlu.")
	}
	if strings.ContainsAny(seriesName, `/\`) || strings.Contains(seriesName, "..") {
		return "", errors.New("Geçersiz seri adı.")
	}
	if strings.ContainsAny(chapterName, `/\`) || strings.Contains(chapterName, "..") {
		return "", errors.New("Bölüm adı klasör adı olmalı, yol içeremez (ör. Bölüm 12).")
	}
	if len(chapterName) > 120 {
		return "", errors.New("Bölüm adı çok uzun.")
	}
	series, err := uploads.FindSeries(a.uploadsDir(), seriesName)
	if err != nil {
		return "", fmt.Errorf("seri bulunamadı: %s", seriesName)
	}
	target := filepath.Join(series.Dir, chapterName)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

func webSafeImageName(name string) (string, bool) {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || strings.HasPrefix(base, ".") {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(base))
	if !constants.IsImageExt(ext) {
		return "", false
	}
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_' || r == ' ':
			return r
		case r == 'ç' || r == 'ğ' || r == 'ı' || r == 'ö' || r == 'ş' || r == 'ü' || r == 'Ç' || r == 'Ğ' || r == 'İ' || r == 'Ö' || r == 'Ş' || r == 'Ü':
			return r
		default:
			return '_'
		}
	}, base)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "", false
	}
	return clean, true
}

func webUniquePath(dir, name string) (string, error) {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate, nil
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; i < 1000; i++ {
		candidate = filepath.Join(dir, stem+"-"+strconv.Itoa(i)+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", errors.New("benzersiz dosya adı üretilemedi")
}

func webResult(w http.ResponseWriter, ok bool, msg string, skipped []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"tr\"><head><meta charset=\"utf-8\"><title>Sonuç</title>")
	b.WriteString("<style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}.ok{color:#047857}.err{color:#b91c1c}</style>")
	b.WriteString("</head><body>")
	if ok {
		b.WriteString("<h1 class=\"ok\">Tamamlandı</h1>")
	} else {
		w.WriteHeader(http.StatusBadRequest)
		b.WriteString("<h1 class=\"err\">Kaydedilemedi</h1>")
	}
	b.WriteString("<p>" + html.EscapeString(msg) + "</p>")
	if len(skipped) > 0 {
		b.WriteString("<p>Atlanan dosyalar:</p><ul>")
		for _, s := range skipped {
			b.WriteString("<li>" + html.EscapeString(s) + "</li>")
		}
		b.WriteString("</ul>")
	}
	b.WriteString("<p><a href=\"/\">Geri dön</a></p></body></html>")
	_, _ = io.WriteString(w, b.String())
}
