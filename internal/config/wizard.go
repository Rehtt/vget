package config

import (
	"github.com/charmbracelet/huh"
)

// RunInitWizard runs an interactive TUI wizard to configure vget
func RunInitWizard() (*Config, error) {
	cfg := DefaultConfig()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Language").
				Description("Preferred language for metadata").
				Options(
					huh.NewOption("English", "en"),
					huh.NewOption("中文", "zh"),
					huh.NewOption("日本語", "ja"),
					huh.NewOption("한국어", "ko"),
					huh.NewOption("Español", "es"),
					huh.NewOption("Français", "fr"),
					huh.NewOption("Deutsch", "de"),
				).
				Value(&cfg.Language),

			huh.NewInput().
				Title("Proxy").
				Description("Leave empty for no proxy (e.g., http://127.0.0.1:7890)").
				Placeholder("http://127.0.0.1:7890").
				Value(&cfg.Proxy),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Output Directory").
				Description("Where to save downloaded videos").
				Placeholder(".").
				Value(&cfg.OutputDir),

			huh.NewSelect[string]().
				Title("Preferred Format").
				Description("Video format to download").
				Options(
					huh.NewOption("MP4 (recommended)", "mp4"),
					huh.NewOption("WebM", "webm"),
					huh.NewOption("MKV", "mkv"),
					huh.NewOption("Best available", "best"),
				).
				Value(&cfg.Format),

			huh.NewSelect[string]().
				Title("Preferred Quality").
				Description("Video quality to download").
				Options(
					huh.NewOption("Best available", "best"),
					huh.NewOption("4K (2160p)", "2160p"),
					huh.NewOption("1080p", "1080p"),
					huh.NewOption("720p", "720p"),
					huh.NewOption("480p", "480p"),
				).
				Value(&cfg.Quality),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	// Set defaults for empty values
	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}

	return cfg, nil
}
