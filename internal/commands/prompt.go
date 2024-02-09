package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var promptCmd = &cobra.Command{
	Use:     "prompt <prompt>",
	Aliases: []string{"p", "ask"},
	Short:   "Send a prompt to a Gemini model",
	Long: `Send a prompt to the LLM. The prompt can be provided in an argument,
through stdin, or both; in case both are provided, the prompt sent to the
LLM is a concatenation of the stdin contents, followed by the argument.`,
	Run: runPromptCmd,
}

func init() {
	rootCmd.AddCommand(promptCmd)

	promptCmd.Flags().StringP("system", "s", "", "set a system prompt")
}

// TODO: support image input with URL and file
func runPromptCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)

	// Build up parts of prompt.
	var promptParts []genai.Part

	sysPrompt, _ := cmd.Flags().GetString("system")
	if sysPrompt != "" {
		promptParts = append(promptParts, genai.Text(sysPrompt))
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		promptParts = append(promptParts, genai.Text(string(b)))
	}

	if len(args) >= 1 {
		promptParts = append(promptParts, genai.Text(args[0]))
	}

	if len(promptParts) == 0 {
		log.Fatal("expect a prompt from stdin and/or command-line argument")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	modelName, _ := cmd.Flags().GetString("model")
	model := client.GenerativeModel(modelName)
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
	}

	// TODO: no-stream flag?
	iter := model.GenerateContentStream(ctx, promptParts...)
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if len(resp.Candidates) < 1 {
			fmt.Println("<empty response from model>")
		} else {
			c := resp.Candidates[0]
			if c.Content != nil {
				for _, part := range c.Content.Parts {
					fmt.Print(part)
				}
			} else {
				fmt.Println("<empty response from model>")
			}
		}
	}
	fmt.Println()
}
