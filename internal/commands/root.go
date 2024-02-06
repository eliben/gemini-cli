package commands

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// This is the usage for the plain root command without subcommands.
	Use:   "gemini-cli <prompt>",
	Short: "Interact with GoogleAI's Gemini LLMs through the command line",
	Long: `This tool lets you interact with Google's Gemini LLMs from the
command-line.`,
	Args: cobra.ArbitraryArgs,
	Run:  runRootCmd,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

const defaultCmd = "prompt"

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once to
// the rootCmd.
func Execute() int {
	err := rootCmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().String("key", "", "API key for Google AI")
	rootCmd.PersistentFlags().String("model", "gemini-pro", "model to use")
}

func runRootCmd(cmd *cobra.Command, args []string) {
	promptCmd.Run(cmd, args)
}
