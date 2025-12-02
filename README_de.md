# vget

Vielseitiges Kommandozeilen-Tool zum Herunterladen von Audio, Video, Podcasts und mehr.

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Français](README_fr.md)

## Installation

### macOS

```bash
curl -fsSL https://github.com/guiyumin/vget/releases/latest/download/vget-darwin-arm64 -o vget
chmod +x vget
sudo mv vget /usr/local/bin/
```

### Linux / WSL

```bash
curl -fsSL https://github.com/guiyumin/vget/releases/latest/download/vget-linux-amd64 -o vget
chmod +x vget
sudo mv vget /usr/local/bin/
```

### Windows

Laden Sie `vget-windows-amd64.exe` von [Releases](https://github.com/guiyumin/vget/releases/latest) herunter und fügen Sie es zum PATH hinzu.

## Befehle

| Befehl                             | Beschreibung                          |
|------------------------------------|---------------------------------------|
| `vget [url]`                       | Medien herunterladen (`-o`, `-q`, `--info`) |
| `vget ls <remote>:<path>`          | Remote-Verzeichnis auflisten (`--json`) |
| `vget init`                        | Interaktiver Konfigurationsassistent  |
| `vget update`                      | Automatische Aktualisierung           |
| `vget search --podcast <query>`    | Podcasts suchen                       |
| `vget completion [shell]`          | Shell-Vervollständigung generieren    |
| `vget config show`                 | Konfiguration anzeigen                |
| `vget config path`                 | Konfigurationsdateipfad anzeigen      |
| `vget config webdav list`          | Konfigurierte WebDAV-Server auflisten |
| `vget config webdav add <name>`    | WebDAV-Server hinzufügen              |
| `vget config webdav show <name>`   | Serverdetails anzeigen                |
| `vget config webdav delete <name>` | Server löschen                        |

### Beispiele

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o mein_video.mp4
vget --info https://example.com/video
vget search --podcast "tech news"
vget pikpak:/path/to/file.mp4              # WebDAV-Download
vget ls pikpak:/Movies                     # Remote-Verzeichnis auflisten
```

## Unterstützte Quellen

| Quelle         | Typ             | Status       |
| -------------- | --------------- | ------------ |
| Twitter/X      | Video           | Unterstützt  |
| Xiaoyuzhou FM  | Audio (Podcast) | Unterstützt  |
| Apple Podcasts | Audio (Podcast) | Unterstützt  |

## Konfiguration

Speicherort der Konfigurationsdatei:

| OS          | Pfad                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

Führen Sie `vget init` aus, um die Konfigurationsdatei interaktiv zu erstellen, oder erstellen Sie sie manuell:

```yaml
language: de # en, zh, jp, kr, es, fr, de
```

## Sprachen

vget unterstützt mehrere Sprachen:

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## Lizenz

Apache License 2.0
