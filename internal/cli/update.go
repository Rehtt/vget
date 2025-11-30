package cli

import (
	"github.com/guiyumin/vget/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update vget to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updater.Update()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
