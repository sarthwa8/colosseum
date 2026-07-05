package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/sarthaksukhral/colosseum/internal/tui"
	"github.com/sarthaksukhral/colosseum/internal/web"
)

// cmdServe starts the spectator web UI, serving saved match records as
// browser-replayable arenas.
func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	dataDir := fs.String("data-dir", "data/matches", "match records directory")
	_ = fs.Parse(args)

	srv := web.NewServer(*dataDir)
	fmt.Printf("%s  http://localhost%s   %s\n", tui.Bold("⚔ colosseum arena"), *addr, tui.Dim("serving "+*dataDir))
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		fatal("serve: %v", err)
	}
}
