package trust

import (
	"errors"

	"github.com/aras/presto/internal/ui"
)

type Mode int

const (
	ModeAsk Mode = iota
	ModeAlways
	ModeNever
)

type Script struct {
	Event    string
	Commands []string
}

// Decide prompts once for the whole project, not once per event.
func Decide(projectDir string, scripts []Script, mode Mode) bool {
	if len(scripts) == 0 {
		return true
	}

	switch mode {
	case ModeNever:
		return false
	case ModeAlways:
		return true
	}

	hash := Hash(scripts)

	store, err := Load()
	if err != nil {
		ui.Warn("%v", err)
		return false
	}

	if store.Allows(projectDir, hash) {
		return true
	}

	if !ui.IsInteractive() {
		return false
	}

	return ask(store, projectDir, hash, scripts)
}

func ask(store *Store, projectDir, hash string, scripts []Script) bool {
	ui.Blank()
	ui.Warn("this project defines scripts that presto would run")
	ui.Blank()
	Describe(scripts)
	ui.Blank()

	choice, err := ui.Choose("Run these scripts?", []ui.Option{
		{Key: 'o', Label: "once", Desc: "run them for this install only", Value: "once"},
		{Key: 'a', Label: "always", Desc: "trust this project from now on", Value: "always"},
		{Key: 'n', Label: "never", Desc: "skip them", Value: "never"},
	})
	if err != nil {
		if !errors.Is(err, ui.ErrCancelled) {
			ui.Warn("%v", err)
		}
		return false
	}

	if choice == "always" {
		if err := store.Allow(projectDir, hash); err != nil {
			ui.Warn("%v", err)
		}
	}

	return choice == "once" || choice == "always"
}

func Describe(scripts []Script) {
	for _, script := range scripts {
		ui.Status("  %s", ui.Bold(script.Event))
		for _, command := range script.Commands {
			ui.Status("    %s", ui.Dim(command))
		}
	}
}
