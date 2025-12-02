package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// MultiStreamConfig configures multi-stream downloads
type MultiStreamConfig struct {
	Streams    int   // Number of parallel streams (default 8)
	ChunkSize  int64 // Size of each chunk in bytes (default 16MB)
	BufferSize int   // Buffer size per stream (default 128KB)
}

// DefaultMultiStreamConfig returns sensible defaults similar to rclone
func DefaultMultiStreamConfig() MultiStreamConfig {
	return MultiStreamConfig{
		Streams:    8,                  // 8 parallel streams
		ChunkSize:  16 * 1024 * 1024,   // 16MB chunks
		BufferSize: 128 * 1024,         // 128KB buffer per stream
	}
}

// multiStreamState tracks progress across all streams
type multiStreamState struct {
	downloaded int64 // atomic counter for total bytes downloaded
	total      int64
	startTime  time.Time
	mu         sync.RWMutex
	errors     []error
}

func (s *multiStreamState) addBytes(n int64) {
	atomic.AddInt64(&s.downloaded, n)
}

func (s *multiStreamState) getDownloaded() int64 {
	return atomic.LoadInt64(&s.downloaded)
}

func (s *multiStreamState) addError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
}

func (s *multiStreamState) getErrors() []error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errors
}

// chunk represents a portion of the file to download
type chunk struct {
	index int
	start int64
	end   int64 // inclusive
}

// MultiStreamDownload downloads a file using multiple parallel HTTP Range requests
func MultiStreamDownload(ctx context.Context, url, output string, config MultiStreamConfig, state *downloadState) error {
	// Create HTTP client with no timeout (we handle it per-request)
	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			MaxIdleConns:        config.Streams * 2,
			MaxIdleConnsPerHost: config.Streams * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// First, get the file size with a HEAD request
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HEAD request failed: %w", err)
	}
	resp.Body.Close()

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		return fmt.Errorf("server did not return Content-Length")
	}

	// Check if server supports Range requests
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		// Fall back to single-stream download
		return downloadWithProgress(client, url, output, state)
	}

	state.update(0, totalSize)

	// Create the output file
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Pre-allocate file size for efficiency
	if err := file.Truncate(totalSize); err != nil {
		// Non-fatal, continue anyway
	}

	// Calculate chunks
	chunks := calculateChunks(totalSize, config.Streams, config.ChunkSize)

	// Create multi-stream state
	msState := &multiStreamState{
		total:     totalSize,
		startTime: state.startTime,
	}

	// Start progress updater goroutine
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-ticker.C:
				state.update(msState.getDownloaded(), totalSize)
			}
		}
	}()

	// Download chunks in parallel using a worker pool
	var wg sync.WaitGroup
	chunkChan := make(chan chunk, len(chunks))

	// Feed chunks to the channel
	for _, c := range chunks {
		chunkChan <- c
	}
	close(chunkChan)

	// Start worker goroutines
	for i := 0; i < config.Streams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range chunkChan {
				if err := downloadChunk(ctx, client, url, file, c, config.BufferSize, msState); err != nil {
					msState.addError(fmt.Errorf("chunk %d failed: %w", c.index, err))
				}
			}
		}()
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(progressDone)

	// Final progress update
	state.update(msState.getDownloaded(), totalSize)

	// Check for errors
	if errs := msState.getErrors(); len(errs) > 0 {
		return fmt.Errorf("download failed with %d errors: %v", len(errs), errs[0])
	}

	return nil
}

// calculateChunks divides the file into download chunks
func calculateChunks(totalSize int64, streams int, chunkSize int64) []chunk {
	var chunks []chunk

	// If file is small, just use one chunk
	if totalSize <= chunkSize {
		return []chunk{{index: 0, start: 0, end: totalSize - 1}}
	}

	// Calculate number of chunks needed
	numChunks := (totalSize + chunkSize - 1) / chunkSize

	// Limit to reasonable number based on streams
	maxChunks := int64(streams * 4) // Allow some queue depth
	if numChunks > maxChunks {
		// Recalculate chunk size to fit within maxChunks
		chunkSize = (totalSize + maxChunks - 1) / maxChunks
		numChunks = maxChunks
	}

	var start int64
	for i := int64(0); i < numChunks; i++ {
		end := start + chunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
		chunks = append(chunks, chunk{
			index: int(i),
			start: start,
			end:   end,
		})
		start = end + 1
		if start >= totalSize {
			break
		}
	}

	return chunks
}

// downloadChunk downloads a single chunk using HTTP Range request
func downloadChunk(ctx context.Context, client *http.Client, url string, file *os.File, c chunk, bufferSize int, state *multiStreamState) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	buf := make([]byte, bufferSize)
	offset := c.start

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			// Write at specific offset (thread-safe with pwrite)
			written, writeErr := file.WriteAt(buf[:n], offset)
			if writeErr != nil {
				return fmt.Errorf("write failed: %w", writeErr)
			}
			offset += int64(written)
			state.addBytes(int64(written))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read failed: %w", readErr)
		}
	}

	return nil
}

// RunMultiStreamDownloadTUI runs a multi-stream download with TUI progress
func RunMultiStreamDownloadTUI(url, output, displayID, lang string, config MultiStreamConfig) error {
	state := &downloadState{
		startTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start download in background
	go func() {
		err := MultiStreamDownload(ctx, url, output, config, state)
		if err != nil {
			state.setError(err)
		} else {
			state.setDone()
		}
	}()

	model := newDownloadModel(output, displayID, lang, state)

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	m := finalModel.(downloadModel)
	_, _, _, _, downloadErr := m.state.get()
	if downloadErr != nil {
		return downloadErr
	}

	return nil
}

// MultiStreamDownloadWithAuth downloads a file using multiple parallel HTTP Range requests with auth
func MultiStreamDownloadWithAuth(ctx context.Context, url, authHeader, output string, totalSize int64, config MultiStreamConfig, state *downloadState) error {
	// Create HTTP client
	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			MaxIdleConns:        config.Streams * 2,
			MaxIdleConnsPerHost: config.Streams * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// First check if server supports Range requests
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HEAD request failed: %w", err)
	}
	resp.Body.Close()

	// Check if server supports Range requests
	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange := acceptRanges == "bytes"

	state.update(0, totalSize)

	// If no Range support, fall back to single-stream
	if !supportsRange {
		return downloadWithAuthSingleStream(ctx, client, url, authHeader, output, totalSize, state)
	}

	// Create the output file
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Pre-allocate file size for efficiency
	if err := file.Truncate(totalSize); err != nil {
		// Non-fatal, continue anyway
	}

	// Calculate chunks
	chunks := calculateChunks(totalSize, config.Streams, config.ChunkSize)

	// Create multi-stream state
	msState := &multiStreamState{
		total:     totalSize,
		startTime: state.startTime,
	}

	// Start progress updater goroutine
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-ticker.C:
				state.update(msState.getDownloaded(), totalSize)
			}
		}
	}()

	// Download chunks in parallel using a worker pool
	var wg sync.WaitGroup
	chunkChan := make(chan chunk, len(chunks))

	// Feed chunks to the channel
	for _, c := range chunks {
		chunkChan <- c
	}
	close(chunkChan)

	// Start worker goroutines
	for i := 0; i < config.Streams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range chunkChan {
				if err := downloadChunkWithAuth(ctx, client, url, authHeader, file, c, config.BufferSize, msState); err != nil {
					msState.addError(fmt.Errorf("chunk %d failed: %w", c.index, err))
				}
			}
		}()
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(progressDone)

	// Final progress update
	state.update(msState.getDownloaded(), totalSize)

	// Check for errors
	if errs := msState.getErrors(); len(errs) > 0 {
		return fmt.Errorf("download failed with %d errors: %v", len(errs), errs[0])
	}

	return nil
}

// downloadChunkWithAuth downloads a single chunk using HTTP Range request with auth
// It includes retry logic for transient failures
func downloadChunkWithAuth(ctx context.Context, client *http.Client, url, authHeader string, file *os.File, c chunk, bufferSize int, state *multiStreamState) error {
	const maxRetries = 5
	var lastErr error
	var previousAttemptBytes int64

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Subtract previously counted bytes since we're retrying the whole chunk
			if previousAttemptBytes > 0 {
				state.addBytes(-previousAttemptBytes)
				previousAttemptBytes = 0
			}

			// Exponential backoff: 1s, 2s, 4s, 8s, 16s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		bytesWritten, err := downloadChunkWithAuthOnce(ctx, client, url, authHeader, file, c, bufferSize, state)
		if err == nil {
			return nil
		}
		lastErr = err
		previousAttemptBytes = bytesWritten

		// Check if context was cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// downloadChunkWithAuthOnce performs a single attempt to download a chunk
// Returns bytes written and any error. Updates state in real-time for progress display.
func downloadChunkWithAuthOnce(ctx context.Context, client *http.Client, url, authHeader string, file *os.File, c chunk, bufferSize int, state *multiStreamState) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	buf := make([]byte, bufferSize)
	offset := c.start
	expectedEnd := c.end + 1 // end is inclusive, so we expect to read up to end+1
	var totalWritten int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			// Write at specific offset (thread-safe with pwrite)
			written, writeErr := file.WriteAt(buf[:n], offset)
			if writeErr != nil {
				return totalWritten, fmt.Errorf("write failed: %w", writeErr)
			}
			offset += int64(written)
			totalWritten += int64(written)
			// Update progress in real-time
			state.addBytes(int64(written))
		}
		if readErr == io.EOF {
			// Verify we got the full chunk
			if offset < expectedEnd {
				return totalWritten, fmt.Errorf("incomplete chunk: got %d bytes, expected %d", offset-c.start, expectedEnd-c.start)
			}
			break
		}
		if readErr != nil {
			return totalWritten, fmt.Errorf("read failed: %w", readErr)
		}
	}

	return totalWritten, nil
}

// downloadWithAuthSingleStream falls back to single-stream download when Range not supported
func downloadWithAuthSingleStream(ctx context.Context, client *http.Client, url, authHeader, output string, total int64, state *downloadState) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Download with progress tracking
	buf := make([]byte, 128*1024) // 128KB buffer
	var current int64

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write file: %w", writeErr)
			}
			current += int64(n)
			state.update(current, total)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	}

	return nil
}

// RunMultiStreamDownloadWithAuthTUI runs a multi-stream download with auth and TUI progress
func RunMultiStreamDownloadWithAuthTUI(url, authHeader, output, displayID, lang string, totalSize int64, config MultiStreamConfig) error {
	state := &downloadState{
		startTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start download in background
	go func() {
		err := MultiStreamDownloadWithAuth(ctx, url, authHeader, output, totalSize, config, state)
		if err != nil {
			state.setError(err)
		} else {
			state.setDone()
		}
	}()

	model := newDownloadModel(output, displayID, lang, state)

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	m := finalModel.(downloadModel)
	_, _, _, _, downloadErr := m.state.get()
	if downloadErr != nil {
		return downloadErr
	}

	return nil
}
