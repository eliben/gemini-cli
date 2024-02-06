package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// TODO: usage here
	Use:   "gemini-cli",
	Short: "Interact with GoogleAI's Gemini LLMs through the command line",
	Long: `This tool lets you interact with Google's Gemini LLMs from the
command-line.`,
	Run: runRootCmd,
}

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once to
// the rootCmd.
func Execute() int {
	err := rootCmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().String("key", "", "API key for Google AI")
	rootCmd.PersistentFlags().String("model", "gemini-pro", "model to use")
}

func runRootCmd(cmd *cobra.Command, args []string) {
	key := getAPIKey(cmd)

	var prompt string
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		prompt = string(b)
	}

	if len(args) >= 1 {
		prompt = prompt + " " + args[0]
	}
	if len(strings.TrimSpace(prompt)) == 0 {
		log.Fatal("empty prompt")
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
	iter := model.GenerateContentStream(ctx, genai.Text(prompt))
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
