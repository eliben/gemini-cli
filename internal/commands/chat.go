package commands

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive chat with a model",
	Long:  `Start an interactive terminal chat with a Gemini model.`,
	Run:   runChatCmd,
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

func runChatCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)

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

	session := model.StartChat()
	fmt.Printf("Chatting with %s\n", modelName)
	fmt.Println("Type 'exit' or 'quit' to exit")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "exit" || text == "quit" {
			break
		}

		iter := session.SendMessageStream(ctx, genai.Text(text))

	ResponseIter:
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				break ResponseIter
			}
			if err != nil {
				log.Fatal(err)
			}
			if len(resp.Candidates) >= 0 {
				c := resp.Candidates[0]
				if c.Content != nil {
					for _, part := range c.Content.Parts {
						fmt.Print(part)
					}
					fmt.Println()
				}
			}
		}
	}
}
