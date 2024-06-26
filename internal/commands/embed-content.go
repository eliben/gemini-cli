package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
)

var embedContentCmd = &cobra.Command{
	Use:   "content <content or '-'>",
	Short: "Embed a single input using an embedding model",
	Long:  strings.TrimSpace(embedContentUsage),
	Args:  cobra.ExactArgs(1),
	Run:   runEmbedContentCmd,
}

var embedContentUsage = `
Use a Gemini embedding model to embed a single string of content, emitting the
result to stdout. The --format flag controls the format of the emitted
embedding.

The content is passed as a string on the command-line (quote it if spaces are
included), or read from standard input if '-' is provided.
`

func init() {
	embedCmd.AddCommand(embedContentCmd)
	embedContentCmd.Flags().String("format", "json", "format for embedding output: json, base64, blob")
}

func runEmbedContentCmd(cmd *cobra.Command, args []string) {
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
