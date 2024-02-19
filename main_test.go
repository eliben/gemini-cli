package main_test

import (
	"os"
	"path/filepath"
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
			// Make all the files from test/datafiles available for tests in
			// their datafiles/ directory.
			rootdir, err := os.Getwd()
			check(t, err)
			copyDataFiles(t,
				filepath.Join(rootdir, "test", "datafiles"),
				filepath.Join(env.WorkDir, "datafiles"))

			// Propagate the test env's GEMINI_API_KEY to scripts.
			env.Setenv("GEMINI_API_KEY", os.Getenv("GEMINI_API_KEY"))

			// This is to help testing some error scenarios.
			env.Setenv("TEST_API_KEY", os.Getenv("GEMINI_API_KEY"))
			return nil
		},
	})
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// copyDataFiles copies all files from rootdir to targetdir, creating
// targetdir if needed
func copyDataFiles(t *testing.T, rootdir string, targetdir string) {
	check(t, os.MkdirAll(targetdir, 0777))

	entries, err := os.ReadDir(rootdir)
	check(t, err)
	for _, e := range entries {
		if !e.IsDir() {
			fullpath := filepath.Join(rootdir, e.Name())
			targetpath := filepath.Join(targetdir, e.Name())

			data, err := os.ReadFile(fullpath)
			check(t, err)
			err = os.WriteFile(targetpath, data, 0666)
			check(t, err)
		}
	}
}
