package ui

import (
	"errors"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

var ErrCancelled = errors.New("cancelled")

type Option struct {
	Key   rune
	Label string
	Desc  string
	Value string
}

type chooseModel struct {
	question  string
	options   []Option
	cursor    int
	choice    string
	cancelled bool
	done      bool
}

// IsInteractive reports whether a prompt can be both drawn and answered.
func IsInteractive() bool {
	return IsTerminal() && term.IsTerminal(os.Stdin.Fd())
}

func Choose(question string, options []Option) (string, error) {
	if !IsInteractive() {
		return "", ErrCancelled
	}

	trapInterrupt()

	prog := tea.NewProgram(
		chooseModel{question: question, options: options},
		tea.WithOutput(os.Stderr),
	)

	result, err := prog.Run()
	if err != nil {
		return "", err
	}

	model, ok := result.(chooseModel)
	if !ok || model.cancelled {
		return "", ErrCancelled
	}

	return model.choice, nil
}

func (m chooseModel) Init() tea.Cmd {
	return nil
}

func (m chooseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyUp:
		m.cursor = (m.cursor - 1 + len(m.options)) % len(m.options)
		return m, nil
	case tea.KeyDown, tea.KeyTab:
		m.cursor = (m.cursor + 1) % len(m.options)
		return m, nil
	case tea.KeyEnter:
		m.choice = m.options[m.cursor].Value
		m.done = true
		return m, tea.Quit
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		m.done = true
		return m, tea.Quit
	case tea.KeyRunes:
		for i, opt := range m.options {
			if key.Runes[0] == opt.Key {
				m.cursor = i
				m.choice = opt.Value
				m.done = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m chooseModel) View() string {
	if m.done {
		answer := "cancelled"
		if !m.cancelled {
			answer = m.options[m.cursor].Label
		}

		return cursorStyle.Render("?") + " " + m.question + " " + boldStyle.Render(answer) + "\n"
	}

	view := cursorStyle.Render("?") + " " + boldStyle.Render(m.question) + "\n"

	for i, opt := range m.options {
		prefix := "  "
		label := opt.Label

		if i == m.cursor {
			prefix = cursorStyle.Render(">") + " "
			label = cursorStyle.Render(label)
		}

		view += prefix + "[" + string(opt.Key) + "] " + label
		if opt.Desc != "" {
			view += " " + dimStyle.Render(opt.Desc)
		}
		view += "\n"
	}

	return view
}
