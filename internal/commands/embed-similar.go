package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/chewxy/math32"
	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var embedSimilarCmd = &cobra.Command{
	Use:   "similar <DB path> <content or '-'>",
	Short: "Find items in the DB similar to the given content",
	Long:  strings.TrimSpace(embedSimilarUsage),
	Args:  cobra.ExactArgs(2),
	Run:   runEmbedSimilarCmd,
}

var embedSimilarUsage = `
Use vector embeddings to calculate similarity.

The given content (argument or from standard input if '-' is passed) is embedded
and compared to the items stored in the DB's 'embeddings' table. The most
similar items are reported. This command expects the rows in the DB to have
at least 'id' and 'embeddings' columns. By default, the 'id' of similar items
is reported along with a similarity score; this can be controlled with the
'--show' flag.

The items are reported in JSONLines format (each entry is encoded as a JSON
object and printed on a separate line).
`

func init() {
	embedCmd.AddCommand(embedSimilarCmd)
	embedSimilarCmd.Flags().Int("topk", 5, "top K: how many most similar entries to return")
	embedSimilarCmd.Flags().StringSlice("show", []string{"id", "score"}, "the columns to emit for the most similar DB entries")
}

func runEmbedSimilarCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	dbPath := args[0]

	// Read content from argument or stdin
	content := args[1]
	if content == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			log.Fatal("error reading content from stdin:", err)
		}
		content = string(b)
	}

	// Calculate the content's embedding vector
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

	var contentEmb []float32
	if emb := res.Embedding; emb != nil {
		contentEmb = emb.Values
	} else {
		log.Fatal("got no embedding back from model")
	}

	// Open the DB and read items and their embeddings from the 'embeddings'
	// table. For each item, calculate its cosine similarity to the content's
	// embedding.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("unable to open DB at %v", dbPath)
	}
	defer db.Close()

	query := `SELECT * FROM embeddings`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("error running SQL query:", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		panic(err)
	}

	// After scanning, dbEntries will list all DB rows with their data in cols
	// and their similarity score stored.
	type Entry struct {
		cols  map[string]any
		score float32
	}
	var dbEntries []Entry

	for rows.Next() {
		columns := scanRowIntoSlice(rows)

		entryCols := make(map[string]any)
		for i, col := range columnNames {
			entryCols[col] = columns[i]
		}

		entryEmb := decodeEmbedding(entryCols["embedding"].([]byte))
		score := cosineSimilarity(entryEmb, contentEmb)

		dbEntries = append(dbEntries, Entry{cols: entryCols, score: score})
	}

	// Sort by descending similarity score.
	slices.SortFunc(dbEntries, func(a, b Entry) int {
		// The similarity scores are in the range [0, 1], so scale them to get
		// integers for comparison. Negate the result to get descending similarity.
		return -int(100.0 * (a.score - b.score))
	})

	showList := mustGetStringSliceFlag(cmd, "show")
	for i := 0; i < mustGetIntFlag(cmd, "topk"); i++ {
		display := make(map[string]string)
		for _, col := range showList {
			if col == "score" {
				display["score"] = fmt.Sprintf("%v", dbEntries[i].score)
			} else {
				entry, ok := dbEntries[i].cols[col]
				if !ok {
					log.Fatalf("no column '%v' to show", col)
				}
				display[col] = fmt.Sprintf("%v", entry)
			}
		}

		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(display); err != nil {
			log.Fatal(err)
		}
	}
}

// cosineSimilarity calculates cosine similarity (magnitude-adjusted dot
// product) between two vectors that must be of the same size.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("different lengths")
	}

	var aMag, bMag, dotProduct float32
	for i := 0; i < len(a); i++ {
		aMag += a[i] * a[i]
		bMag += b[i] * b[i]
		dotProduct += a[i] * b[i]
	}
	return dotProduct / (math32.Sqrt(aMag) * math32.Sqrt(bMag))
}
