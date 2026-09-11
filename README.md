# Mangile CLI

A single, purpose-built CLI for publishing and managing Mangile content in Sanity. It consolidates chapter upload, draft management, publishing, and rollback into one interactive workflow.

[![Release](https://img.shields.io/github/v/release/falsisdev/mangile-cli?sort=semver)](https://github.com/falsisdev/mangile-cli/releases)

## Installation

### Prebuilt binaries (recommended)

Download the archive for your platform from the [Releases](https://github.com/falsisdev/mangile-cli/releases) page. No toolchain or source checkout needed.

**macOS / Linux:**
```bash
tar xzf mangile_<version>_darwin_<arch>.tar.gz   # or mangile_<version>_linux_<arch>.tar.gz
sudo mv mangile /usr/local/bin/
```
**Windows:** Extract `mangile_<version>_windows_<arch>.zip` and add the folder to your `PATH`, or call it directly as `.\mangile.exe`.

On macOS, the first launch may show an "unidentified developer" prompt (binaries are unsigned). Right-click the file, select **Open**, then confirm once. Windows SmartScreen shows a similar "unknown publisher" warning — choose **More info → Run anyway**.

### From source

```bash
git clone https://github.com/falsisdev/mangile-cli.git
cd mangile-cli
go build -o mangile .
```

## Getting Started

On first run, the CLI starts a setup wizard that asks where your content should live (default: `Documents/mangile`) and creates the workspace there. The choice is saved to your user configuration, so the command works from any terminal. You can re-run `mangile init` at any time.

Export your Sanity API token, then open the menu:

```bash
export SANITY_TOKEN="your_sanity_token"   # Windows: setx SANITY_TOKEN "your_sanity_token"
mangile run
```

The content root is resolved in this order: `MANGILE_UPLOADS` env var → user configuration → `./uploads`.

## Usage

The CLI is command-driven. Run `mangile run` (or just `mangile`) to open the interactive menu, or invoke a specific subcommand directly:

```bash
mangile run            # interactive menu (default)
mangile init           # scaffold the uploads/ workspace
mangile publish        # publish every pending draft
mangile rollback       # list rollback journals
mangile --dry-run run  # preview the plan, write nothing
mangile help           # usage help
```

### Command reference

The interactive menu (`mangile run`) is currently localized in Turkish.

| Operation | How to reach it | What it does |
|---|---|---|
| Upload manga chapter | `run` → _Upload manga chapter_ | Orders pages, uploads them as image assets, writes a draft chapter |
| Upload light novel chapter | `run` → _Upload light novel chapter_ | Parses the text file, builds Portable Text content, writes a draft |
| Create series directory | `run` → _Create series directory_ | Scaffolds `uploads/<Series>/` with a `config.yaml` |
| Show series status | `run` → _Show series status_ | Reconciles local series against Sanity by `myAnimeListId` |
| Publish drafts | `publish` (or from the menu) | Publishes every `drafts.**` document in batches of 20 |
| Rollback | `rollback` (or from the menu) | Picks a journal and reverts its operations |

### Example session

```bash
export SANITY_TOKEN="your_sanity_token"
mangile init
mkdir -p "uploads/Mushoku Tensei/Chapter 1"
cp scans/*.png "uploads/Mushoku Tensei/Chapter 1/"
cat > "uploads/Mushoku Tensei/config.yaml" <<'EOF'
myAnimeListId: 12345
type: manga
title: "Mushoku Tensei"
uploadStatus: uploading
EOF
mangile run        # upload the chapter, review the draft
mangile publish    # make the reviewed draft live
mangile rollback   # undo the upload if something looks wrong
```

## Content Layout

Each series is a folder under `uploads/` with a `config.yaml` describing how it maps to Sanity:

```
uploads/
├── Mushoku Tensei/
│   ├── config.yaml
│   ├── Chapter 1/
│   │   ├── 001.png
│   │   ├── 002.png
│   │   └── ...
│   └── Chapter 2/
│       └── ...
```

### config.yaml

The series is resolved against Sanity through `myAnimeListId` — a single source of truth shared with the backend (`chapter.groq` matches sibling chapters by this field, so uniqueness is required).

```yaml
myAnimeListId: 12345
type: manga # manga or lightNovel
title: "Mushoku Tensei"
uploadStatus: uploading
```

All other fields (slug, format, tags, cover, banner, ...) are optional and can be populated later.

## Uploading Chapters

### Manga

Drop the page files into a chapter folder. Page filenames are irrelevant — `1.png`, `001.png`, `sayfa3`, and `p12` are all resolved correctly. Files without a detectable number are sorted naturally, flagged, and placed last. A single `cbz`, `zip`, or `tar.gz` archive is also accepted in place of a folder. `ComicInfo.xml` metadata (number/volume/title) is honored when present.

### Light Novel

Each chapter is a folder containing a `*.txt` or `*.md` file:

```
Chapter 12/
└── content.txt
```

The chapter number is derived from the folder name (e.g. "Chapter 12"). You can also provide auxiliary metadata files:

- `baslik.txt` — chapter title
- `data.txt` — volume/chapter info, e.g. "Volume 2 Chapter 12"
- `image_1.txt`, `image_2.txt`, ... — illustration URLs (preserved as link annotations)

## Upload Pipeline

1. Chapters are selected and pages are ordered with a preview confirmation
2. Pages are uploaded to Sanity as image assets
3. The chapter is written as a **draft** — it does not appear on the site yet
4. You are asked whether to publish immediately; otherwise publish later with `mangile publish` after review in Sanity Studio

Re-uploading a chapter overwrites the existing one via deterministic IDs (`mangaChapter-<MAL>-<volume>-<chapter>`), so duplicates never accumulate in `latestChapters`.

## Rollback

Every session writes a journal to `uploads/.state/journal-<ts>.json`, enabling clean reversal even after the CLI exits:

- Newly created chapter documents are deleted
- Patched documents (e.g. `scan.titles`) are reverted via Sanity `revert`
- Orphaned assets are pruned when no longer referenced

```bash
mangile rollback
```

## Environment Variables

| Variable          | Default                         | Description                                        |
| ----------------- | ------------------------------- | -------------------------------------------------- |
| `SANITY_TOKEN`    | —                               | Sanity API token (required for network operations) |
| `MANGILE_UPLOADS` | user config → `./uploads`       | Content root directory                             |

`MANGILE_PROJECT_ID`, `MANGILE_DATASET`, and `MANGILE_API_VERSION` are also honored and default to the Mangile production values (`1yge7tlr`, `production`, `v2024-01-01`).

### User configuration

First-run settings are stored at:

- **macOS:** `~/Library/Application Support/mangile/config.yaml`
- **Windows:** `%AppData%\mangile\config.yaml`
- **Linux:** `~/.config/mangile/config.yaml`

It currently holds the content directory (`uploadsPath`). `MANGILE_UPLOADS` always overrides it.

> **Windows note:** for Turkish output on the legacy `cmd.exe`, run `chcp 65001`. PowerShell and Windows Terminal handle it natively.

## Roadmap

- **Phase 1** ✓ — Skeleton, pagesorter, manga/novel upload, draft/publish, rollback
- **Phase 2** — Embedded web server for drag-and-drop uploads, chapter edit/delete, cbr/7z support
- **Phase 3** — `create/update --fetch` (Jikan metadata), `doctor`, `import` (migrations + CSV)
- **Phase 4** — Polish, end-to-end tests, removal of legacy tooling

## License

MIT
