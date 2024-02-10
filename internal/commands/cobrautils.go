package commands

import "github.com/spf13/cobra"

// mustGetStringFlag gets a string flag value from cmd, and panics if this
// results in an error (for example, if such a flag wasn't defined for the
// command or if its type isn't string).
func mustGetStringFlag(cmd *cobra.Command, name string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		panic(err)
	}
	return v
}
