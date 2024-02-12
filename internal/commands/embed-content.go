package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var embedContentCmd = &cobra.Command{
	Use:   "content",
	Short: "Embed a single input using an embedding model",
	Long:  `Use a Gemini embedding model to embed a single string of content`,
	Args:  cobra.ExactArgs(1),
	Run:   runEmbedContentCmd,
}

// TODO: add reading from '-' here for stdin
func init() {
	embedCmd.AddCommand(embedContentCmd)
	embedContentCmd.Flags().String("format", "json", "format for embedding output: json, base64, blob")
}

func runEmbedContentCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	content := args[0]

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal()
	}

	model := client.EmbeddingModel(mustGetStringFlag(cmd, "model"))
	res, err := model.EmbedContent(ctx, genai.Text(content))
	if err != nil {
		log.Fatal("error embedding content:", err)
	}

	if emb := res.Embedding; emb != nil {
		emitEmbedding(os.Stdout, emb.Values, mustGetStringFlag(cmd, "format"))
	} else {
		log.Fatal("got no embedding back from model")
	}
}

func emitEmbedding(w io.Writer, v []float32, format string) {
	switch format {
	case "json":
		encoder := json.NewEncoder(w)
		err := encoder.Encode(v)
		if err != nil {
			log.Fatal(err)
		}
	case "base64":
		b := encodeEmbedding(v)
		encoder := base64.NewEncoder(base64.StdEncoding, w)
		encoder.Write(b)
		encoder.Close()
		fmt.Println()
	case "blob":
		b := encodeEmbedding(v)
		w.Write(b)
	default:
		log.Fatalf("invalid format: %s", format)
	}
}
