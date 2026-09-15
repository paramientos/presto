package scripts

import (
	"github.com/aras/presto/internal/parser"
	"github.com/aras/presto/internal/trust"
	"github.com/aras/presto/internal/ui"
)

// LifecycleEvents are the scripts presto triggers on its own, in the order an
// install fires them. Anything else only runs when the user asks for it by name.
var LifecycleEvents = []string{
	"pre-install-cmd",
	"pre-update-cmd",
	"pre-autoload-dump",
	"post-autoload-dump",
	"post-root-package-install",
	"post-install-cmd",
	"post-update-cmd",
}

type Lifecycle struct {
	runner   *Runner
	composer *parser.ComposerJSON
	mode     trust.Mode
	allowed  bool
	skipped  []string
}

func Pending(composer *parser.ComposerJSON) []trust.Script {
	if composer == nil || composer.Scripts == nil {
		return nil
	}

	var pending []trust.Script

	for _, event := range LifecycleEvents {
		script, ok := composer.Scripts[event]
		if !ok {
			continue
		}

		pending = append(pending, trust.Script{Event: event, Commands: commandsOf(script)})
	}

	return pending
}

func commandsOf(script interface{}) []string {
	switch v := script.(type) {
	case string:
		return []string{v}
	case []interface{}:
		commands := make([]string, 0, len(v))
		for _, command := range v {
			if s, ok := command.(string); ok {
				commands = append(commands, s)
			}
		}
		return commands
	}

	return nil
}

func NewLifecycle(runner *Runner, composer *parser.ComposerJSON, mode trust.Mode) *Lifecycle {
	return &Lifecycle{
		runner:   runner,
		composer: composer,
		mode:     mode,
		allowed:  trust.Decide(".", Pending(composer), mode),
	}
}

func (l *Lifecycle) Run(event string) {
	if l.composer.Scripts == nil {
		return
	}

	if _, ok := l.composer.Scripts[event]; !ok {
		return
	}

	if !l.allowed {
		l.skipped = append(l.skipped, event)
		return
	}

	if err := l.runner.Run(event, l.composer); err != nil {
		ui.Warn("%s: %v", event, err)
	}
}

func (l *Lifecycle) Skipped() []string {
	return l.skipped
}

func (l *Lifecycle) Report() {
	if len(l.skipped) == 0 {
		return
	}

	ui.Blank()
	ui.Warn("%d %s not run: %s", len(l.skipped), plural(len(l.skipped), "script was", "scripts were"), l.reason())

	for _, event := range l.skipped {
		ui.Status("  %s", ui.Bold(event))
	}

	if l.mode != trust.ModeNever {
		ui.Note("  run `presto trust` to allow them")
	}
}

func (l *Lifecycle) reason() string {
	if l.mode == trust.ModeNever {
		return "--no-scripts was set"
	}

	return "this project is not trusted"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}

	return many
}
