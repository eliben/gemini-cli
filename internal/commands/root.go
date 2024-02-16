package commands

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gemini-cli <command>",
	Short: "Interact with GoogleAI's Gemini LLMs through the command line",
	Long: `This tool lets you interact with Google's Gemini LLMs from the
command-line.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once to
// the rootCmd.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().String("key", "", "API key for Google AI")
	rootCmd.PersistentFlags().String("model", "gemini-pro", "Name of model to use; see https://ai.google.dev/models/gemini")
}
