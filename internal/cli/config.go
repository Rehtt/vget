package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/guiyumin/vget/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage vget configuration",
	Long:  "View and modify vget settings, including WebDAV remotes",
}

// vget config show - show current config
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadOrDefault()

		fmt.Println("Current configuration:")
		fmt.Printf("  Language:  %s\n", cfg.Language)
		fmt.Printf("  Proxy:     %s\n", orDefault(cfg.Proxy, "(none)"))
		fmt.Printf("  OutputDir: %s\n", cfg.OutputDir)
		fmt.Printf("  Format:    %s\n", cfg.Format)
		fmt.Printf("  Quality:   %s\n", cfg.Quality)
		fmt.Printf("  Config:    %s\n", config.SavePath())

		if len(cfg.WebDAVServers) > 0 {
			fmt.Println("\nWebDAV servers:")
			for name, server := range cfg.WebDAVServers {
				fmt.Printf("  %s: %s\n", name, server.URL)
			}
		}
	},
}

// vget config path - show config file path
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.SavePath())
	},
}

// --- WebDAV remote management ---

var configWebdavCmd = &cobra.Command{
	Use:     "webdav",
	Short:   "Manage WebDAV remotes",
	Aliases: []string{"remote"},
}

var configWebdavListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured WebDAV servers",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadOrDefault()
		if len(cfg.WebDAVServers) == 0 {
			fmt.Println("No WebDAV servers configured.")
			fmt.Println("Add one with: vget config webdav add <name>")
			return
		}

		fmt.Println("WebDAV servers:")
		for name, server := range cfg.WebDAVServers {
			if server.Username != "" {
				fmt.Printf("  %s: %s (user: %s)\n", name, server.URL, server.Username)
			} else {
				fmt.Printf("  %s: %s\n", name, server.URL)
			}
		}
	},
}

var configWebdavAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new WebDAV server",
	Long: `Add a new WebDAV server configuration.

Examples:
  vget config webdav add pikpak
  vget config webdav add nextcloud

After adding, download files like:
  vget pikpak:/Movies/video.mp4`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg := config.LoadOrDefault()

		if cfg.GetWebDAVServer(name) != nil {
			fmt.Fprintf(os.Stderr, "WebDAV server '%s' already exists.\n", name)
			fmt.Fprintf(os.Stderr, "Delete it first: vget config webdav delete %s\n", name)
			os.Exit(1)
		}

		reader := bufio.NewReader(os.Stdin)

		// Get URL
		fmt.Print("WebDAV URL: ")
		urlStr, _ := reader.ReadString('\n')
		urlStr = strings.TrimSpace(urlStr)
		if urlStr == "" {
			fmt.Fprintln(os.Stderr, "URL is required")
			os.Exit(1)
		}

		// Get username
		fmt.Print("Username (enter to skip): ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		// Get password
		var password string
		if username != "" {
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read password: %v\n", err)
				os.Exit(1)
			}
			password = string(passwordBytes)
		}

		cfg.SetWebDAVServer(name, config.WebDAVServer{
			URL:      urlStr,
			Username: username,
			Password: password,
		})

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nWebDAV server '%s' added.\n", name)
		fmt.Printf("Usage: vget %s:/path/to/file.mp4\n", name)
	},
}

var configWebdavDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete a WebDAV server",
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg := config.LoadOrDefault()

		if cfg.GetWebDAVServer(name) == nil {
			fmt.Fprintf(os.Stderr, "WebDAV server '%s' not found.\n", name)
			os.Exit(1)
		}

		cfg.DeleteWebDAVServer(name)

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("WebDAV server '%s' deleted.\n", name)
	},
}

var configWebdavShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a WebDAV server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg := config.LoadOrDefault()

		server := cfg.GetWebDAVServer(name)
		if server == nil {
			fmt.Fprintf(os.Stderr, "WebDAV server '%s' not found.\n", name)
			os.Exit(1)
		}

		fmt.Printf("Name:     %s\n", name)
		fmt.Printf("URL:      %s\n", server.URL)
		if server.Username != "" {
			fmt.Printf("Username: %s\n", server.Username)
			fmt.Printf("Password: %s\n", strings.Repeat("*", len(server.Password)))
		}
	},
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func init() {
	// config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)

	// config webdav subcommands
	configWebdavCmd.AddCommand(configWebdavListCmd)
	configWebdavCmd.AddCommand(configWebdavAddCmd)
	configWebdavCmd.AddCommand(configWebdavDeleteCmd)
	configWebdavCmd.AddCommand(configWebdavShowCmd)
	configCmd.AddCommand(configWebdavCmd)

	rootCmd.AddCommand(configCmd)
}
