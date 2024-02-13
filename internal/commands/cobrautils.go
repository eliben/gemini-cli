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

// mustGetIntFlag gets an int flag value from cmd, and panics if this
// results in an error.
func mustGetIntFlag(cmd *cobra.Command, name string) int {
	v, err := cmd.Flags().GetInt(name)
	if err != nil {
		panic(err)
	}
	return v
}

// mustGetStringSliceFlag gets an string slice flag value from cmd, and panics
// if this results in an error.
func mustGetStringSliceFlag(cmd *cobra.Command, name string) []string {
	v, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		panic(err)
	}
	return v
}
