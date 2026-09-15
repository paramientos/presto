package scripts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aras/presto/internal/parser"
	"github.com/aras/presto/internal/trust"
)

func composerWithScripts() *parser.ComposerJSON {
	return &parser.ComposerJSON{
		Scripts: map[string]interface{}{
			"pre-install-cmd":  "touch pre.txt",
			"post-install-cmd": []interface{}{"touch post.txt", "@php -v"},
			"test":             "phpunit",
		},
	}
}

func TestPendingListsOnlyTheScriptsPrestoTriggersItself(t *testing.T) {
	pending := Pending(composerWithScripts())

	if len(pending) != 2 {
		t.Fatalf("expected 2 lifecycle scripts, got %d", len(pending))
	}

	if pending[0].Event != "pre-install-cmd" {
		t.Fatalf("expected pre-install-cmd first, got %s", pending[0].Event)
	}

	if got := pending[1].Commands; len(got) != 2 || got[0] != "touch post.txt" {
		t.Fatalf("expected the array script flattened to its commands, got %v", got)
	}

	for _, script := range pending {
		if script.Event == "test" {
			t.Fatal("expected a named script to stay out of the trust prompt")
		}
	}
}

func TestAnUntrustedProjectRunsNothingAndReportsEveryEvent(t *testing.T) {
	t.Chdir(t.TempDir())

	lifecycle := NewLifecycle(NewRunner(false), composerWithScripts(), trust.ModeNever)

	lifecycle.Run("pre-install-cmd")
	lifecycle.Run("post-install-cmd")

	if _, err := os.Stat("pre.txt"); !os.IsNotExist(err) {
		t.Fatal("expected an untrusted project to run no script")
	}

	skipped := lifecycle.Skipped()
	if len(skipped) != 2 || skipped[0] != "pre-install-cmd" || skipped[1] != "post-install-cmd" {
		t.Fatalf("expected both events reported as skipped, got %v", skipped)
	}
}

func TestATrustedProjectRunsItsScripts(t *testing.T) {
	t.Chdir(t.TempDir())

	lifecycle := NewLifecycle(NewRunner(false), composerWithScripts(), trust.ModeAlways)

	lifecycle.Run("pre-install-cmd")

	if _, err := os.Stat("pre.txt"); err != nil {
		t.Fatalf("expected the script to run: %v", err)
	}

	if len(lifecycle.Skipped()) != 0 {
		t.Fatalf("expected nothing reported as skipped, got %v", lifecycle.Skipped())
	}
}

func TestAStoredTrustLetsTheScriptsRunWithoutAPrompt(t *testing.T) {
	project := t.TempDir()
	config := t.TempDir()

	t.Setenv("PRESTO_CONFIG_DIR", config)
	t.Chdir(project)

	composer := composerWithScripts()

	store, err := trust.LoadFrom(filepath.Join(config, "trust.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Allow(project, trust.Hash(Pending(composer))); err != nil {
		t.Fatal(err)
	}

	lifecycle := NewLifecycle(NewRunner(false), composer, trust.ModeAsk)
	lifecycle.Run("pre-install-cmd")

	if _, err := os.Stat("pre.txt"); err != nil {
		t.Fatalf("expected a stored trust to run the script: %v", err)
	}
}

func TestAnUnansweredPromptSkipsTheScripts(t *testing.T) {
	t.Setenv("PRESTO_CONFIG_DIR", t.TempDir())
	t.Chdir(t.TempDir())

	lifecycle := NewLifecycle(NewRunner(false), composerWithScripts(), trust.ModeAsk)
	lifecycle.Run("pre-install-cmd")

	if _, err := os.Stat("pre.txt"); !os.IsNotExist(err) {
		t.Fatal("expected a project nobody could be asked about to run no script")
	}

	if len(lifecycle.Skipped()) != 1 {
		t.Fatalf("expected the skipped event reported, got %v", lifecycle.Skipped())
	}
}

func TestAProjectWithoutScriptsNeedsNoTrust(t *testing.T) {
	t.Chdir(t.TempDir())

	lifecycle := NewLifecycle(NewRunner(false), &parser.ComposerJSON{}, trust.ModeAsk)
	lifecycle.Run("pre-install-cmd")

	if len(lifecycle.Skipped()) != 0 {
		t.Fatalf("expected nothing to skip, got %v", lifecycle.Skipped())
	}
}
