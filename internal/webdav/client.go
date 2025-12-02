package webdav

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/emersion/go-webdav"
	"github.com/guiyumin/vget/internal/config"
)

// Client wraps go-webdav client with convenience methods
type Client struct {
	client  *webdav.Client
	baseURL string
}

// FileInfo contains information about a remote file
type FileInfo struct {
	Name string
	Path string
	Size int64
	IsDir bool
}

// NewClient creates a new WebDAV client
// URL format: webdav://user:pass@host/path or https://user:pass@host/path
func NewClient(rawURL string) (*Client, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Convert webdav:// to https://
	scheme := parsed.Scheme
	if scheme == "webdav" {
		scheme = "https"
	} else if scheme == "webdav+http" {
		scheme = "http"
	}

	// Build base URL without credentials and path
	baseURL := fmt.Sprintf("%s://%s", scheme, parsed.Host)

	// Extract credentials and create HTTP client
	var httpClient webdav.HTTPClient
	if parsed.User != nil {
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		httpClient = webdav.HTTPClientWithBasicAuth(nil, username, password)
	}

	client, err := webdav.NewClient(httpClient, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebDAV client: %w", err)
	}

	return &Client{
		client:  client,
		baseURL: baseURL,
	}, nil
}

// ParseURL extracts the file path from a WebDAV URL
func ParseURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsed.Path, nil
}

// Stat returns information about a file
func (c *Client) Stat(ctx context.Context, filePath string) (*FileInfo, error) {
	info, err := c.client.Stat(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", filePath, err)
	}

	return &FileInfo{
		Name:  path.Base(info.Path),
		Path:  info.Path,
		Size:  info.Size,
		IsDir: info.IsDir,
	}, nil
}

// List returns the contents of a directory
func (c *Client) List(ctx context.Context, dirPath string) ([]FileInfo, error) {
	infos, err := c.client.ReadDir(ctx, dirPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", dirPath, err)
	}

	result := make([]FileInfo, 0, len(infos))
	for _, info := range infos {
		name := path.Base(info.Path)
		result = append(result, FileInfo{
			Name:  name,
			Path:  info.Path,
			Size:  info.Size,
			IsDir: info.IsDir,
		})
	}
	return result, nil
}

// Open opens a file for reading and returns the reader and file size
func (c *Client) Open(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	// First get the file size
	info, err := c.client.Stat(ctx, filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat %s: %w", filePath, err)
	}

	if info.IsDir {
		return nil, 0, fmt.Errorf("%s is a directory", filePath)
	}

	reader, err := c.client.Open(ctx, filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open %s: %w", filePath, err)
	}

	return reader, info.Size, nil
}

// IsWebDAVURL checks if a URL is a WebDAV URL or a remote path (remote:path)
func IsWebDAVURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "webdav://") ||
		strings.HasPrefix(rawURL, "webdav+http://") ||
		IsRemotePath(rawURL)
}

// IsRemotePath checks if the URL is a remote path format (e.g., "pikpak:/path/to/file")
func IsRemotePath(rawURL string) bool {
	// Check for remote:path format (not a URL scheme like http://)
	if idx := strings.Index(rawURL, ":"); idx > 0 {
		prefix := rawURL[:idx]
		// Make sure it's not a URL scheme (no slashes after colon at position idx+1)
		if idx+1 < len(rawURL) && rawURL[idx+1] != '/' {
			return true
		}
		// Also match remote:/path (single slash for absolute path)
		if idx+2 < len(rawURL) && rawURL[idx+1] == '/' && rawURL[idx+2] != '/' {
			return true
		}
		// Check if prefix looks like a remote name (no dots, not a known scheme)
		if !strings.Contains(prefix, ".") &&
			prefix != "http" && prefix != "https" &&
			prefix != "webdav" && prefix != "webdav+http" {
			return true
		}
	}
	return false
}

// ParseRemotePath parses a remote path like "pikpak:/path/to/file" into remote name and path
func ParseRemotePath(remotePath string) (remoteName, filePath string, err error) {
	idx := strings.Index(remotePath, ":")
	if idx <= 0 {
		return "", "", fmt.Errorf("invalid remote path format: %s", remotePath)
	}
	remoteName = remotePath[:idx]
	filePath = remotePath[idx+1:]

	// Ensure path starts with /
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return remoteName, filePath, nil
}

// NewClientFromConfig creates a WebDAV client from a configured server
func NewClientFromConfig(server *config.WebDAVServer) (*Client, error) {
	var httpClient webdav.HTTPClient
	if server.Username != "" {
		httpClient = webdav.HTTPClientWithBasicAuth(nil, server.Username, server.Password)
	}

	client, err := webdav.NewClient(httpClient, server.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebDAV client: %w", err)
	}

	return &Client{
		client:  client,
		baseURL: server.URL,
	}, nil
}

// ExtractFilename extracts the filename from a WebDAV path
func ExtractFilename(filePath string) string {
	return path.Base(filePath)
}
