package extractor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Public bearer token (same as used by web client)
	twitterBearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs=1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	twitterGuestTokenURL  = "https://api.x.com/1.1/guest/activate.json"
	twitterGraphQLURL     = "https://x.com/i/api/graphql/NmCeCgkVlsRGS1cAwqtgmw/TweetResultByRestId"
	twitterSyndicationURL = "https://cdn.syndication.twimg.com/tweet-result"
)

var (
	// Matches twitter.com and x.com URLs with status
	twitterURLRegex = regexp.MustCompile(`(?:twitter\.com|x\.com)/(?:[^/]+)/status/(\d+)`)
)

// TwitterExtractor handles Twitter/X media extraction
type TwitterExtractor struct {
	client     *http.Client
	guestToken string
}

// Name returns the extractor name
func (t *TwitterExtractor) Name() string {
	return "twitter"
}

// Match checks if URL is a Twitter/X status URL
func (t *TwitterExtractor) Match(u *url.URL) bool {
	host := u.Hostname()
	if host != "twitter.com" && host != "www.twitter.com" && host != "x.com" && host != "www.x.com" {
		return false
	}
	// Check path matches /username/status/id pattern
	return twitterURLRegex.MatchString(u.String())
}

// Extract retrieves media from a Twitter/X URL
func (t *TwitterExtractor) Extract(urlStr string) (Media, error) {
	// Initialize HTTP client
	if t.client == nil {
		t.client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Extract tweet ID from URL
	matches := twitterURLRegex.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract tweet ID from URL")
	}
	tweetID := matches[1]

	// Try syndication API first (simpler, no auth needed for public tweets)
	media, err := t.fetchFromSyndication(tweetID)
	if err == nil {
		return media, nil
	}

	// Fallback to GraphQL API
	if err := t.fetchGuestToken(); err != nil {
		return nil, fmt.Errorf("failed to get guest token: %w", err)
	}

	media, err = t.fetchFromGraphQL(tweetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tweet: %w", err)
	}

	return media, nil
}

// fetchFromSyndication tries the syndication endpoint (works for public tweets)
func (t *TwitterExtractor) fetchFromSyndication(tweetID string) (Media, error) {
	params := url.Values{}
	params.Set("id", tweetID)
	params.Set("token", "x") // Required but value doesn't matter

	reqURL := twitterSyndicationURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("syndication request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var data syndicationResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse syndication response: %w", err)
	}

	return t.parseSyndicationResponse(&data, tweetID)
}

// fetchGuestToken obtains a guest token for API access
func (t *TwitterExtractor) fetchGuestToken() error {
	req, err := http.NewRequest("POST", twitterGuestTokenURL, nil)
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

// fetchFromGraphQL uses the GraphQL API
func (t *TwitterExtractor) fetchFromGraphQL(tweetID string) (Media, error) {
	variables := map[string]interface{}{
		"tweetId":                tweetID,
		"withCommunity":          false,
		"includePromotedContent": false,
		"withVoice":              false,
	}

	features := map[string]interface{}{
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"rweb_video_timestamps_enabled":                                           true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	variablesJSON, _ := json.Marshal(variables)
	featuresJSON, _ := json.Marshal(features)

	params := url.Values{}
	params.Set("variables", string(variablesJSON))
	params.Set("features", string(featuresJSON))

	reqURL := twitterGraphQLURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+twitterBearerToken)
	req.Header.Set("x-guest-token", t.guestToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return t.parseGraphQLResponse(body, tweetID)
}

// parseSyndicationResponse extracts media from syndication API response
func (t *TwitterExtractor) parseSyndicationResponse(data *syndicationResponse, tweetID string) (Media, error) {
	if len(data.MediaDetails) == 0 {
		return nil, fmt.Errorf("no media found in tweet")
	}

	title := truncateText(data.Text, 100)
	uploader := data.User.ScreenName

	var videoFormats []VideoFormat
	var images []Image

	for _, media := range data.MediaDetails {
		switch media.Type {
		case "video", "animated_gif":
			for _, variant := range media.VideoInfo.Variants {
				if variant.ContentType != "video/mp4" {
					continue
				}

				format := VideoFormat{
					URL:     variant.URL,
					Ext:     "mp4",
					Bitrate: variant.Bitrate,
				}

				if w, h := extractResolutionFromURL(variant.URL); w > 0 {
					format.Width = w
					format.Height = h
					format.Quality = fmt.Sprintf("%dp", h)
				} else if variant.Bitrate > 0 {
					format.Quality = estimateQualityFromBitrate(variant.Bitrate)
				}

				videoFormats = append(videoFormats, format)
			}

		case "photo":
			imageURL := getHighQualityImageURL(media.MediaURLHTTPS)
			ext := getImageExtension(media.MediaURLHTTPS)

			img := Image{
				URL: imageURL,
				Ext: ext,
			}

			if media.OriginalWidth > 0 {
				img.Width = media.OriginalWidth
				img.Height = media.OriginalHeight
			}

			images = append(images, img)
		}
	}

	// Also check video field directly
	if data.Video.Variants != nil {
		for _, variant := range data.Video.Variants {
			if variant.Type != "video/mp4" {
				continue
			}

			// Check if this URL already exists
			exists := false
			for _, f := range videoFormats {
				if f.URL == variant.Src {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			format := VideoFormat{
				URL: variant.Src,
				Ext: "mp4",
			}

			if w, h := extractResolutionFromURL(variant.Src); w > 0 {
				format.Width = w
				format.Height = h
				format.Quality = fmt.Sprintf("%dp", h)
			}

			videoFormats = append(videoFormats, format)
		}
	}

	// Return appropriate media type
	if len(videoFormats) > 0 {
		// Sort by bitrate/height (highest first)
		sort.Slice(videoFormats, func(i, j int) bool {
			if videoFormats[i].Bitrate != videoFormats[j].Bitrate {
				return videoFormats[i].Bitrate > videoFormats[j].Bitrate
			}
			return videoFormats[i].Height > videoFormats[j].Height
		})

		return &VideoMedia{
			ID:       tweetID,
			Title:    title,
			Uploader: uploader,
			Formats:  videoFormats,
		}, nil
	}

	if len(images) > 0 {
		return &ImageMedia{
			ID:       tweetID,
			Title:    title,
			Uploader: uploader,
			Images:   images,
		}, nil
	}

	return nil, fmt.Errorf("no media found in tweet")
}

// parseGraphQLResponse extracts media from GraphQL API response
func (t *TwitterExtractor) parseGraphQLResponse(body []byte, tweetID string) (Media, error) {
	var resp graphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	result := resp.Data.TweetResult.Result
	if result == nil {
		return nil, fmt.Errorf("tweet not found or not accessible")
	}

	// Handle tweet with visibility results
	legacy := result.Legacy
	if legacy == nil && result.Tweet != nil {
		legacy = result.Tweet.Legacy
	}

	if legacy == nil {
		return nil, fmt.Errorf("could not find tweet data")
	}

	title := truncateText(legacy.FullText, 100)
	var uploader string
	if result.Core != nil && result.Core.UserResults.Result != nil {
		uploader = result.Core.UserResults.Result.Legacy.ScreenName
	}

	if legacy.ExtendedEntities == nil || len(legacy.ExtendedEntities.Media) == 0 {
		return nil, fmt.Errorf("no media found in tweet")
	}

	var videoFormats []VideoFormat
	var images []Image
	var duration int

	for _, media := range legacy.ExtendedEntities.Media {
		switch media.Type {
		case "video", "animated_gif":
			duration = media.VideoInfo.DurationMillis / 1000

			for _, variant := range media.VideoInfo.Variants {
				if variant.ContentType != "video/mp4" {
					continue
				}

				format := VideoFormat{
					URL:     variant.URL,
					Ext:     "mp4",
					Bitrate: variant.Bitrate,
				}

				if w, h := extractResolutionFromURL(variant.URL); w > 0 {
					format.Width = w
					format.Height = h
					format.Quality = fmt.Sprintf("%dp", h)
				} else if variant.Bitrate > 0 {
					format.Quality = estimateQualityFromBitrate(variant.Bitrate)
				}

				videoFormats = append(videoFormats, format)
			}

		case "photo":
			imageURL := getHighQualityImageURL(media.MediaURLHTTPS)
			ext := getImageExtension(media.MediaURLHTTPS)

			img := Image{
				URL: imageURL,
				Ext: ext,
			}

			if media.OriginalInfo.Width > 0 {
				img.Width = media.OriginalInfo.Width
				img.Height = media.OriginalInfo.Height
			}

			images = append(images, img)
		}
	}

	// Return appropriate media type
	if len(videoFormats) > 0 {
		sort.Slice(videoFormats, func(i, j int) bool {
			return videoFormats[i].Bitrate > videoFormats[j].Bitrate
		})

		return &VideoMedia{
			ID:       tweetID,
			Title:    title,
			Uploader: uploader,
			Duration: duration,
			Formats:  videoFormats,
		}, nil
	}

	if len(images) > 0 {
		return &ImageMedia{
			ID:       tweetID,
			Title:    title,
			Uploader: uploader,
			Images:   images,
		}, nil
	}

	return nil, fmt.Errorf("no media found in tweet")
}

// Syndication API response structures
type syndicationResponse struct {
	Text string `json:"text"`
	User struct {
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
	} `json:"user"`
	MediaDetails []struct {
		Type           string `json:"type"`
		MediaURLHTTPS  string `json:"media_url_https"`
		OriginalWidth  int    `json:"original_info_width"`
		OriginalHeight int    `json:"original_info_height"`
		VideoInfo      struct {
			Variants []struct {
				Bitrate     int    `json:"bitrate"`
				ContentType string `json:"content_type"`
				URL         string `json:"url"`
			} `json:"variants"`
		} `json:"video_info"`
	} `json:"mediaDetails"`
	Video struct {
		Variants []struct {
			Type string `json:"type"`
			Src  string `json:"src"`
		} `json:"variants"`
	} `json:"video"`
}

// GraphQL API response structures
type graphQLResponse struct {
	Data struct {
		TweetResult struct {
			Result *graphQLTweetResult `json:"result"`
		} `json:"tweetResult"`
	} `json:"data"`
}

type graphQLTweetResult struct {
	TypeName string              `json:"__typename"`
	Legacy   *graphQLLegacy      `json:"legacy"`
	Core     *graphQLCore        `json:"core"`
	Tweet    *graphQLTweetResult `json:"tweet"` // For TweetWithVisibilityResults
}

type graphQLCore struct {
	UserResults struct {
		Result *struct {
			Legacy struct {
				ScreenName string `json:"screen_name"`
			} `json:"legacy"`
		} `json:"result"`
	} `json:"user_results"`
}

type graphQLLegacy struct {
	FullText         string `json:"full_text"`
	ExtendedEntities *struct {
		Media []struct {
			Type          string `json:"type"`
			MediaURLHTTPS string `json:"media_url_https"`
			OriginalInfo  struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"original_info"`
			VideoInfo struct {
				DurationMillis int `json:"duration_millis"`
				Variants       []struct {
					Bitrate     int    `json:"bitrate"`
					ContentType string `json:"content_type"`
					URL         string `json:"url"`
				} `json:"variants"`
			} `json:"video_info"`
		} `json:"media"`
	} `json:"extended_entities"`
}

// Helper functions

func truncateText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
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

// getHighQualityImageURL converts a Twitter image URL to highest quality version
func getHighQualityImageURL(imageURL string) string {
	baseURL := strings.Split(imageURL, "?")[0]

	format := "jpg"
	if strings.Contains(baseURL, ".png") {
		format = "png"
	} else if strings.Contains(baseURL, ".webp") {
		format = "webp"
	}

	return baseURL + "?format=" + format + "&name=orig"
}

// getImageExtension extracts the image extension from URL
func getImageExtension(imageURL string) string {
	baseURL := strings.Split(imageURL, "?")[0]
	if strings.HasSuffix(baseURL, ".png") {
		return "png"
	} else if strings.HasSuffix(baseURL, ".webp") {
		return "webp"
	} else if strings.HasSuffix(baseURL, ".gif") {
		return "gif"
	}
	return "jpg"
}
