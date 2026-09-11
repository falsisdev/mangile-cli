# Mangile CLI

Mangile içeriğini Sanity'ye yayınlamak ve yönetmek için geliştirilmiş tek amaçlı CLI. Bölüm yükleme, taslak yönetimi, yayınlama ve geri alma işlemlerini tek bir etkileşimli akışta birleştirir.

[![Release](https://img.shields.io/github/v/release/falsisdev/mangile-cli?sort=semver)](https://github.com/falsisdev/mangile-cli/releases)

## Kurulum

### Hazır derlemeler (önerilen)

İşletim sisteminize uygun arşivi [Releases](https://github.com/falsisdev/mangile-cli/releases) sayfasından indirin. Ne Go araç zinciri ne de kaynak kodu gerekir.

Hangi arşivi indirmelisiniz?

- **macOS Apple Silicon (M1/M2/M3/M4/M5):** `mangile_<sürüm>_darwin_arm64.tar.gz`
- **macOS Intel:** `mangile_<sürüm>_darwin_amd64.tar.gz`
- **Windows (64-bit):** `mangile_<sürüm>_windows_amd64.zip`
- **Linux (64-bit):** `mangile_<sürüm>_linux_amd64.tar.gz`

**macOS / Linux:**
```bash
tar xzf mangile_<sürüm>_darwin_arm64.tar.gz   # veya linux_amd64
sudo mv mangile /usr/local/bin/
```

> **Önemli:** Arşivi açtığınızda çıkan `mangile` dosyasını `/usr/local/bin` gibi PATH'teki bir klasöre **taşımalısınız**. Aksi hâlde terminale `mangile` yazdığınızda "komut bulunamadı" hatası alırsınız. Taşıma sonrası `mangile --version` ile doğrulayın.

**Windows:** `mangile_<sürüm>_windows_amd64.zip` arşivini açın ve içindeki klasörü `PATH`'e ekleyin, ya da doğrudan `.\mangile.exe` olarak çağırın.

**İmza uyarıları:** Derlemeler imzasızdır. macOS'ta ilk açılışta "tanınmayan geliştirici" uyarısı çıkabilir; dosyaya **sağ tıklayıp Aç** deyin ve bir kez onaylayın. Windows SmartScreen benzer bir "bilinmeyen yayımcı" uyarısı gösterir — **Diğer bilgiler → Yine de çalıştır** deyin.

### Kaynak koddan derleme

```bash
git clone https://github.com/falsisdev/mangile-cli.git
cd mangile-cli
go build -o mangile .
```

## İlk Çalıştırma

CLI ilk çalıştırıldığında bir **kurulum sihirbazı** başlatır. Sihirbaz, içeriklerinizin hangi dizinde tutulacağını sorar (varsayılan: `~/Documents/mangile`) ve çalışma alanını orada oluşturur. Seçiminiz kullanıcı yapılandırmanıza kaydedilir; böylece komut herhangi bir terminalden çalışır. İstediğiniz zaman `mangile init` komutunu yeniden çalıştırabilirsiniz.

İçerik kök dizini şu sırayla çözülür: `MANGILE_UPLOADS` ortam değişkeni → kullanıcı yapılandırması → `./uploads`.

## SANITY_TOKEN — bilinmesi gerekenler

CLI, Sanity API token'ını **yalnızca `SANITY_TOKEN` ortam değişkeninden** okur. Bu bilinçli bir güvenlik kararıdır:

- Token hiçbir dosyaya yazılmaz; kurulum sihirbazı da token sormaz.
- Yükleme/yayınlama gibi ağ işlemlerinde **"SANITY_TOKEN tanımlı değil"** uyarısı, token'ın ortam değişkeninde olmadığını gösterir. Bu bir hata değildir; token'ı şu şekilde tanımlmanız yeterlidir:

```bash
export SANITY_TOKEN="sanity_tokenunuz"
```

`export` yalnızca o terminal penceresi için geçerlidir. Her yeni terminalde otomatik yüklensin isterseniz shell yapılandırmanıza ekleyin:

```bash
echo 'export SANITY_TOKEN="sanity_tokenunuz"' >> ~/.zshrc   # macOS / Linux (bash: ~/.bashrc)
setx SANITY_TOKEN "sanity_tokenunuz"                        # Windows (kalıcı)
```

> Token ile tekrar başlayın: `mangile --version` veya `mangile` komutu. Token'ı yalnızca kendi cihazınızda tutun; asla repoya, config dosyalarına veya paylaşılan yerlere yazmayın.

## Kullanım

CLI komut odaklıdır. `mangile run` (ya da doğrudan `mangile`) ile etkileşimli menüyü açın veya alt komutu doğrudan çağırın:

```bash
mangile run            # etkileşimli ana menü (varsayılan)
mangile init           # uploads/ çalışma alanını hazırlar
mangile publish        # bekleyen tüm taslakları yayınlar
mangile rollback       # işlem günlüklerini listeler / geri alır
mangile web            # yerel sürükle-bırak yükleme sunucusu (localhost:8787)
mangile chapter        # bölüm listele / düzenle / sil (alt komut: list, edit, delete)
mangile create         # seri oluştur [--fetch ile Jikan'dan bilgi çek]
mangile update         # seri güncelle [--fetch ile eksikleri doldur]
mangile doctor         # tutarlılık taraması
mangile import         # içe aktar (alt komut: csv <dosya> [seri], migrate)
mangile --dry-run run  # planı gösterir, hiçbir şey yazmaz
mangile version        # sürüm bilgisi
mangile help           # kullanım yardımı
```

### Komut referansı

Etkileşimli menü (`mangile run`) Türkçedir.

| İşlem | Nasıl ulaşılır | Ne yapar |
|---|---|---|
| Manga bölümü yükle | `run` → _Manga bölümü yükle_ | Sayfaları sıralar, görsel asset'lerini yükler, taslak bölüm yazar |
| Light novel bölümü yükle | `run` → _Light novel bölümü yükle_ | Metin dosyasını ayrıştırır, Portable Text içeriği üretir, taslak yazar |
| Seri dizini oluştur | `run` → _Seri dizini oluştur_ | `uploads/<Seri>/` dizinini bir `config.yaml` ile oluşturur |
| Seri durumunu göster | `run` → _Seri durumunu göster_ | Yerel serileri Sanity ile `myAnimeListId` üzerinden karşılaştırır |
| Taslakları yayınla | `publish` (veya menüden) | `drafts.**` cinsindeki tüm dokümanları 20'lik gruplar hâlinde yayınlar |
| Geri alma | `rollback` (veya menüden) | Bir günlüğü seçip işlemlerini geri alır |
| Bölüm düzenle / sil | `chapter` (veya menüden) | Sanity'deki bölümleri listeler; başlık ve cilt düzenler, bölüm siler |
| Seri oluştur | `create [--fetch]` (veya menüden) | Sanity'de seri taslağı açar, yerel dizin + config yazar; `--fetch` Jikan'dan başlık/özet/kapak çeker |
| Seri güncelle | `update [--fetch]` (veya menüden) | Yayın durumu ve etiketleri günceller; `--fetch` eksik başlık/özet/kapağı doldurur |
| Tutarlılık tara | `doctor` (veya menüden) | Yerel dizin + Sanity tutarlılık sorunlarını raporlar |
| Web yükleyici | `web` (veya menüden) | Tarayıcıdan sürükle-bırak ile sayfa dosyası yükler |

## Web Yükleyici

`mangile web`, yalnızca yerel dosyaya yazan bir yükleme sunucusu başlatır (varsayılan `http://localhost:8787`):

```bash
mangile web
```

Tarayıcıda seri seçip bölüm klasörü adı yazarak sayfaları sürükleyip bırakın. Dosyalar `uploads/<Seri>/<Bölüm>/` altına yazılır; **Sanity'ye dokunulmaz**, token gerekmez. Yalnızca `jpg`, `jpeg`, `png`, `webp` kabul edilir; aynı isimli dosya varsa `-2`, `-3` ekiyle benzersizleşir. Portu değiştirmek için `MANGILE_WEB_PORT` ortam değişkenini kullanın. Yüklenen sayfalar daha sonra normal akışla (`mangile run` → bölüm yükle) Sanity'ye gönderilir.

## Bölüm Düzenle / Sil

```bash
mangile chapter        # ne yapılacağını sorar
mangile chapter list   # bölümleri listeler
mangile chapter edit   # başlık ve cilt düzenler
mangile chapter delete # bölümü siler
```

Düzenleme, bölümün başlık ve cilt alanını yamalar; işlem günlüğe yazılır ve `mangile rollback` ile geri alınabilir. Silme, bölümün taslak ve yayınlanmış kopyasını birlikte kaldırır, başka hiçbir yerde kullanılmayan görselleri temizler ve silinen kaydın özetini günlüğe işler. Novel metin içeriğinin düzenlenmesi bu sürümde yoktur.

## Seri Oluştur / Güncelle

```bash
mangile create --fetch   # MAL ID sorar, Jikan'dan başlık/özet/kapak çeker
mangile update --fetch   # eksik alanları doldurur, durum ve etiketleri günceller
```

`create`, aynı MAL ID'nin Sanity'de kayıtlı olup olmadığını denetler; kayıtlıysa `update` önerir. Seri belgesi taslak olarak açılır (`manga-<MAL>` / `lightNovel-<MAL>`), kapak görseli indirilip asset olarak yüklenir, yerel `uploads/<Seri>/config.yaml` yazılır. Yayınlamadan önce onay sorulur. Jikan tür bilgisi yalnızca öneri amaçlı gösterilir; seri türünü (manga/lightNovel) siz seçersiniz.

## Tutarlılık Taraması

```bash
mangile doctor
```

Yerel dizini ve (token varsa) Sanity'yi tarar: `config.yaml` eksikliği, MAL ID yokluğu, geçersiz tür/durum, kanonik dışı etiket, sayfasız bölüm klasörü, numarasız bölüm, MAL ID çakışması ve karşılığı bulunamayan seriler raporlanır. Salt okunurdur; sorun varsa çıkış kodu 1 döner.

## İçe Aktar

```bash
mangile import csv veri.csv "Seri Adı"   # Web Scraper CSV'sinden bölüm klasörleri üretir
mangile import migrate                   # eski gömülü chapters[] dizisini novelChapter taslaklarına taşır
```

CSV içe aktarma, `data` sütunundan cilt/bölüm numaralarını çıkarıp `Cilt 001 Bölüm 002 - Başlık` klasörleri ve her sütuna bir `.txt` yazar. Migrasyon, deterministik kimliklerle (`novelChapter-<MAL>-<cilt>-<bölüm>`) taslak üretir; eski `chapters[]` dizisine dokunmaz, Studio'dan doğrulayıp temizlersiniz.

### Örnek oturum

```bash
export SANITY_TOKEN="sanity_tokenunuz"
mangile                              # ilk çalıştırmada kurulum sihirbazı
mkdir -p "~/Documents/mangile/Mushoku Tensei/Bölüm 1"
cp taramalar/*.png "~/Documents/mangile/Mushoku Tensei/Bölüm 1/"
cat > "~/Documents/mangile/Mushoku Tensei/config.yaml" <<'EOF'
myAnimeListId: 12345
type: manga
title: "Mushoku Tensei"
uploadStatus: uploading
EOF
mangile run        # bölümü yükleyin, taslağı inceleyin
mangile publish    # incelenen taslağı yayına alın
mangile rollback   # bir sorun görürseniz yüklemeyi geri alın
```

## İçerik Düzeni

Her seri, `uploads/` altında bir klasördür ve Sanity'ye nasıl bağlanacağını anlatan bir `config.yaml` içerir:

```
uploads/
├── Mushoku Tensei/
│   ├── config.yaml
│   ├── Bölüm 1/
│   │   ├── 001.png
│   │   ├── 002.png
│   │   └── ...
│   └── Bölüm 2/
│       └── ...
```

### config.yaml

Seri, Sanity ile `myAnimeListId` üzerinden eşleştirilir — arka uçla paylaşılan tek doğruluk kaynağıdır (`chapter.groq`, kardeş bölümleri bu alan üzerinden eşleştirir; bu yüzden değerin benzersiz olması zorunludur).

```yaml
myAnimeListId: 12345
type: manga # manga veya lightNovel
title: "Mushoku Tensei"
uploadStatus: uploading
```

Diğer tüm alanlar (slug, format, tags, cover, banner, ...) isteğe bağlıdır ve daha sonra doldurulabilir.

## Bölüm Yükleme

### Manga

Sayfa dosyalarını bölüm klasörüne koyun. Dosya adları önemsizdir — `1.png`, `001.png`, `sayfa3`, `p12` hepsi doğru çözülür. Sayı tespit edilemeyen dosyalar doğal sırayla listelenir, işaretlenir ve sona yerleştirilir. Klasör yerine tek bir `cbz`, `zip` veya `tar.gz` arşivi de kabul edilir. `ComicInfo.xml` meta verisi (numara/cilt/başlık) varsa dikkate alınır.

### Light Novel

Her bölüm, bir `*.txt` veya `*.md` dosyası içeren bir klasördür:

```
Bölüm 12/
└── content.txt
```

Bölüm numarası klasör adından çıkarılır (örn. "Bölüm 12"). İsteğe bağlı ek meta dosyaları:

- `baslik.txt` — bölüm başlığı
- `data.txt` — cilt/bölüm bilgisi, örn. "Cilt 2 Bölüm 12"
- `image_1.txt`, `image_2.txt`, ... — illüstrasyon URL'leri (link ek açıklaması olarak korunur)

## Yükleme Akışı

1. Bölümler seçilir, sayfalar önizlemeli sıralamayla yüklendik önce gösterilir
2. Sayfalar Sanity'ye görsel asset'i olarak yüklenir
3. Bölüm **taslak** olarak yazılır — siteye henüz düşmez
4. Hemen yayınlamak isteyip istemediğiniz sorulur; istemezseniz Sanity Studio'da inceledikten sonra `mangile publish` ile yayınlarsınız

Bölümü yeniden yüklemek, belirleyici kimlikler aracılığıyla mevcut kaydı ezer (`mangaChapter-<MAL>-<cilt>-<bölüm>`); böylece `latestChapters`'ta hiçbir zaman dupelike birikmez.

## Geri Alma

Her oturum, CLI kapansa bile temiz geri alma sağlayan bir günlüğü `uploads/.state/journal-<ts>.json` dosyasına yazar:

- Yeni oluşturulan bölüm dokümanları silinir
- Yaması yapılan dokümanlar (örn. `scan.titles`) Sanity `revert` ile geri alınır
- Başka hiçbir yerde referansı olmayan asset'ler temizlenir

```bash
mangile rollback
```

## Ortam Değişkenleri

| Değişken            | Varsayılan                       | Açıklama                                       |
| ------------------- | -------------------------------- | ---------------------------------------------- |
| `SANITY_TOKEN`      | —                                | Sanity API token (ağ işlemleri için zorunlu)   |
| `MANGILE_UPLOADS`   | kullanıcı config'i → `./uploads` | İçerik kök dizini                              |
| `MANGILE_WEB_PORT`  | `8787`                           | Web yükleyici portu                            |

`MANGILE_PROJECT_ID`, `MANGILE_DATASET` ve `MANGILE_API_VERSION` de tanınır ve Mangile üretim değerlerine eşittir (`1yge7tlr`, `production`, `v2024-01-01`).

### Kullanıcı yapılandırması

İlk çalıştırmada alınan ayarlar şu konumda saklanır:

- **macOS:** `~/Library/Application Support/mangile/config.yaml`
- **Windows:** `%AppData%\mangile\config.yaml`
- **Linux:** `~/.config/mangile/config.yaml`

Şu anda yalnızca içerik dizinini (`uploadsPath`) tutar. `MANGILE_UPLOADS` her zaman önceliklidir.

> **Windows notu:** Eski `cmd.exe` üzerinde Türkçe çıktı almak için `chcp 65001` çalıştırın. PowerShell ve Windows Terminal bu durumu yerel olarak halleder.

## Yol Haritası

- **Faz 1** ✓ — İskelet, pagesorter, manga/novel yükleme, taslak/yayın, geri alma
- **Faz 2** ✓ — Yerleşik web sunucusu, bölüm düzenle/sil + asset temizliği (cbr/7z sonraya bırakıldı)
- **Faz 3** ✓ — `create/update --fetch` (Jikan), `doctor`, `import` (CSV + migrasyon)
- **Faz 4** ✓ — Cilalama, uçtan uca testler, eski araç klasörlerinin kaldırılması (içerik `Documents/mangile` altına taşındı)
- **Faz 4** — Cilalama, uçtan uca testler, eski araçların kaldırılması

## Lisans

MIT