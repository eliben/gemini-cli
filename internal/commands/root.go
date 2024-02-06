package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
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
	Args: cobra.ExactArgs(1),
	Run:  runRootCmd,
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
}

func runRootCmd(cmd *cobra.Command, args []string) {
	key := getAPIKey(cmd)

	if len(args) < 1 {
		log.Fatal("expect at least one argument: <prompt>")
	}
	prompt := args[0]
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// TODO: this is the default model, but make it configurable
	model := client.GenerativeModel("gemini-pro")
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
