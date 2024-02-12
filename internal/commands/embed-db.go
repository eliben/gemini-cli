package commands

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/eliben/gemini-cli/internal/tableloader"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var embedDBCmd = &cobra.Command{
	// TODO: fix usage
	Use:   "db",
	Short: "Embed a multiple inputs, storing results into a SQLite DB",
	Long:  `TODO`,
	Args:  cobra.RangeArgs(1, 2),
	Run:   runEmbedDBCmd,
}

func init() {
	embedCmd.AddCommand(embedDBCmd)
	embedDBCmd.Flags().String("table", "embeddings", "DB table name to store embeddings into")
	embedDBCmd.Flags().String("sql", "", "SQL mode with a query")
	embedDBCmd.Flags().Int("batch-size", 32, "size of batches (number of rows) to send for embedding")
	embedDBCmd.Flags().StringSlice("attach", nil, "additional DB to attach - specify <alias>,<filename> pair")
}

func runEmbedDBCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	dbPath := args[0]

	sqlMode := mustGetStringFlag(cmd, "sql")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("unable to open DB at %v", dbPath)
	}
	defer db.Close()

	tableName := mustGetStringFlag(cmd, "table")
	_, err = db.Exec(fmt.Sprintf(`
  CREATE TABLE IF NOT EXISTS %s (
	id TEXT PRIMARY KEY,
	embedding BLOB
	)`, tableName))
	if err != nil {
		log.Fatal("unable to create table in DB: ", tableName)
	}

	// We extract a list of [id, text] pairs - either from the DB itself (in --sql
	// mode) or from an input file. These texts are going to be sent to the model
	// for calculating embeddings. Each text is the concatenation of all the text
	// columns following ID that the SQL query specifies.
	var ids []string
	var texts []string

	if sqlMode != "" {
		attachPair, _ := cmd.Flags().GetStringSlice("attach")
		if len(attachPair) > 0 {
			if len(attachPair) != 2 {
				log.Fatal("expect <alias>,<db path> pair for --attach")
			}

			alias := attachPair[0]
			path := attachPair[1]

			attachStmt := fmt.Sprintf("ATTACH DATABASE '%v' as %v", path, alias)
			_, err := db.Exec(attachStmt)
			if err != nil {
				log.Fatalf("unable to attach %v: %v", path, err)
			}
		}

		rows, err := db.Query(sqlMode)
		if err != nil {
			log.Fatal("error running SQL query:", err)
		}
		defer rows.Close()

		for rows.Next() {
			// Scan all len(colNames) columns into the values slice.
			values := scanRowIntoSlice(rows)
			if len(values) < 2 {
				log.Fatalf("expect at least 2 columns from query; got %v", len(values))
			}

			var rowTexts []string
			for _, v := range values[1:] {
				rowTexts = append(rowTexts, fmt.Sprintf("%v", v))
			}
			ids = append(ids, fmt.Sprintf("%v", values[0]))
			texts = append(texts, strings.Join(rowTexts, " "))
		}

		// Check for errors from iterating over rows.
		if err := rows.Err(); err != nil {
			log.Fatal("error scanning DB:", err)
		}
	} else {
		if len(args) < 2 {
			log.Fatal("when --sql is not passed, expect filename or '-' as second argument")
		}
		inputFilename := args[1]

		var inputReader io.Reader
		if inputFilename == "-" {
			inputReader = cmd.InOrStdin()
		} else {
			file, err := os.Open(inputFilename)
			if err != nil {
				log.Fatal("unable to open %v: %v", inputFilename, err)
			}
			inputReader = file
		}

		_, table, err := tableloader.LoadTable(inputReader, tableloader.FormatUnknown)
		if err != nil {
			log.Fatal(err)
		}

		for _, row := range table {
			// It's mandatory to have an 'id'; the other columns will be concatenated
			// into texts.
			id, ok := row["id"]
			if !ok {
				log.Fatalf("expect input row to have 'id' column; got %v", row)
			}

			var rowTexts []string
			for k, v := range row {
				if k != "id" {
					rowTexts = append(rowTexts, v)
				}
			}

			ids = append(ids, id)
			texts = append(texts, strings.Join(rowTexts, " "))
		}
	}
	log.Printf("Found %d values to embed", len(texts))

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	em := client.EmbeddingModel(mustGetStringFlag(cmd, "model"))

	batchSize := mustGetIntFlag(cmd, "batch-size")
	numBatches := len(texts) / batchSize
	if len(texts)%batchSize != 0 {
		numBatches++
	}
	log.Printf("Splitting to %d batches", numBatches)

	// Build a batch and send it for embedding. cursor points to the current
	// text being added to a batch.
	cursor := 0
	embs := make([][]float32, 0, len(texts))
	for bn := 0; bn < numBatches; bn++ {
		batch := em.NewBatch()

		sizeOfThisBatch := batchSize
		if cursor+batchSize >= len(texts) {
			sizeOfThisBatch = len(texts) - cursor
		}
		log.Printf("Embedding batch #%d / %d, size=%d", bn+1, numBatches, sizeOfThisBatch)

		for i := 0; i < sizeOfThisBatch; i++ {
			batch.AddContent(genai.Text(texts[cursor]))
			cursor++
		}

		res, err := em.BatchEmbedContents(ctx, batch)
		if err != nil {
			log.Fatalf("error embedding batch %d: %v", bn, err)
		}

		if len(res.Embeddings) != sizeOfThisBatch {
			log.Fatalf("expected %d embeddings for batch, got %d", sizeOfThisBatch, len(res.Embeddings))
		}

		for _, e := range res.Embeddings {
			embs = append(embs, e.Values)
		}
	}

	log.Printf("Collected %d embeddings; inserting into table %s", len(embs), tableName)
	query := fmt.Sprintf("INSERT INTO %s VALUES (?, ?)", tableName)

	for i, emb := range embs {
		_, err = db.Exec(query, ids[i], encodeEmbedding(emb))
		if err != nil {
			log.Fatal("unable to insert embedding into DB", err)
		}
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

// scanRowIntoSlice scans a row into a slice of any.
func scanRowIntoSlice(row *sql.Rows) []any {
	colNames, err := row.Columns()
	if err != nil {
		panic(err)
	}

	values := make([]interface{}, len(colNames))
	scanArgs := make([]interface{}, len(colNames))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	err = row.Scan(scanArgs...)
	if err != nil {
		log.Fatal("error scanning row:", err)
	}
	return values
}