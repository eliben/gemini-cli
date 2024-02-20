package apikey

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

// Get obtains the API key from a flag or a default env var, and
// returns it. It fails with log.Fatal if neither method produces a non-empty
// key.
func Get(cmd *cobra.Command) string {
	token, _ := cmd.Flags().GetString("key")
	if len(token) > 0 {
		return token
	}

	key := os.Getenv("GEMINI_API_KEY")
	if len(key) > 0 {
		return key
	}

	log.Fatal("Unable to obtain API key for Google AI; use --key or GEMINI_API_KEY env var")
	return ""
}
