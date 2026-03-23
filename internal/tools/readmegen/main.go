package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	caphcmd "github.com/syself/caphcli/internal/cmd"
)

const readmeTemplate = `# caphcli

` + "`caphcli`" + ` is a standalone CLI for CAPH bare-metal provisioning checks.

It currently exposes the ` + "`check-bm-servers`" + ` command that was extracted from CAPH PR ` + "`#1873`" + ` and moved into this repository.

## Build

` + "```bash" + `
go build .
` + "```" + `

## Required Environment Variables

- ` + "`HETZNER_ROBOT_USER`" + ` and ` + "`HETZNER_ROBOT_PASSWORD`" + ` for Hetzner Robot API access.
- ` + "`SSH_KEY_NAME`" + ` for the Robot SSH key name to use or create.
- One of ` + "`HETZNER_SSH_PUB_PATH`" + ` or ` + "`HETZNER_SSH_PUB`" + ` for the SSH public key.
- One of ` + "`HETZNER_SSH_PRIV_PATH`" + ` or ` + "`HETZNER_SSH_PRIV`" + ` for the SSH private key.

## Keeping This README Up To Date

The official Cobra helper for generated command docs is ` + "`github.com/spf13/cobra/doc`" + `.  
This repo keeps the help blocks below in sync with the actual command tree via:

` + "```bash" + `
go generate ./...
` + "```" + `

That runs ` + "`go run ./internal/tools/readmegen`" + `, which rebuilds this README from the live Cobra commands.

## CLI Help

### ` + "`caphcli --help`" + `

` + "```text" + `
{{ROOT_HELP}}
` + "```" + `

### ` + "`caphcli check-bm-servers --help`" + `

` + "```text" + `
{{CHECK_HELP}}
` + "```" + `
`

func main() {
	rootHelp, err := renderHelp()
	if err != nil {
		fail(err)
	}

	checkHelp, err := renderHelp("check-bm-servers")
	if err != nil {
		fail(err)
	}

	readme := strings.ReplaceAll(readmeTemplate, "{{ROOT_HELP}}", strings.TrimSpace(rootHelp))
	readme = strings.ReplaceAll(readme, "{{CHECK_HELP}}", strings.TrimSpace(checkHelp))

	if err := os.WriteFile("README.md", []byte(readme), 0o644); err != nil {
		fail(fmt.Errorf("write README.md: %w", err))
	}
}

func renderHelp(args ...string) (string, error) {
	rootCmd := caphcmd.NewRootCommand()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append(args, "--help"))

	if err := rootCmd.Execute(); err != nil {
		return "", fmt.Errorf("render help for %q: %w", strings.Join(args, " "), err)
	}

	return buf.String(), nil
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "readmegen: %v\n", err)
	os.Exit(1)
}
