package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	caphcmd "github.com/syself/caphcli/internal/cmd"
)

const (
	readmePath  = "README.md"
	startMarker = "<!-- readmegen:cli-help:start -->"
	endMarker   = "<!-- readmegen:cli-help:end -->"
)

const generatedSectionTemplate = `## CLI Help

### ` + "`caphcli --help`" + `

` + "```text" + `
{{ROOT_HELP}}
` + "```" + `

### ` + "`caphcli check-bm-server --help`" + `

` + "```text" + `
{{CHECK_HELP}}
` + "```" + `

### ` + "`caphcli create-host-yaml --help`" + `

` + "```text" + `
{{CREATE_HOST_YAML_HELP}}
` + "```" + `
`

func main() {
	rootHelp, err := renderHelp()
	if err != nil {
		fail(err)
	}

	checkHelp, err := renderHelp("check-bm-server")
	if err != nil {
		fail(err)
	}

	createHostYAMLHelp, err := renderHelp("create-host-yaml")
	if err != nil {
		fail(err)
	}

	generatedSection := strings.ReplaceAll(generatedSectionTemplate, "{{ROOT_HELP}}", strings.TrimSpace(rootHelp))
	generatedSection = strings.ReplaceAll(generatedSection, "{{CHECK_HELP}}", strings.TrimSpace(checkHelp))
	generatedSection = strings.ReplaceAll(generatedSection, "{{CREATE_HOST_YAML_HELP}}", strings.TrimSpace(createHostYAMLHelp))

	readme, err := os.ReadFile(readmePath)
	if err != nil {
		fail(fmt.Errorf("read %s: %w", readmePath, err))
	}

	updatedReadme, err := replaceMarkedSection(string(readme), generatedSection)
	if err != nil {
		fail(err)
	}

	if err := os.WriteFile(readmePath, []byte(updatedReadme), 0o644); err != nil {
		fail(fmt.Errorf("write %s: %w", readmePath, err))
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

func replaceMarkedSection(readme string, generatedSection string) (string, error) {
	start := strings.Index(readme, startMarker)
	if start == -1 {
		return "", fmt.Errorf("missing start marker %q in %s", startMarker, readmePath)
	}

	searchFrom := start + len(startMarker)
	endOffset := strings.Index(readme[searchFrom:], endMarker)
	if endOffset == -1 {
		return "", fmt.Errorf("missing end marker %q in %s", endMarker, readmePath)
	}

	end := searchFrom + endOffset
	if start > end {
		return "", fmt.Errorf("invalid marker order in %s", readmePath)
	}

	return readme[:searchFrom] + "\n\n" + strings.TrimSpace(generatedSection) + "\n\n" + readme[end:], nil
}
