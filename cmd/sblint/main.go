// Command sblint lints an Obsidian secondBrain vault (ADR 0004, rules 1-22).
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

	"github.com/Thaelvyn/secondBrain-linter/internal/changes"
	"github.com/Thaelvyn/secondBrain-linter/internal/report"
	"github.com/Thaelvyn/secondBrain-linter/internal/rules"
	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// version is injected at build time: -ldflags "-X main.version=<tag>".
var version = "v0.2.1"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sblint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "machine-readable JSON output to stdout")
	writeReport := fs.Bool("report", false, "write a timestamped report to <vault>/_system/status/lint/")
	fixFlag := fs.Bool("fix", false, "apply best-effort deterministic fixes (rules 19, 3, 11), then re-lint")
	recountFlag := fs.Bool("recount", false, "recompute and rewrite folder-note children_counts only (rule 19); implies fix")
	changedFlag := fs.Bool("changed", false, "report/fix only files changed vs git (mtime fallback without git)")
	showVersion := fs.Bool("version", false, "print the sblint version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "sblint - secondBrain vault linter (ADR 0004, rules 1-22)\n\n")
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

	var changedSet map[string]bool
	changeMode := changes.ModeMtime
	if *changedFlag {
		set, mode, cerr := changes.Detect(root)
		if cerr != nil {
			fmt.Fprintf(stderr, "sblint: --changed: %v\n", cerr)
			return 2
		}
		changedSet, changeMode = set, mode
	}

	rules.Run(context)

	if *fixFlag || *recountFlag {
		fixed, ferr := context.Fix(rules.FixOptions{RecountOnly: *recountFlag, Changed: changedSet})
		if ferr != nil {
			fmt.Fprintf(stderr, "sblint: fix: %v\n", ferr)
			return 2
		}
		label := "fixed"
		if *recountFlag {
			label = "recounted"
		}
		fmt.Fprintf(stderr, "%s %d file(s):\n", label, len(fixed))
		for _, p := range fixed {
			fmt.Fprintf(stderr, "  %s\n", p)
		}
		v, err = vault.Scan(root)
		if err != nil {
			fmt.Fprintf(stderr, "sblint: %v\n", err)
			return 2
		}
		context = rules.New(v, tax)
		if terr != nil {
			context.Add(4, vault.SeverityError, "_system/taxonomy.yaml", "cannot read _system/taxonomy.yaml: "+terr.Error())
		}
		rules.Run(context)
	}

	findings := context.Findings
	if *changedFlag {
		findings = rules.FilterByChanged(findings, changedSet)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})

	errors, warnings := report.Counts(findings)

	if *writeReport {
		path, rerr := report.WriteFile(root, report.Console(rootArg, findings))
		if rerr != nil {
			fmt.Fprintf(stderr, "sblint: cannot write report: %v\n", rerr)
			return 2
		}
		fmt.Fprintf(stderr, "report written: %s\n", path)
	}

	if *jsonOut {
		data, jerr := report.JSON(rootArg, findings)
		if jerr != nil {
			fmt.Fprintf(stderr, "sblint: cannot render JSON: %v\n", jerr)
			return 2
		}
		stdout.Write(append(data, '\n'))
	} else {
		fmt.Fprint(stdout, report.Console(rootArg, findings))
	}

	if *changedFlag && changeMode == changes.ModeMtime {
		if serr := changes.SaveMtime(root); serr != nil {
			fmt.Fprintf(stderr, "sblint: cannot save --changed state: %v\n", serr)
		}
	}

	if errors+warnings > 0 {
		return 1
	}
	return 0
}
