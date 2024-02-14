package commands

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/eliben/gemini-cli/internal/tableloader"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var embedDBCmd = &cobra.Command{
	Use:   "db <output DB path> [input file or '-']",
	Short: "Embed a multiple inputs, storing results into a SQLite DB",
	Long:  strings.TrimSpace(embedDBUsage),
	Args:  cobra.RangeArgs(1, 2),
	Run:   runEmbedDBCmd,
}

var embedDBUsage = `
Embed multiple texts and store the results into a SQLite DB. The path to the
output DB is given as the first argument. This command has multiple modes of
operation based on flags.

* With --sql, provide a SQL query to use on the DB itself. The query should
  specify at least 2 columns; the first is used as the ID for the resulting
  embedding; the rest are concatenated into a single text and the embedding is
  computed on this text. The --attach flag can provide an alternative DB file so
  the SQL query can read from it.
* With --files or --files-list, the inputs are taken from the filesystem, each
  file becoming the contents to be embedded.
* Otherwise, the input is read from a file provided as an argument (or '-',
  which reads from standard input). The format of the file should be either CSV,
  TSV (tab-separated), JSON or JSONLines (one line per JSON object). At least 2
  columns are expected: one for ID, and the rest are concatenated as inputs to
  the embedding model.
`

func init() {
	embedCmd.AddCommand(embedDBCmd)
	embedDBCmd.Flags().String("table", "embeddings", "DB table name to store embeddings into")
	embedDBCmd.Flags().Int("batch-size", 32, "size of batches (number of rows) to send for embedding")

	embedDBCmd.Flags().String("sql", "", "SQL mode with a query")
	embedDBCmd.Flags().StringSlice("attach", nil, "additional DB to attach - specify <alias>,<filename> pair")

	embedDBCmd.Flags().StringSlice("files", nil, strings.TrimSpace(`
files to embed as a <root dir>,<glob> pair;
the directory will be traversed recursively,
picking all the files that match the glob`))
	embedDBCmd.Flags().StringSlice("files-list", nil, `comma-separated list of files to embed`)

	embedDBCmd.Flags().Bool("store", false, `also store the original content in the embeddings table ('content' column)`)
	embedDBCmd.Flags().String("metadata", "", `also store this metadata in the embeddings table ('metadata' column)`)
	embedDBCmd.Flags().String("prefix", "", `prepend a prefix to the stored ID of each row`)
}

func runEmbedDBCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)
	dbPath := args[0]

	sqlMode := mustGetStringFlag(cmd, "sql")
	filesMode := len(mustGetStringSliceFlag(cmd, "files")) > 0 ||
		len(mustGetStringSliceFlag(cmd, "files-list")) > 0

	if sqlMode != "" && filesMode {
		log.Fatal("--files* mode is mutually exclusive with --sql")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("unable to open DB at %v", dbPath)
	}
	defer db.Close()

	tableName := mustGetStringFlag(cmd, "table")

	// Build up table schema based on passed flags
	columns := []string{
		"id TEXT PRIMARY KEY",
		"embedding BLOB",
	}

	if mustGetBoolFlag(cmd, "store") {
		columns = append(columns, "content TEXT")
	}
	if mustGetStringFlag(cmd, "metadata") != "" {
		columns = append(columns, "metadata TEXT")
	}

	tableCreateSchema := strings.TrimSpace(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
%s
)`, tableName, strings.Join(columns, ",\n")))

	_, err = db.Exec(tableCreateSchema)
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
		attachPair := mustGetStringSliceFlag(cmd, "attach")
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
	} else if filesMode {
		ids, texts = collectFiles(cmd)
	} else {
		if len(args) < 2 {
			log.Fatal("when --sql or --files* is not passed, expect filename or '-' as second argument")
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

	numColumns := 2
	if mustGetBoolFlag(cmd, "store") {
		numColumns++
	}
	if mustGetStringFlag(cmd, "metadata") != "" {
		numColumns++
	}

	query := fmt.Sprintf("INSERT INTO %s VALUES (%s)",
		tableName, strings.Join(strings.Split(strings.Repeat("?", numColumns), ""), ", "))

	for i, emb := range embs {
		id := ids[i]
		if prefix := mustGetStringFlag(cmd, "prefix"); prefix != "" {
			id = prefix + id
		}

		columns := []any{id, encodeEmbedding(emb)}
		if mustGetBoolFlag(cmd, "store") {
			columns = append(columns, texts[i])
		}
		if metadata := mustGetStringFlag(cmd, "metadata"); metadata != "" {
			columns = append(columns, metadata)
		}
		_, err = db.Exec(query, columns...)
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

// collectFiles reads files provided with the --files or --files-list flags
// and generates a list of ids (file paths) and a corresponding list of texts
// (file contents).
func collectFiles(cmd *cobra.Command) ([]string, []string) {
	filesList := mustGetStringSliceFlag(cmd, "files-list")
	filesDirGlobPair := mustGetStringSliceFlag(cmd, "files")

	var ids []string
	var texts []string
	if len(filesList) > 0 {
		if len(filesDirGlobPair) > 0 {
			log.Fatal("expect only one of --files & --files-list")
		}

		for _, path := range filesList {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}
			ids = append(ids, path)
			texts = append(texts, string(b))
		}
	} else if len(filesDirGlobPair) > 0 {
		if len(filesDirGlobPair) != 2 {
			log.Fatal("expect <root dir>,<glob> pair for --files")
		}
		rootDir := filesDirGlobPair[0]
		glob := filesDirGlobPair[1]

		fileInfo, err := os.Stat(rootDir)
		if err != nil {
			log.Fatal(err)
		}
		if !fileInfo.IsDir() {
			log.Fatalf("expect directory as the first item provided to --files, got %v", rootDir)
		}

		visit := func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				matched, err := filepath.Match(glob, d.Name())
				if err != nil {
					return err
				}
				if matched {
					b, err := os.ReadFile(path)
					if err != nil {
						log.Fatal(err)
					}
					ids = append(ids, path)
					texts = append(texts, string(b))
				}
			}
			return nil
		}

		err = filepath.WalkDir(rootDir, visit)
		if err != nil {
			log.Fatalf("error visiting %v: %v", rootDir, err)
		}
	} else {
		panic("expect --files or --files-list")
	}
	return ids, texts
}
