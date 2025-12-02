# WebDAV Support

vget supports downloading files from WebDAV servers and browsing remote directories.

## Configuration

### Add a WebDAV Server

```bash
vget config webdav add <name>
```

Interactive prompts for:
- WebDAV URL (e.g., `https://dav.example.com`)
- Username (optional)
- Password (masked input)

### Manage Servers

```bash
vget config webdav list              # List all configured servers
vget config webdav show <name>       # Show server details
vget config webdav delete <name>     # Remove a server
```

## Commands

### Download Files

```bash
# Using configured remote
vget pikpak:/path/to/file.mp4

# Using full URL with credentials
vget webdav://user:pass@server.com/path/file.mp4

# With output filename
vget pikpak:/movies/video.mp4 -o my_video.mp4

# Show file metadata
vget --info pikpak:/movies/video.mp4
```

### List Directory

```bash
vget ls pikpak:/movies
vget ls pikpak:/                      # List root directory
vget ls pikpak:/movies --json         # JSON output for scripting
```

Output format:
```
pikpak:/movies
  📁 Action/
  📁 Comedy/
  📄 movie.mp4                    1.5 GB
  📄 readme.txt                   2.3 KB
```

JSON output (`--json`):
```json
[
  {"name": "Action", "path": "pikpak:/movies/Action", "is_dir": true, "size": 0},
  {"name": "movie.mp4", "path": "pikpak:/movies/movie.mp4", "is_dir": false, "size": 1610612736}
]
```

### Bulk Download with Pipe

Use `--json` with `jq` to download all files in a directory:

```bash
# Download all files (skip directories)
vget ls pikpak:/movies --json | jq -r '.[] | select(.is_dir == false) | .path' | xargs -n1 vget

# Download only files > 1GB
vget ls pikpak:/movies --json | jq -r '.[] | select(.is_dir == false and .size > 1073741824) | .path' | xargs -n1 vget
```

### Command Behavior

| Command | File | Directory |
|---------|------|-----------|
| `vget <url>` | Download | Error (use `vget ls`) |
| `vget ls <url>` | Error (not a directory) | List contents |
| `vget --info <url>` | Show metadata | Show metadata |

## URL Schemes

| Scheme | Protocol |
|--------|----------|
| `webdav://` | HTTPS (default) |
| `webdav+http://` | HTTP (insecure) |
| `https://` | HTTPS |

## Shell Completion

Enable tab completion for remote paths:

```bash
# Bash - add to ~/.bashrc
source <(vget completion bash)

# Zsh - add to ~/.zshrc
source <(vget completion zsh)

# Fish
vget completion fish > ~/.config/fish/completions/vget.fish

# PowerShell
vget completion powershell >> $PROFILE
```

After setup, tab completion works for remote paths:
```bash
vget pikpak:/Mo<TAB>           # Completes to pikpak:/Movies/
vget ls pikpak:/Movies/<TAB>   # Shows files in Movies/
```

## Examples

```bash
# Setup
vget config webdav add pikpak
# Enter URL: https://dav.pikpak.com
# Enter username: user@example.com
# Enter password: ****

# Browse
vget ls pikpak:/
vget ls pikpak:/Movies

# Download
vget pikpak:/Movies/film.mp4
vget pikpak:/Movies/film.mp4 -o ~/Downloads/film.mp4

# Info
vget --info pikpak:/Movies/film.mp4
```
