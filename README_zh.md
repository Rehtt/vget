# vget

多功能命令行下载工具，支持音频、视频、播客等。

[English](README.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## 安装

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

从 [Releases](https://github.com/guiyumin/vget/releases/latest) 下载 `vget-windows-amd64.exe` 并添加到系统 PATH。

## 命令

| 命令                               | 描述                                  |
|------------------------------------|---------------------------------------|
| `vget [url]`                       | 下载媒体 (`-o`, `-q`, `--info`)       |
| `vget ls <remote>:<path>`          | 列出远程目录 (`--json`)               |
| `vget init`                        | 交互式配置向导                        |
| `vget update`                      | 自动更新                              |
| `vget search --podcast <query>`    | 搜索播客                              |
| `vget completion [shell]`          | 生成 shell 补全脚本                   |
| `vget config show`                 | 显示配置                              |
| `vget config path`                 | 显示配置文件路径                      |
| `vget config webdav list`          | 列出已配置的 WebDAV 服务器            |
| `vget config webdav add <name>`    | 添加 WebDAV 服务器                    |
| `vget config webdav show <name>`   | 显示服务器详情                        |
| `vget config webdav delete <name>` | 删除服务器                            |

### 示例

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o my_video.mp4
vget --info https://example.com/video
vget search --podcast "科技"
vget pikpak:/path/to/file.mp4              # WebDAV 下载
vget ls pikpak:/Movies                     # 列出远程目录
```

## 支持的来源

| 来源           | 类型            | 状态   |
| -------------- | --------------- | ------ |
| Twitter/X      | 视频            | 已支持 |
| 小宇宙 FM      | 音频（播客）    | 已支持 |
| Apple Podcasts | 音频（播客）    | 已支持 |

## 配置

配置文件位置：

| 操作系统    | 路径                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

运行 `vget init` 交互式创建配置文件，或手动创建：

```yaml
language: zh # en, zh, jp, kr, es, fr, de
```

## 语言

vget 支持多种语言：

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## 许可证

Apache License 2.0
