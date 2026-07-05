package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/events"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/match"
	"github.com/sarthaksukhral/colosseum/internal/problem"
	"github.com/sarthaksukhral/colosseum/internal/tui"
)

// cmdMatch runs a single match between two fighters and prints the event stream
// plus the outcome, saving a replayable record.
func cmdMatch(args []string) {
	fs := flag.NewFlagSet("match", flag.ExitOnError)
	slug := fs.String("problem", "", "problem slug")
	aSpec := fs.String("a", "", "fighter A spec (provider:model), e.g. anthropic:claude-haiku-4-5, ollama:qwen2.5-coder, mock:reference")
	bSpec := fs.String("b", "", "fighter B spec")
	format := fs.String("format", "race", "match format (race)")
	maxIters := fs.Int("max-iters", 3, "max debug iterations per fighter")
	budget := fs.Int("budget", 0, "per-fighter token budget (0 = unlimited)")
	dir := fs.String("problems-dir", "problems", "problems root")
	dataDir := fs.String("data-dir", "data/matches", "where to write the match record")
	docker := fs.String("docker", "", "docker binary (default: docker on PATH)")
	_ = fs.Parse(args)

	if *slug == "" || *aSpec == "" || *bSpec == "" {
		fmt.Fprintln(os.Stderr, "error: --problem, --a, and --b are required")
		fs.Usage()
		os.Exit(2)
	}

	prob, err := problem.Load(filepath.Join(*dir, *slug))
	if err != nil {
		fatal("load problem: %v", err)
	}

	aCfg, err := parseFighter(*aSpec, "A", prob)
	if err != nil {
		fatal("fighter A: %v", err)
	}
	bCfg, err := parseFighter(*bSpec, "B", prob)
	if err != nil {
		fatal("fighter B: %v", err)
	}

	var fmtImpl match.Format
	switch *format {
	case "race":
		fmtImpl = match.Race{}
	case "ad", "attack_defense":
		fmtImpl = match.AttackDefense{}
	default:
		fatal("unknown format %q (use race or ad)", *format)
	}

	fighters := map[string]agent.Config{"A": aCfg, "B": bCfg}
	j := judge.New(judge.NewDockerRunner(*docker))
	m := match.New(prob, j, fighters, []string{"A", "B"}, fmtImpl, *maxIters, *budget)

	fmt.Printf("%s  %s\n", tui.Bold("⚔  "+prob.Title), tui.Dim(fmt.Sprintf("%s · %s vs %s", *format, aCfg.Model, bCfg.Model)))
	fmt.Println(tui.Dim(strings.Repeat("─", 50)))

	// Print events live as the match runs.
	sub, cancel := m.Log.Subscribe(1)
	printDone := make(chan struct{})
	go func() {
		for ev := range sub {
			printEvent(ev)
		}
		close(printDone)
	}()

	outcome, err := m.Run(context.Background(), fmtImpl)
	cancel()
	<-printDone
	if err != nil {
		fatal("match: %v", err)
	}

	fmt.Println(tui.Dim(strings.Repeat("─", 50)))
	printOutcome(outcome, fmtImpl.Name())

	rec := m.ToRecord(outcome)
	if path, err := match.SaveRecord(*dataDir, rec); err == nil {
		fmt.Println(tui.Dim("record saved → " + path))
	}
}

// parseFighter turns a "provider:model" spec into a fighter config. Mock
// fighters need no API key or network: "mock:reference" replays the problem's
// pinned reference solution (always solves); "mock:wrong" always fails.
func parseFighter(spec, id string, prob *problem.Problem) (agent.Config, error) {
	provider, model, ok := strings.Cut(spec, ":")
	if !ok {
		return agent.Config{}, fmt.Errorf("spec must be provider:model, got %q", spec)
	}
	cfg := agent.Config{ID: id, Model: model, MaxTokens: 4096, Temperature: 0.7}

	switch provider {
	case "anthropic":
		cfg.Provider = agent.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))
	case "ollama":
		cfg.Provider = agent.NewOllamaProvider(os.Getenv("OLLAMA_BASE_URL"))
	case "openai":
		base := os.Getenv("OPENAI_BASE_URL")
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		cfg.Provider = agent.NewOpenAICompatProvider(base, os.Getenv("OPENAI_API_KEY"), "openai")
	case "mock":
		// Role-aware offline fighters: they defend with a solution and, when
		// prompted to attack, submit one of the problem's own edge-case inputs —
		// so `match --format ad` shows a real, oracle-validated break with no API.
		attackInput := fenceRaw(edgeCaseInput(prob))
		switch model {
		case "reference":
			cfg.Provider = agent.NewFuncProvider("reference", func(r agent.Request) string {
				if isAttackReq(r) {
					return attackInput
				}
				return fence(prob.Reference)
			})
			cfg.Model = "mock:reference"
		case "wrong":
			cfg.Provider = agent.NewFuncProvider("wrong", func(r agent.Request) string {
				if isAttackReq(r) {
					return attackInput
				}
				return "```python\nprint(0)\n```"
			})
			cfg.Model = "mock:wrong"
		default:
			return agent.Config{}, fmt.Errorf("mock model must be 'reference' or 'wrong', got %q", model)
		}
	default:
		return agent.Config{}, fmt.Errorf("unknown provider %q (use anthropic, ollama, openai, or mock)", provider)
	}
	return cfg, nil
}

func fence(code string) string {
	return "```python\n" + strings.TrimSpace(code) + "\n```"
}

func fenceRaw(text string) string {
	return "```\n" + strings.TrimSpace(text) + "\n```"
}

// isAttackReq detects whether a fighter is being asked to attack (vs solve),
// by inspecting the system prompt persona.
func isAttackReq(r agent.Request) bool {
	return strings.Contains(strings.ToLower(r.System), "adversarial tester")
}

// edgeCaseInput picks a problem test input whose answer is non-trivial (not
// "0"), so a print(0)-style buggy defender is provably broken by it. Falls back
// to the last case.
func edgeCaseInput(prob *problem.Problem) string {
	for _, c := range prob.Cases {
		if strings.TrimSpace(c.Expected) != "0" {
			return c.Input
		}
	}
	if len(prob.Cases) > 0 {
		return prob.Cases[len(prob.Cases)-1].Input
	}
	return ""
}

func printEvent(ev events.Event) {
	actor := ev.Actor
	if actor == "" {
		actor = "system"
	}
	switch ev.Type {
	case events.FighterThinking:
		fmt.Printf("  %s %s thinking (iter %v)\n", tui.Dim("·"), tui.Bold(actor), ev.Payload["iteration"])
	case events.Submission:
		fmt.Printf("  %s %s submission: %s  %v/%v\n", tui.Dim("→"), tui.Bold(actor),
			ev.Payload["verdict"], ev.Payload["passed"], ev.Payload["total"])
	case events.FighterForfeit:
		fmt.Printf("  %s %s forfeit: %v\n", tui.Dim("✗"), tui.Bold(actor), ev.Payload["reason"])
	case events.PhaseStarted:
		if ev.Payload["phase"] == "attack" {
			fmt.Printf("  %s attack: %v → %v\n", tui.Dim("▸"), ev.Payload["attacker"], ev.Payload["defender"])
		} else {
			fmt.Printf("  %s phase: %v\n", tui.Dim("▸"), ev.Payload["phase"])
		}
	case events.AttackSubmitted:
		in := fmt.Sprintf("%v", ev.Payload["input"])
		fmt.Printf("  %s %s attacks with input %s\n", tui.Dim("⚑"), tui.Bold(actor), tui.Dim(oneLine(in)))
	case events.AttackResult:
		broke, _ := ev.Payload["broke"].(bool)
		mark := tui.Dim("shielded")
		if broke {
			mark = tui.Bold("BROKE DEFENDER")
		}
		fmt.Printf("  %s %s → %s  %s\n", tui.Dim("⚔"), tui.Bold(actor), mark, tui.Dim(fmt.Sprintf("%v", ev.Payload["detail"])))
	case events.MatchFinished:
		// summarized by printOutcome
	}
}

func printOutcome(o match.Outcome, format string) {
	if o.WinnerID == "" {
		fmt.Printf("%s  %s\n", tui.Bold("DRAW"), tui.Dim("("+o.Reason+")"))
	} else {
		fmt.Printf("%s %s  %s\n", tui.Bold("WINNER:"), tui.Bold(o.WinnerID), tui.Dim("("+o.Reason+")"))
	}
	for _, id := range []string{"A", "B"} {
		s, ok := o.Scores[id]
		if !ok {
			continue
		}
		solved := yesno(s.Solved)
		if format == "attack_defense" {
			fmt.Printf("  %s %-14s solved=%-3s survived=%-3s broke=%-3s tokens=%d/%d cost=$%.4f\n",
				tui.Bold(id), s.Model, solved, yesno(s.Survived), yesno(s.Broke),
				s.TokensIn, s.TokensOut, s.CostUSD)
		} else {
			fmt.Printf("  %s %-14s solved=%-3s cases=%d/%d iters=%d tokens=%d/%d cost=$%.4f %s\n",
				tui.Bold(id), s.Model, solved, s.CasesPassed, s.CasesTotal, s.Iterations,
				s.TokensIn, s.TokensOut, s.CostUSD, tui.Dim(fmt.Sprintf("%dms", s.WallMs)))
		}
	}
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "⏎")
	if len(s) > 60 {
		s = s[:60] + "…"
	}
	return s
}
