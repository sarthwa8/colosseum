package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/ladder"
	"github.com/sarthaksukhral/colosseum/internal/match"
	"github.com/sarthaksukhral/colosseum/internal/problem"
	"github.com/sarthaksukhral/colosseum/internal/rank"
	"github.com/sarthaksukhral/colosseum/internal/tui"
)

// cmdLadder runs a tournament across fighters/problems/formats and prints the
// eval report (the headline: does adversarial robustness diverge from pass rate?).
func cmdLadder(args []string) {
	fs := flag.NewFlagSet("ladder", flag.ExitOnError)
	fightersCSV := fs.String("fighters", "", "comma-separated fighter specs, e.g. anthropic:claude-haiku-4-5,ollama:qwen2.5-coder")
	problemsCSV := fs.String("problems", "all", "comma-separated problem slugs, or 'all'")
	formatsCSV := fs.String("formats", "race,ad", "comma-separated formats (race, ad)")
	rounds := fs.Int("rounds", 1, "rounds per (pair, problem, format)")
	maxIters := fs.Int("max-iters", 3, "max debug iterations per fighter")
	dir := fs.String("problems-dir", "problems", "problems root")
	dataDir := fs.String("data-dir", "data/matches", "where to save match records")
	outJSON := fs.String("out", "data/report.json", "eval report JSON output")
	outMD := fs.String("report", "data/report.txt", "eval report text output")
	bootstrap := fs.Int("bootstrap", 1000, "bootstrap iterations for Elo CIs")
	docker := fs.String("docker", "", "docker binary")
	seed := fs.Int64("seed", 1, "schedule + bootstrap seed")
	_ = fs.Parse(args)

	if *fightersCSV == "" {
		fmt.Fprintln(os.Stderr, "error: --fighters is required (>= 2 specs)")
		fs.Usage()
		os.Exit(2)
	}

	// Load problems.
	var probs []*problem.Problem
	if *problemsCSV == "all" {
		all, err := problem.LoadAll(*dir)
		if err != nil {
			fatal("load problems: %v", err)
		}
		probs = all
	} else {
		for _, slug := range splitCSV(*problemsCSV) {
			p, err := problem.Load(filepath.Join(*dir, slug))
			if err != nil {
				fatal("load problem %s: %v", slug, err)
			}
			probs = append(probs, p)
		}
	}
	if len(probs) == 0 {
		fatal("no problems found")
	}

	// Build fighters.
	var fighters []ladder.Fighter
	for _, spec := range splitCSV(*fightersCSV) {
		f, err := fighterFactory(spec)
		if err != nil {
			fatal("fighter %q: %v", spec, err)
		}
		fighters = append(fighters, f)
	}
	if len(fighters) < 2 {
		fatal("need at least 2 fighters")
	}

	// Build formats.
	var formats []match.Format
	for _, name := range splitCSV(*formatsCSV) {
		switch name {
		case "race":
			formats = append(formats, match.Race{})
		case "ad", "attack_defense":
			formats = append(formats, match.AttackDefense{})
		default:
			fatal("unknown format %q", name)
		}
	}

	j := judge.New(judge.NewDockerRunner(*docker))
	cfg := ladder.Config{
		Fighters: fighters,
		Problems: probs,
		Formats:  formats,
		Rounds:   *rounds,
		Judge:    j,
		MaxIters: *maxIters,
		DataDir:  *dataDir,
		Seed:     *seed,
		Progress: func(done, total int, rec match.Record) {
			o := rec.Outcome
			w := o.WinnerID
			if w == "" {
				w = "draw"
			} else {
				w = o.Scores[w].Model
			}
			fmt.Printf("  %s %s %-14s → %s %s\n",
				tui.Dim(fmt.Sprintf("[%d/%d]", done, total)),
				rec.Manifest.Format, rec.Manifest.Problem,
				tui.Bold(w), tui.Dim("("+o.Reason+")"))
		},
	}

	fmt.Printf("%s  %d fighters · %d problems · %v · %d rounds\n",
		tui.Bold("⚔  ladder"), len(fighters), len(probs), splitCSV(*formatsCSV), *rounds)
	fmt.Println(tui.Dim(strings.Repeat("─", 60)))

	summaries, err := ladder.Run(context.Background(), cfg)
	if err != nil {
		fatal("ladder: %v", err)
	}

	report := rank.Build(summaries, *bootstrap, *seed)
	fmt.Println(tui.Dim(strings.Repeat("─", 60)))
	report.Render(os.Stdout)

	// Persist report JSON + text.
	if b, err := json.MarshalIndent(report, "", "  "); err == nil {
		_ = os.MkdirAll(filepath.Dir(*outJSON), 0o755)
		if os.WriteFile(*outJSON, b, 0o644) == nil {
			fmt.Println(tui.Dim("report json  → " + *outJSON))
		}
	}
	if f, err := os.Create(*outMD); err == nil {
		report.Render(f)
		f.Close()
		fmt.Println(tui.Dim("report text  → " + *outMD))
	}
}

// fighterFactory turns a spec into a ladder.Fighter. Real providers share one
// instance across problems; mock fighters are rebuilt per problem so they can
// serve problem-specific solutions and attacks.
func fighterFactory(spec string) (ladder.Fighter, error) {
	provider, model, ok := strings.Cut(spec, ":")
	if !ok {
		return ladder.Fighter{}, fmt.Errorf("spec must be provider:model")
	}
	switch provider {
	case "anthropic":
		p := agent.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))
		return ladder.Fighter{Label: model, New: func(*problem.Problem) agent.Provider { return p }}, nil
	case "ollama":
		p := agent.NewOllamaProvider(os.Getenv("OLLAMA_BASE_URL"))
		return ladder.Fighter{Label: model, New: func(*problem.Problem) agent.Provider { return p }}, nil
	case "openai":
		base := os.Getenv("OPENAI_BASE_URL")
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		p := agent.NewOpenAICompatProvider(base, os.Getenv("OPENAI_API_KEY"), "openai")
		return ladder.Fighter{Label: model, New: func(*problem.Problem) agent.Provider { return p }}, nil
	case "mock":
		if model != "reference" && model != "wrong" {
			return ladder.Fighter{}, fmt.Errorf("mock model must be 'reference' or 'wrong'")
		}
		label := "mock:" + model
		return ladder.Fighter{Label: label, New: func(prob *problem.Problem) agent.Provider {
			attackInput := fenceRaw(edgeCaseInput(prob))
			if model == "reference" {
				return agent.NewFuncProvider("reference", func(r agent.Request) string {
					if isAttackReq(r) {
						return attackInput
					}
					return fence(prob.Reference)
				})
			}
			return agent.NewFuncProvider("wrong", func(r agent.Request) string {
				if isAttackReq(r) {
					return attackInput
				}
				return "```python\nprint(0)\n```"
			})
		}}, nil
	default:
		return ladder.Fighter{}, fmt.Errorf("unknown provider %q", provider)
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
