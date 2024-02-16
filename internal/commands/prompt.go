package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/eliben/gemini-cli/internal/apikey"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// TODO: implement loading files from URL
// improve documentation
var promptCmd = &cobra.Command{
	Use:     "prompt <prompt or '-'>...",
	Aliases: []string{"p", "ask"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Send a prompt to a Gemini model",
	Long:    strings.TrimSpace(promptUsage),
	Run:     runPromptCmd,
}

var promptUsage = `
Send a prompt to the LLM. The prompt can be provided in a sequence of arguments,
one of which can be '-' for standard input.

The prompts are sent as a sequence to the model in the order provided.
If --system is provided, it's prepended to the other prompts.

TODO: support images with filenames and URLs
If you're providing multi-modal prompts (e.g. with images), make sure to
select an appropriate model like gemini-pro-vision
(see https://ai.google.dev/models/gemini for a list of model names).
`

func init() {
	rootCmd.AddCommand(promptCmd)

	promptCmd.Flags().StringP("system", "s", "", "set a system prompt")
	promptCmd.Flags().Bool("stream", true, "stream the response from the model")
}

func runPromptCmd(cmd *cobra.Command, args []string) {
	key := apikey.Get(cmd)

	// Build up parts of prompt.
	var promptParts []genai.Part

	if sysPrompt := mustGetStringFlag(cmd, "system"); sysPrompt != "" {
		promptParts = append(promptParts, genai.Text(sysPrompt))
	}

	seenStdin := false
	for _, arg := range args {
		if arg == "-" {
			if seenStdin {
				log.Fatal("expect a single '-' in list of prompts")
			}

			b, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				log.Fatal("error reading content from stdin:", err)
			}
			promptParts = append(promptParts, genai.Text(string(b)))
			seenStdin = true
		} else if argLooksLikeURL(arg) {
			part, err := getPartFromURL(arg)
			if err != nil {
				log.Fatal(err)
			}
			promptParts = append(promptParts, part)
		} else if argLooksLikeFilename(arg) {
			// TODO: this is too permissive... it will detect all URLs too!
			// need a better heuristic
			part, err := getPartFromFile(arg)
			if err != nil {
				log.Fatal(err)
			}
			promptParts = append(promptParts, part)
		} else {
			promptParts = append(promptParts, genai.Text(arg))
		}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel(mustGetStringFlag(cmd, "model"))
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
	}

	if stream := mustGetBoolFlag(cmd, "stream"); stream {
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
	} else {
		resp, err := model.GenerateContent(ctx, promptParts...)
		if err != nil {
			log.Fatal(err)
		}
		if len(resp.Candidates) < 1 {
			fmt.Println("<empty response from model>")
		} else {
			c := resp.Candidates[0]
			if c.Content != nil {
				for _, part := range c.Content.Parts {
					fmt.Println(part)
				}
			} else {
				fmt.Println("<empty response from model>")
			}
		}
	}
}

func argLooksLikeFilename(arg string) bool {
	ext := filepath.Ext(arg)
	return ext != ""
}

func argLooksLikeURL(arg string) bool {
	_, err := url.ParseRequestURI(arg)
	if err != nil {
		return false
	}
	return true
}

func getPartFromFile(path string) (genai.Part, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	switch strings.TrimSpace(ext) {
	case ".jpg", ".jpeg":
		return genai.ImageData("jpeg", b), nil
	case ".png":
		return genai.ImageData("png", b), nil
	default:
		return nil, fmt.Errorf("invalid image file extension: %s", ext)
	}
}

func getPartFromURL(url string) (genai.Part, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image from url: %w", err)
	}
	defer resp.Body.Close()

	urlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image bytes: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mime type %v", mimeType)
	}

	return genai.ImageData(parts[1], urlData), nil
}
