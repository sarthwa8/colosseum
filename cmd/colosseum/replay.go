package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sarthaksukhral/colosseum/internal/events"
	"github.com/sarthaksukhral/colosseum/internal/match"
	"github.com/sarthaksukhral/colosseum/internal/tui"
)

// cmdReplay re-renders a saved match from its event log. Because the log is the
// single source of truth, replay and the live broadcast produce the identical
// stream — this command just re-establishes the original timeline.
func cmdReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	file := fs.String("file", "", "path to a saved match record (data/matches/*.json)")
	speed := fs.Float64("speed", 4.0, "replay speed multiplier (0 = instant)")
	jsonl := fs.Bool("jsonl", false, "dump the event log as JSONL instead of replaying")
	_ = fs.Parse(args)

	if *file == "" {
		fmt.Fprintln(os.Stderr, "error: --file is required")
		fs.Usage()
		os.Exit(2)
	}

	rec, err := match.LoadRecord(*file)
	if err != nil {
		fatal("load record: %v", err)
	}

	if *jsonl {
		enc := json.NewEncoder(os.Stdout)
		for _, ev := range rec.Events {
			_ = enc.Encode(ev)
		}
		return
	}

	fmt.Printf("%s  %s\n", tui.Bold("▶ replay "+rec.Manifest.MatchID), tui.Dim(rec.Manifest.Format+" · "+rec.Manifest.Problem))
	fmt.Println(tui.Dim(strings.Repeat("─", 50)))

	_ = events.Replay(context.Background(), rec.Events, *speed, printEvent)

	fmt.Println(tui.Dim(strings.Repeat("─", 50)))
	printOutcome(rec.Outcome, rec.Manifest.Format)
}
