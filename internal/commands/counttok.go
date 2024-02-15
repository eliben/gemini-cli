package commands

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var countTokCmd = &cobra.Command{
	Use:     "counttok <content or '-'>",
	Aliases: []string{"tokcount"},
	Short:   "Count tokens in content",
	Long: `Count the number of LLM tokens in the given content.

The content is passed as a string on the command-line (quote it if spaces are
included), or read from standard input if '-' is provided.
`,
	Run: runCountTokCmd,
}

func init() {
	rootCmd.AddCommand(countTokCmd)
}

func runCountTokCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	content := args[0]

	if content == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			log.Fatal("error reading content from stdin:", err)
		}
		content = string(b)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
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
