# Making changes to gemini-cli

When releasing a new version:

1. Bump the version in `interna/version/version.go` and commit
2. Create a new tag and push with `--tags`
3. In a separate terminal, run `go install` with the latest version specified
   explicitly to tell the mod-proxy about it
