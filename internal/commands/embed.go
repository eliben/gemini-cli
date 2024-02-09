package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Embed content using an embedding model",
	Long:  `Use a Gemini embedding model to embed content in various forms`,
	Run:   runEmbedCmd,
}

func init() {
	rootCmd.AddCommand(embedCmd)

	// TODO: options:
	// -c for content
	// -m for specifying model
	// --format for output format (base64 etc)

	embedCmd.Flags().StringP("content", "c", "", "content to embed")
	embedCmd.Flags().StringP("model", "m", "embedding-001", "name of embedding model to use")
}

func runEmbedCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal()
	}

	modelName, _ := cmd.Flags().GetString("model")
	model := client.EmbeddingModel(modelName)

	content, _ := cmd.Flags().GetString("content")
	res, err := model.EmbedContent(ctx, genai.Text(content))

	if emb := res.Embedding; emb != nil {
		fmt.Println(emb.Values)
	} else {
		log.Fatal("got no embeddinb back from model")
	}
}
