package commands

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
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
	ctx := context.Background()
	client, err := newGenaiClient(ctx, cmd)
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
	fmt.Println("Type 'exit' or 'quit' to exit, or '$load <file path>' to load a file")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "exit" || text == "quit" {
			break
		}

		var inputPart genai.Part
		// Detect a special chat command.
		if path, found := strings.CutPrefix(text, "$load"); found {
			part, err := getPartFromFile(strings.TrimSpace(path))
			if err != nil {
				log.Fatalf("error loading file %s: %v", path, err)
			}
			inputPart = part
		} else {
			inputPart = genai.Text(text)
		}

		iter := session.SendMessageStream(ctx, inputPart)

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
				}
			}
		}
	}
}
