package packagist

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aras/presto/internal/cache"
)

const demoManifest = `{"packages":{"acme/demo":[{"version":"1.0.0","description":"demo","dist":{"type":"zip","url":"https://example.test/demo.zip"}}]}}`

func serveManifest(t *testing.T, etag string, hits *int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++

		if r.Header.Get("If-None-Match") == etag && etag != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(demoManifest))
	}))

	t.Cleanup(server.Close)

	return server
}

func clientFor(url string) *Client {
	client := NewClient()
	client.baseURL = url

	return client
}

func age(t *testing.T, name string, d time.Duration) {
	t.Helper()

	path, err := cache.Metadata(name)
	if err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().Add(-d)

	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}

func TestASecondProcessReadsTheManifestFromDiskWithoutAsking(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", t.TempDir())

	hits := 0
	server := serveManifest(t, `"v1"`, &hits)

	if _, err := clientFor(server.URL).GetPackage("acme/demo"); err != nil {
		t.Fatal(err)
	}

	if _, err := clientFor(server.URL).GetPackage("acme/demo"); err != nil {
		t.Fatal(err)
	}

	if hits != 1 {
		t.Fatalf("expected one request inside the freshness window, got %d", hits)
	}
}

func TestAStaleManifestIsRevalidatedRatherThanRefetched(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", t.TempDir())

	hits := 0
	server := serveManifest(t, `"v1"`, &hits)

	if _, err := clientFor(server.URL).GetPackage("acme/demo"); err != nil {
		t.Fatal(err)
	}

	age(t, "acme/demo", manifestTTL+time.Minute)

	info, err := clientFor(server.URL).GetPackage("acme/demo")
	if err != nil {
		t.Fatal(err)
	}

	if hits != 2 {
		t.Fatalf("expected the stale copy to be revalidated, got %d requests", hits)
	}

	if _, ok := info.Versions["1.0.0"]; !ok {
		t.Fatal("expected the 304 to be answered from the cached body")
	}

	if !cachedManifestIsFresh(mustPath(t, "acme/demo")) {
		t.Fatal("expected a 304 to restart the freshness window")
	}
}

func TestACachedManifestAnswersWhenPackagistCannotBeReached(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", t.TempDir())

	hits := 0
	server := serveManifest(t, `"v1"`, &hits)

	if _, err := clientFor(server.URL).GetPackage("acme/demo"); err != nil {
		t.Fatal(err)
	}

	age(t, "acme/demo", manifestTTL+time.Minute)
	server.Close()

	info, err := clientFor(server.URL).GetPackage("acme/demo")
	if err != nil {
		t.Fatalf("expected the cached manifest to answer offline: %v", err)
	}

	if _, ok := info.Versions["1.0.0"]; !ok {
		t.Fatal("expected the cached versions to come back")
	}
}

func TestAnUncachedPackageStillFailsWhenPackagistCannotBeReached(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", t.TempDir())

	hits := 0
	server := serveManifest(t, `"v1"`, &hits)
	server.Close()

	if _, err := clientFor(server.URL).GetPackage("acme/demo"); err == nil {
		t.Fatal("expected an error with nothing cached to fall back on")
	}
}

func mustPath(t *testing.T, name string) string {
	t.Helper()

	path, err := cache.Metadata(name)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func TestRequireAsksForARangeNotAnExactVersion(t *testing.T) {
	cases := map[string]string{
		"v8.1.0":      "^8.1",
		"3.0.2":       "^3.0",
		"v10.50.3":    "^10.50",
		"v9.18.1.10":  "^9.18",
		"0.20.0":      "^0.20.0",
		"0.12.3":      "^0.12.3",
		"1.0.0-beta1": "1.0.0-beta1",
	}

	for release, want := range cases {
		t.Run(release, func(t *testing.T) {
			if got := RecommendedConstraint(release); got != want {
				t.Errorf("RecommendedConstraint(%q) = %q, want %q", release, got, want)
			}
		})
	}
}
