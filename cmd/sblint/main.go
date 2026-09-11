// Command sblint lints an Obsidian secondBrain vault (ADR 0004, rules 1-18).
//
// Exit codes: 0 = clean, 1 = violations found, 2 = internal error.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Thaelvyn/secondBrain-linter/internal/report"
	"github.com/Thaelvyn/secondBrain-linter/internal/rules"
	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// version is injected at build time: -ldflags "-X main.version=<tag>".
var version = "v0.1.1"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sblint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "machine-readable JSON output to stdout")
	writeReport := fs.Bool("report", false, "write a timestamped report to <vault>/_system/status/lint/")
	showVersion := fs.Bool("version", false, "print the sblint version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "sblint - secondBrain vault linter (ADR 0004, rules 1-18)\n\n")
		fmt.Fprintf(stderr, "Usage: sblint [flags] <vault-path>\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExit codes: 0 = clean, 1 = violations found, 2 = internal error.\n")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "sblint "+version)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	rootArg := fs.Arg(0)
	root := filepath.Clean(rootArg)

	v, err := vault.Scan(root)
	if err != nil {
		fmt.Fprintf(stderr, "sblint: %v\n", err)
		return 2
	}
	tax, terr := taxonomy.Load(root)
	context := rules.New(v, tax)
	if terr != nil {
		context.Add(4, vault.SeverityError, "_system/taxonomy.yaml", "cannot read _system/taxonomy.yaml: "+terr.Error())
	}

	rules.Run(context)

	sort.Slice(context.Findings, func(i, j int) bool {
		if context.Findings[i].Rule != context.Findings[j].Rule {
			return context.Findings[i].Rule < context.Findings[j].Rule
		}
		if context.Findings[i].Path != context.Findings[j].Path {
			return context.Findings[i].Path < context.Findings[j].Path
		}
		return context.Findings[i].Message < context.Findings[j].Message
	})

	errors, warnings := report.Counts(context.Findings)

	if *writeReport {
		path, rerr := report.WriteFile(root, report.Console(rootArg, context.Findings))
		if rerr != nil {
			fmt.Fprintf(stderr, "sblint: cannot write report: %v\n", rerr)
			return 2
		}
		fmt.Fprintf(stderr, "report written: %s\n", path)
	}

	if *jsonOut {
		data, jerr := report.JSON(rootArg, context.Findings)
		if jerr != nil {
			fmt.Fprintf(stderr, "sblint: cannot render JSON: %v\n", jerr)
			return 2
		}
		stdout.Write(append(data, '\n'))
	} else {
		fmt.Fprint(stdout, report.Console(rootArg, context.Findings))
	}

	if errors+warnings > 0 {
		return 1
	}
	return 0
}
