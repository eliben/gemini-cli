package main

import (
	"os"

	"github.com/eliben/gemini-cli/internal/commands"
)

func main() {
	os.Exit(commands.Execute())
}
