// Command comprun executes competition pipeline stages.
//
// Usage:
//
//	go run ./cmd/comprun metadata              # Stage 1: discover metadata
//	go run ./cmd/comprun metadata --refresh    # force re-discovery
//	go run ./cmd/comprun compliance            # Stage 2: run spec suites
//	go run ./cmd/comprun bench                 # Stage 3: run benchmarks
//	go run ./cmd/comprun bench --count=5       # 5 iterations per benchmark
//	go run ./cmd/comprun full                  # all stages in sequence
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codewandler/markdown/competition"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "metadata":
		cmdMetadata(os.Args[2:])
	case "compliance":
		cmdCompliance(os.Args[2:])
	case "bench":
		cmdBench(os.Args[2:])
	case "full":
		cmdFull(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: comprun <command> [flags]

Commands:
  metadata    Stage 1: Discover metadata for all candidates
  compliance  Stage 2: Run CommonMark + GFM spec suites
  bench       Stage 3: Run benchmarks
  full        Run all stages in sequence

Common flags:
  --results-dir   Results directory (default "results")

Metadata flags:
  --refresh       Force re-discovery (ignore cache)
  --max-age       Staleness threshold (default 24h)

Bench flags:
  --count         Benchmark iterations (default 3)
  --benchtime     Benchmark duration (default "1s")
  --bench         Benchmark pattern (default ".")
  --timeout       Test timeout (default "300s")`)
}

// --- metadata ---------------------------------------------------------------

func cmdMetadata(args []string) {
	fs := flag.NewFlagSet("metadata", flag.ExitOnError)
	refresh := fs.Bool("refresh", false, "force re-discovery of all candidates")
	maxAge := fs.Duration("max-age", 24*time.Hour, "staleness threshold for cached metadata")
	resultsDir := fs.String("results-dir", competition.ResultsDir, "directory for result snapshots")
	fs.Parse(args)

	fmt.Fprintln(os.Stderr, "Stage 1: Metadata Discovery")
	fmt.Fprintln(os.Stderr)

	result, err := competition.RunMetadata(competition.All, competition.MetadataOptions{
		MaxAge:     *maxAge,
		ForceAll:   *refresh,
		ResultsDir: *resultsDir,
		Log:        os.Stderr,
	})
	if err != nil {
		fatal("metadata: %v", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Discovered %d candidates\n", len(result.Candidates))
}

// --- compliance -------------------------------------------------------------

func cmdCompliance(args []string) {
	fs := flag.NewFlagSet("compliance", flag.ExitOnError)
	resultsDir := fs.String("results-dir", competition.ResultsDir, "directory for result snapshots")
	fs.Parse(args)

	fmt.Fprintln(os.Stderr, "Stage 2: Compliance Testing")
	fmt.Fprintln(os.Stderr)

	_, err := competition.RunCompliance(competition.All, competition.ComplianceOptions{
		ResultsDir: *resultsDir,
		Log:        os.Stderr,
	})
	if err != nil {
		fatal("compliance: %v", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Compliance testing complete")
}

// --- bench ------------------------------------------------------------------

func cmdBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	resultsDir := fs.String("results-dir", competition.ResultsDir, "directory for result snapshots")
	count := fs.Int("count", 3, "benchmark iterations")
	benchTime := fs.String("benchtime", "1s", "benchmark duration per iteration")
	benchPattern := fs.String("bench", ".", "benchmark pattern")
	timeout := fs.String("timeout", "300s", "test timeout")
	fs.Parse(args)

	fmt.Fprintln(os.Stderr, "Stage 3: Benchmarks")
	fmt.Fprintln(os.Stderr)

	_, err := competition.RunBenchmarks(competition.BenchmarkOptions{
		ResultsDir:   *resultsDir,
		BenchPattern: *benchPattern,
		Count:        *count,
		BenchTime:    *benchTime,
		Timeout:      *timeout,
		Log:          os.Stderr,
	})
	if err != nil {
		fatal("bench: %v", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Benchmarks complete")
}

// --- full -------------------------------------------------------------------

func cmdFull(args []string) {
	fs := flag.NewFlagSet("full", flag.ExitOnError)
	resultsDir := fs.String("results-dir", competition.ResultsDir, "directory for result snapshots")
	count := fs.Int("count", 3, "benchmark iterations")
	benchTime := fs.String("benchtime", "1s", "benchmark duration per iteration")
	refresh := fs.Bool("refresh", false, "force metadata re-discovery")
	fs.Parse(args)

	// Stage 1: Metadata
	fmt.Fprintln(os.Stderr, "=== Stage 1: Metadata Discovery ===")
	fmt.Fprintln(os.Stderr)
	_, err := competition.RunMetadata(competition.All, competition.MetadataOptions{
		ForceAll:   *refresh,
		ResultsDir: *resultsDir,
		Log:        os.Stderr,
	})
	if err != nil {
		fatal("metadata: %v", err)
	}

	// Stage 2: Compliance
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "=== Stage 2: Compliance Testing ===")
	fmt.Fprintln(os.Stderr)
	_, err = competition.RunCompliance(competition.All, competition.ComplianceOptions{
		ResultsDir: *resultsDir,
		Log:        os.Stderr,
	})
	if err != nil {
		fatal("compliance: %v", err)
	}

	// Stage 3: Benchmarks
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "=== Stage 3: Benchmarks ===")
	fmt.Fprintln(os.Stderr)
	_, err = competition.RunBenchmarks(competition.BenchmarkOptions{
		ResultsDir: *resultsDir,
		Count:      *count,
		BenchTime:  *benchTime,
		Log:        os.Stderr,
	})
	if err != nil {
		fatal("bench: %v", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "=== Full pipeline complete ===")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "comprun: "+format+"\n", args...)
	os.Exit(1)
}
