package commands

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

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
	embedCmd.Flags().StringP("format", "", "float", "format for embedding output: float, base64, blob")
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
		format, _ := cmd.Flags().GetString("format")
		emitEmbedding(os.Stdout, emb.Values, format)
	} else {
		log.Fatal("got no embeddinb back from model")
	}
}

func emitEmbedding(w io.Writer, v []float32, format string) {
	switch format {
	case "float":
		fmt.Fprintln(w, v)
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

// encodeEmbedding encodes an embedding into a byte buffer, e.g. for DB
// storage as a blob.
func encodeEmbedding(emb []float32) []byte {
	buf := new(bytes.Buffer)
	for _, f := range emb {
		err := binary.Write(buf, binary.LittleEndian, f)
		if err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}

// decodeEmbedding decodes an embedding back from a byte buffer.
func decodeEmbedding(b []byte) []float32 {
	buf := bytes.NewReader(b)

	// Calculate how many float32 values are in the slice
	count := buf.Len() / 4
	numbers := make([]float32, 0, count)

	for i := 0; i < count; i++ {
		var num float32
		err := binary.Read(buf, binary.LittleEndian, &num)
		if err != nil {
			panic(err)
		}
		numbers = append(numbers, num)
	}
	return numbers
}
