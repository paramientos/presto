package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheCacheDirectoryFollowsTheEnvironment(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", "/tmp/explicit")

	if Dir() != "/tmp/explicit" {
		t.Fatalf("expected PRESTO_CACHE_DIR to win, got %s", Dir())
	}

	t.Setenv("PRESTO_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg")

	if Dir() != filepath.Join("/tmp/xdg", "presto") {
		t.Fatalf("expected XDG_CACHE_HOME to be used, got %s", Dir())
	}
}

func TestAMissingHomeFallsBackToTempRatherThanFailing(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")

	if Dir() != filepath.Join(os.TempDir(), "presto") {
		t.Fatalf("expected a temp directory fallback, got %s", Dir())
	}
}

func TestAPackageNameCannotClimbOutOfTheCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PRESTO_CACHE_DIR", root)

	path, err := Metadata("../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(root)) {
		t.Fatalf("expected %s to stay under %s", path, root)
	}
}

func TestAnArchiveIsKeyedByItsDownloadURL(t *testing.T) {
	t.Setenv("PRESTO_CACHE_DIR", t.TempDir())

	first, err := Archive("acme/demo", "1.0.0", "https://example.test/a.zip")
	if err != nil {
		t.Fatal(err)
	}

	second, err := Archive("acme/demo", "1.0.0", "https://example.test/b.zip")
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("expected a republished version at another URL to use another cache entry")
	}

	again, err := Archive("acme/demo", "1.0.0", "https://example.test/a.zip")
	if err != nil {
		t.Fatal(err)
	}

	if first != again {
		t.Fatal("expected the same package, version and URL to reuse one cache entry")
	}
}

func TestClearRemovesTheWholeCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PRESTO_CACHE_DIR", root)

	path, err := Metadata("acme/demo")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Clear(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("expected the cache directory to be gone")
	}
}
