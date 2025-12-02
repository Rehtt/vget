# TODO

## Tomorrow's Tasks

1. [ ] Self update - implement `vget update` command
2. [ ] Support m3u8 streaming format
3. [ ] GoReleaser setup
4. [ ] Cross-platform builds:
   - macOS (Intel x86_64, Apple Silicon arm64)
   - Windows (Intel x86_64, ARM arm64)
   - Linux (x86, amd64, arm64)

## Features

- [x] `vget init` command
  - Language preference
  - Proxy settings
  - Default output directory
  - Default format/quality
- [ ] Self update
- [ ] m3u8 streaming support
- [ ] Bulk download from txt file
  - Read URLs from txt file
  - Sequential or parallel processing
- [x] Format/quality selection (`-q` flag)
- [x] Audio extraction (podcasts)
- [ ] Custom output path/filename template
- [ ] Resume interrupted downloads
- [ ] Retry on failure
- [x] Progress bar with speed/ETA
- [ ] Quiet/verbose modes
- [ ] Dry run mode
- [ ] More extractors (YouTube, TikTok, etc.)
- [ ] Playlist support
- [ ] Concurrent downloads
- [ ] Rate limiting
- [ ] Cookie/auth support
- [ ] Metadata embedding
- [ ] WebDAV client integration
  - Connect to PikPak, other WebDAV-compatible cloud storage
  - Upload downloaded files directly to cloud
  - Lighter alternative to rclone for single-purpose use

## Extractors

- [x] Twitter/X
- [x] Xiaoyuzhou (小宇宙) podcasts
  - [x] Episode download
  - [x] Search (`vget search --podcast <query>`)
  - [ ] Podcast listing (all episodes)
- [ ] YouTube
- [ ] TikTok
- [ ] Apple Podcasts
- [ ] Xiaohongshu (小红书/RED)
  - Requires browser automation (Rod) + cookie auth
  - Reference: [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp) (7.2k stars, stable 1+ year)
  - Extraction approach:
    - Navigate to `https://www.xiaohongshu.com/explore/{feedID}?xsec_token=...`
    - Extract `window.__INITIAL_STATE__.note.noteDetailMap` via JS
    - Parse JSON for images (`urlDefault`) and video URLs
  - Feasibility: Moderate effort, more achievable than Instagram
  - Note: yt-dlp also has extractor but frequently breaks due to bot detection

## Tracking (Versatile Get)

- [ ] FedEx tracking
  - [ ] Scraping (default, no setup)
  - [ ] API mode (user provides own keys in config.yml)
- [ ] UPS tracking
  - [ ] Scraping (default, no setup)
  - [ ] API mode (user provides own keys in config.yml)
- [ ] USPS tracking
  - [ ] Scraping (default, no setup)
  - [ ] API mode (user provides own keys in config.yml)

## DevOps

- [ ] GoReleaser + GitHub Actions for tagged releases
