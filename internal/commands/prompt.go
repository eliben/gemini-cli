package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var promptCmd = &cobra.Command{
	// This is the usage for the plain root command without subcommands.
	Use:   "prompt <prompt>",
	Short: "(default command) Send the prompt to a Gemini model",
	Long: `This tool lets you interact with Google's Gemini LLMs from the
command-line.`,
	Run: runPromptCmd,
}

func init() {
	rootCmd.AddCommand(promptCmd)

	chatCmd.Flags().StringP("system", "s", "", "set a system prompt")
}

// TODO: support image input with URL and file
func runPromptCmd(cmd *cobra.Command, args []string) {
	// TODO: move getAPIKey to its own package
	key := getAPIKey(cmd)

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

// getAPIToken obtains the API token from a flag or a default env var, and
// returns it. It fails with log.Fatal if neither method produces a non-empty
// key.
func getAPIKey(cmd *cobra.Command) string {
	token, _ := cmd.Flags().GetString("key")
	if len(token) > 0 {
		return token
	}

	key := os.Getenv("API_KEY")
	if len(key) > 0 {
		return key
	}

	log.Fatal("Unable to obtain API key for Google AI; use --key or API_KEY env var")
	return ""
}
