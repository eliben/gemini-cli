package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
)

var countTokCmd = &cobra.Command{
	Use:     "counttok <content or '-'>",
	Aliases: []string{"tokcount"},
	Short:   "Count tokens in content",
	Args:    cobra.ExactArgs(1),
	Long:    strings.TrimSpace(countTokUsage),
	Run:     runCountTokCmd,
}

var countTokUsage = `
Count the number of LLM tokens in the given content.

The content is passed as a string on the command-line (quote it if spaces are
included), or read from standard input if '-' is provided.
`

func init() {
	rootCmd.AddCommand(countTokCmd)
}

func runCountTokCmd(cmd *cobra.Command, args []string) {
	content := args[0]

	if content == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			log.Fatal("error reading content from stdin:", err)
		}
		content = string(b)
	}

	ctx := context.Background()
	client, err := newGenaiClient(ctx, cmd)
	if err != nil {
		log.Fatal()
	}

	model := client.GenerativeModel(mustGetStringFlag(cmd, "model"))
	resp, err := model.CountTokens(ctx, genai.Text(content))
	if err != nil {
		log.Fatal("error counting tokens:", err)
	}
	fmt.Println(resp.TotalTokens)
}
