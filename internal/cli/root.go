package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/guiyumin/vget/internal/config"
	"github.com/guiyumin/vget/internal/downloader"
	"github.com/guiyumin/vget/internal/extractor"
	"github.com/guiyumin/vget/internal/i18n"
	"github.com/guiyumin/vget/internal/version"
	"github.com/spf13/cobra"
)

var (
	output  string
	quality string
	info    bool
)

var rootCmd = &cobra.Command{
	Use:     "vget [url]",
	Short:   "A modern, blazing-fast, cross-platform downloader cli",
	Version: version.Version,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		if err := runDownload(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "output filename")
	rootCmd.Flags().StringVarP(&quality, "quality", "q", "", "preferred quality (e.g., 1080p, 720p)")
	rootCmd.Flags().BoolVar(&info, "info", false, "show video info without downloading")
}

func Execute() error {
	return rootCmd.Execute()
}

func runDownload(url string) error {
	cfg := config.LoadOrDefault()
	t := i18n.T(cfg.Language)

	// Check for config file and warn if missing
	if !config.Exists() {
		fmt.Fprintf(os.Stderr, "\033[33m%s. Run 'vget init'.\033[0m\n", t.Errors.ConfigNotFound)
	}

	// Find matching extractor
	ext := extractor.Match(url)
	if ext == nil {
		return fmt.Errorf("%s: %s", t.Errors.NoExtractor, url)
	}

	// Extract video info with spinner
	videoInfo, err := runExtractWithSpinner(ext, url, cfg.Language)
	if err != nil {
		return err
	}

	// Info only mode
	if info {
		for i, f := range videoInfo.Formats {
			fmt.Printf("  [%d] %s %dx%d (%s)\n", i, f.Quality, f.Width, f.Height, f.Ext)
		}
		return nil
	}

	// Select best format (or by quality flag)
	format := selectFormat(videoInfo.Formats, quality)
	if format == nil {
		return errors.New(t.Download.NoFormats)
	}

	fmt.Printf("  %s: %s (%s)\n", t.Download.SelectedFormat, format.Quality, format.Ext)

	// Determine output filename
	outputFile := output
	if outputFile == "" {
		outputFile = fmt.Sprintf("%s.%s", videoInfo.ID, format.Ext)
	}

	// Download
	dl := downloader.New(cfg.Language)
	return dl.Download(format.URL, outputFile, videoInfo.ID)
}

func selectFormat(formats []extractor.Format, preferred string) *extractor.Format {
	if len(formats) == 0 {
		return nil
	}

	// If quality specified, try to match
	if preferred != "" {
		for i := range formats {
			if formats[i].Quality == preferred {
				return &formats[i]
			}
		}
	}

	// Otherwise return highest bitrate
	best := &formats[0]
	for i := range formats {
		if formats[i].Bitrate > best.Bitrate {
			best = &formats[i]
		}
	}
	return best
}
