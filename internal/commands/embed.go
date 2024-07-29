package commands

import (
	_ "modernc.org/sqlite"

	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Access Gemini embedding models",
	Long:  `Use sub-commands of this command`,
	Args:  cobra.MaximumNArgs(1),

	// 'embed' is a parent of subcommands, and doesn't do anything on its own.
	// Therefore we don't define a Run: function for it.
}

func init() {
	rootCmd.AddCommand(embedCmd)
	embedCmd.PersistentFlags().StringP("model", "m", "text-embedding-004", "name of embedding model to use")
}
