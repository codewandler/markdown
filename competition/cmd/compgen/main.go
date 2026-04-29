// Command compgen generates COMPARISON.md from a competition results file.
//
// Usage:
//
//	go run ./cmd/compgen --latest                          # use most recent results
//	go run ./cmd/compgen --results results/results-*.json  # use specific file
//	go run ./cmd/compgen --latest --out ../COMPARISON.md   # write to file
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codewandler/markdown/competition"
)

func main() {
	latest := flag.Bool("latest", false, "use the most recent results file (default behavior)")
	resultsFile := flag.String("results", "", "path to a specific results JSON file")
	resultsDir := flag.String("results-dir", competition.ResultsDir, "directory containing result snapshots")
	outFile := flag.String("out", "", "output file (default: stdout)")
	flag.Parse()

	// Load results.
	var r *competition.RunResult
	var err error

	if *resultsFile != "" {
		r, err = competition.LoadResults(*resultsFile)
		if err != nil {
			fatal("load results: %v", err)
		}
	} else {
		// Default: --latest behavior.
		_ = latest
		r, err = competition.LoadLatestResults(*resultsDir)
		if err != nil {
			fatal("load latest results: %v", err)
		}
		if r == nil {
			fatal("no results found in %s — run 'comprun metadata' first", *resultsDir)
		}
	}

	// Generate report.
	w := os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fatal("create %s: %v", *outFile, err)
		}
		defer f.Close()
		w = f
	}

	if err := competition.GenerateReport(w, r); err != nil {
		fatal("generate report: %v", err)
	}

	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", *outFile)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "compgen: "+format+"\n", args...)
	os.Exit(1)
}
