package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The UI is a committed Vite bundle served out of an embedded FS with an SPA
// fallback. These pin the routing contract: assets are immutable, the shell is
// not cached, unknown paths render the app rather than 404ing, and nothing can
// read outside the bundle.

func TestServesAppShell(t *testing.T) {
	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}
	// The shell points at hashed asset names, so caching it would pin clients
	// to a stale bundle after a redeploy.
	if cc := res.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("cache-control = %q, want no-cache", cc)
	}
}

func TestSPAFallbackForUnknownPaths(t *testing.T) {
	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/some/client/route")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SPA fallback)", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q, want the app shell", ct)
	}
}

func TestHashedAssetsAreImmutable(t *testing.T) {
	// Discover a real asset name from the embedded bundle rather than
	// hardcoding a hash that changes on every build.
	entries, err := content.ReadDir("dist/assets")
	if err != nil {
		t.Fatalf("read embedded assets: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no embedded assets — was the UI built?")
	}

	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/assets/" + entries[0].Name())
	if err != nil {
		t.Fatalf("get asset: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("cache-control = %q, want immutable (asset names are content-hashed)", cc)
	}
}

func TestStaticCannotEscapeBundle(t *testing.T) {
	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	// Raw path, unnormalized by the client, so the server does the cleaning.
	for _, target := range []string{
		"/../server.go",
		"/..%2f..%2fserver.go",
		"/assets/../../server.go",
	} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+target, nil)
		if err != nil {
			t.Fatalf("build request %q: %v", target, err)
		}
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatalf("get %q: %v", target, err)
		}
		body := make([]byte, 512)
		n, _ := res.Body.Read(body)
		res.Body.Close()

		if strings.Contains(string(body[:n]), "package web") {
			t.Errorf("%s leaked Go source out of the bundle", target)
		}
	}
}

func TestListMatchesEmptyDir(t *testing.T) {
	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/matches")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	var out []matchSummary
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// An empty arena must be a valid empty list, not null — the UI maps over it.
	if out == nil {
		t.Error("want [] for an empty data dir, got null")
	}
}

func TestGetMatchRejectsBadIDs(t *testing.T) {
	srv := httptest.NewServer(NewServer(t.TempDir()).Handler())
	defer srv.Close()

	for _, id := range []string{"../server", "a/b", "x.json"} {
		res, err := http.Get(srv.URL + "/api/matches/" + id)
		if err != nil {
			t.Fatalf("get %q: %v", id, err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			t.Errorf("id %q was accepted, want rejection", id)
		}
	}
}
