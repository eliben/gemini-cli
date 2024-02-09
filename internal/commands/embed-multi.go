package commands

import (
	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/spf13/cobra"
)

// TODO: does this really need to be a separate command, since embed itself
// does so little...
var embedMultiCmd = &cobra.Command{
	Use:   "embed-multi",
	Short: "Embed multiple inputs using an embedding model",
	Long:  `Use a Gemini embedding model to embed multiple inputs of content`,
	Run:   runEmbedMultiCmd,
}

func init() {
	rootCmd.AddCommand(embedMultiCmd)

	embedMultiCmd.Flags().StringP("model", "m", "embedding-001", "name of embedding model to use")
}

func runEmbedMultiCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	_ = key
}
