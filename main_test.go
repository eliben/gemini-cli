package main_test

import (
	"os"
	"testing"

	"github.com/eliben/gemini-cli/internal/commands"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gemini-cli": commands.Execute,
	}))
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:      "test/scripts",
		TestWork: true,
		Setup: func(env *testscript.Env) error {
			// Propagate the test env's API_KEY to scripts.
			env.Setenv("API_KEY", os.Getenv("API_KEY"))

			// This is to help testing some error scenarios.
			env.Setenv("AUX_API_KEY", os.Getenv("API_KEY"))
			return nil
		},
	})
}
