package extractor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Public bearer token (same as used by web client)
	twitterBearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs=1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	twitterAPIBase = "https://api.x.com"
)

var (
	// Matches twitter.com and x.com URLs with status
	twitterURLRegex = regexp.MustCompile(`(?:twitter\.com|x\.com)/(?:[^/]+)/status/(\d+)`)
)

// TwitterExtractor handles Twitter/X video extraction
type TwitterExtractor struct {
	client     *http.Client
	guestToken string
}

// Name returns the extractor name
func (t *TwitterExtractor) Name() string {
	return "twitter"
}

// Match checks if URL is a Twitter/X status URL
func (t *TwitterExtractor) Match(url string) bool {
	return twitterURLRegex.MatchString(url)
}

// Extract retrieves video information from a Twitter/X URL
func (t *TwitterExtractor) Extract(url string) (*VideoInfo, error) {
	// Initialize HTTP client
	if t.client == nil {
		t.client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Extract tweet ID from URL
	matches := twitterURLRegex.FindStringSubmatch(url)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract tweet ID from URL")
	}
	tweetID := matches[1]

	// Get guest token
	if err := t.fetchGuestToken(); err != nil {
		return nil, fmt.Errorf("failed to get guest token: %w", err)
	}

	// Fetch tweet data
	tweet, err := t.fetchTweet(tweetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tweet: %w", err)
	}

	return tweet, nil
}

// fetchGuestToken obtains a guest token for API access
func (t *TwitterExtractor) fetchGuestToken() error {
	req, err := http.NewRequest("POST", twitterAPIBase+"/1.1/guest/activate.json", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+twitterBearerToken)

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("guest token request failed with status %d", resp.StatusCode)
	}

	var result struct {
		GuestToken string `json:"guest_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	t.guestToken = result.GuestToken
	return nil
}

// fetchTweet retrieves tweet data including video info
func (t *TwitterExtractor) fetchTweet(tweetID string) (*VideoInfo, error) {
	url := fmt.Sprintf("%s/1.1/statuses/show/%s.json?include_entities=true&tweet_mode=extended", twitterAPIBase, tweetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+twitterBearerToken)
	req.Header.Set("x-guest-token", t.guestToken)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tweet fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tweet twitterTweetResponse
	if err := json.NewDecoder(resp.Body).Decode(&tweet); err != nil {
		return nil, fmt.Errorf("failed to parse tweet response: %w", err)
	}

	return t.parseTweet(&tweet)
}

// parseTweet extracts video info from tweet response
func (t *TwitterExtractor) parseTweet(tweet *twitterTweetResponse) (*VideoInfo, error) {
	info := &VideoInfo{
		ID:       tweet.IDStr,
		Title:    truncateText(tweet.FullText, 100),
		Uploader: tweet.User.ScreenName,
	}

	// Find video in extended_entities
	if tweet.ExtendedEntities == nil || len(tweet.ExtendedEntities.Media) == 0 {
		return nil, fmt.Errorf("no media found in tweet")
	}

	for _, media := range tweet.ExtendedEntities.Media {
		if media.Type != "video" && media.Type != "animated_gif" {
			continue
		}

		info.Thumbnail = media.MediaURLHTTPS
		info.Duration = media.VideoInfo.DurationMillis / 1000

		// Extract video formats
		for _, variant := range media.VideoInfo.Variants {
			if variant.ContentType != "video/mp4" {
				// Skip HLS for now, we'll add support later
				continue
			}

			format := Format{
				URL:     variant.URL,
				Ext:     "mp4",
				Bitrate: variant.Bitrate,
			}

			// Try to extract resolution from URL
			// Twitter URLs often contain resolution like /vid/1280x720/
			if w, h := extractResolutionFromURL(variant.URL); w > 0 {
				format.Width = w
				format.Height = h
				format.Quality = fmt.Sprintf("%dp", h)
			} else if variant.Bitrate > 0 {
				// Estimate quality from bitrate
				format.Quality = estimateQualityFromBitrate(variant.Bitrate)
			}

			info.Formats = append(info.Formats, format)
		}
	}

	if len(info.Formats) == 0 {
		return nil, fmt.Errorf("no video formats found in tweet")
	}

	// Sort by bitrate (highest first)
	sort.Slice(info.Formats, func(i, j int) bool {
		return info.Formats[i].Bitrate > info.Formats[j].Bitrate
	})

	return info, nil
}

// Twitter API response structures
type twitterTweetResponse struct {
	IDStr            string                   `json:"id_str"`
	FullText         string                   `json:"full_text"`
	User             twitterUser              `json:"user"`
	ExtendedEntities *twitterExtendedEntities `json:"extended_entities"`
}

type twitterUser struct {
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
}

type twitterExtendedEntities struct {
	Media []twitterMedia `json:"media"`
}

type twitterMedia struct {
	Type          string           `json:"type"`
	MediaURLHTTPS string           `json:"media_url_https"`
	VideoInfo     twitterVideoInfo `json:"video_info"`
}

type twitterVideoInfo struct {
	DurationMillis int                    `json:"duration_millis"`
	Variants       []twitterVideoVariant  `json:"variants"`
}

type twitterVideoVariant struct {
	Bitrate     int    `json:"bitrate"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

// Helper functions

func truncateText(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var resolutionRegex = regexp.MustCompile(`/(\d+)x(\d+)/`)

func extractResolutionFromURL(url string) (width, height int) {
	matches := resolutionRegex.FindStringSubmatch(url)
	if len(matches) >= 3 {
		w, _ := strconv.Atoi(matches[1])
		h, _ := strconv.Atoi(matches[2])
		return w, h
	}
	return 0, 0
}

func estimateQualityFromBitrate(bitrate int) string {
	switch {
	case bitrate >= 2000000:
		return "1080p"
	case bitrate >= 1000000:
		return "720p"
	case bitrate >= 500000:
		return "480p"
	default:
		return "360p"
	}
}
