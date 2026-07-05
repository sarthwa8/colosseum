// Command colosseum is the entry point for the judge harness, match runner,
// ladder, and server. M1 ships the `judge` subcommand.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/problem"
	"github.com/sarthaksukhral/colosseum/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "judge":
		cmdJudge(os.Args[2:])
	case "match":
		cmdMatch(os.Args[2:])
	case "replay":
		cmdReplay(os.Args[2:])
	case "ladder":
		cmdLadder(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `colosseum — adversarial eval engine for coding agents

Usage:
  colosseum judge  --problem <slug> --file <source> [--lang python]
  colosseum match  --problem <slug> --a <provider:model> --b <provider:model> [--format race|ad]
  colosseum replay --file <record.json> [--speed 4.0] [--jsonl]
  colosseum ladder --fighters <spec,spec,...> [--formats race,ad] [--rounds N]
  colosseum serve  [--addr :8080]

Commands:
  judge    Run a source file against a problem's hidden tests in the sandbox
  match    Run an AI-vs-AI match (race or attack/defense) and save a replayable record
  replay   Re-render a saved match from its event log
  ladder   Run a tournament and print the eval report (Elo + robustness divergence)
  serve    Start the browser spectator UI to watch match replays

Fighter specs: anthropic:claude-haiku-4-5 · ollama:qwen2.5-coder · mock:reference · mock:wrong
`)
}

func cmdJudge(args []string) {
	fs := flag.NewFlagSet("judge", flag.ExitOnError)
	slug := fs.String("problem", "", "problem slug (directory under --problems-dir)")
	file := fs.String("file", "", "path to the source file to judge")
	lang := fs.String("lang", "python", "language (python)")
	dir := fs.String("problems-dir", "problems", "problems root directory")
	docker := fs.String("docker", "", "docker binary (default: docker on PATH)")
	workers := fs.Int("workers", 4, "max concurrent case containers")
	_ = fs.Parse(args)

	if *slug == "" || *file == "" {
		fmt.Fprintln(os.Stderr, "error: --problem and --file are required")
		fs.Usage()
		os.Exit(2)
	}

	prob, err := problem.Load(filepath.Join(*dir, *slug))
	if err != nil {
		fatal("load problem: %v", err)
	}
	code, err := os.ReadFile(*file)
	if err != nil {
		fatal("read source: %v", err)
	}

	sub := judge.Submission{
		Language: *lang,
		Code:     string(code),
		Cases:    prob.Cases,
		Limits:   prob.Limits,
	}

	fmt.Printf("%s  %s  %s\n",
		tui.Bold(prob.Title),
		tui.Dim("["+prob.Difficulty+"]"),
		tui.Dim(fmt.Sprintf("%d cases · %s", len(prob.Cases), *lang)))
	fmt.Println(tui.Dim("────────────────────────────────────────"))

	j := judge.New(judge.NewDockerRunner(*docker))
	j.Workers = *workers

	progress := make(chan judge.CaseResult, len(prob.Cases))
	var rep judge.Report
	var jerr error
	go func() {
		rep, jerr = j.JudgeStream(context.Background(), sub, progress)
		close(progress)
	}()

	start := time.Now()
	for cr := range progress {
		fmt.Printf("  %s  case %-8s %s\n",
			tui.VerdictBadge(cr.Verdict),
			cr.Name,
			tui.Dim(fmt.Sprintf("%dms", cr.Duration.Milliseconds())))
	}
	if jerr != nil {
		fatal("judge: %v", jerr)
	}

	fmt.Println(tui.Dim("────────────────────────────────────────"))
	if rep.Verdict == judge.CompileError && rep.Compile != nil {
		fmt.Printf("%s  %s\n", tui.VerdictBadge(rep.Verdict), tui.Bold("Compile Error"))
		if rep.Compile.Stderr != "" {
			fmt.Println(tui.Dim(rep.Compile.Stderr))
		}
	} else {
		fmt.Printf("%s  %s   %d/%d passed   %s\n",
			tui.VerdictBadge(rep.Verdict),
			tui.Bold(string(rep.Verdict)),
			rep.Passed, rep.Total,
			tui.Dim("total "+time.Since(start).Round(time.Millisecond).String()))
	}

	if rep.Verdict != judge.Accepted {
		os.Exit(1)
	}
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
