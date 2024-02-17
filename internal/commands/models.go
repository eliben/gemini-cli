package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List supported Gemini models",
	Args:  cobra.ExactArgs(0),
	Long:  strings.TrimSpace(modelsUsage),
	Run:   runModelsCmd,
}

var modelsUsage = `
List the Gemini models supported by Google AI, along with some details
about each model.

Many of these may start with a 'models/' prefix; when you pass a model name to
the '--model' flag of other commands, you may omit the prefix for brevity.

Explanation of some non-obvious columns in the output:

* Max In: the maximal number of input tokens supported by the model
* Max Out: the maximal number of output tokens supported by the model
`

func init() {
	rootCmd.AddCommand(modelsCmd)

	modelsCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		command.Flags().MarkHidden("model")
		command.Parent().HelpFunc()(command, strings)
	})
}

func runModelsCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal()
	}

	w := tabwriter.NewWriter(os.Stdout, 6, 16, 1, '\t', 0)
	fmt.Fprintf(w, "%-32s\tVersion\tMax In\tMax Out\tDescription\n", "Name")
	fmt.Fprintf(w, "\n")

	iter := client.ListModels(ctx)
	for {
		mi, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(w, "%-32s\t%s\t%v\t%v\t%s\n", mi.Name, mi.Version, mi.InputTokenLimit, mi.OutputTokenLimit, mi.Description)
		w.Flush()

		//fmt.Println(mi.Name, mi.Version, mi.Description, mi.InputTokenLimit, mi.OutputTokenLimit)
	}
}
