package ui

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

type Spinner struct {
	prog *tea.Program
	done chan struct{}
	once sync.Once
}

type titleMsg string

type detailMsg string

type spinnerModel struct {
	sp     spinner.Model
	title  string
	detail string
}

var (
	activeMu sync.Mutex
	active   *Spinner
	trapOnce sync.Once
)

// StartSpinner returns nil when stderr is not a terminal; every method tolerates
// a nil receiver so callers stay branch-free.
func StartSpinner(title string) *Spinner {
	if !IsTerminal() {
		return nil
	}

	trapInterrupt()

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = accentStyle

	s := &Spinner{done: make(chan struct{})}
	s.prog = tea.NewProgram(
		spinnerModel{sp: sp, title: title},
		tea.WithOutput(os.Stderr),
		tea.WithInput(nil),
		tea.WithoutSignalHandler(),
	)

	activeMu.Lock()
	active = s
	activeMu.Unlock()

	go func() {
		defer close(s.done)
		_, _ = s.prog.Run()
	}()

	return s
}

func (s *Spinner) Title(title string) {
	if s == nil {
		return
	}
	s.prog.Send(titleMsg(title))
}

func (s *Spinner) Detail(detail string) {
	if s == nil {
		return
	}
	s.prog.Send(detailMsg(detail))
}

func (s *Spinner) Stop() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		s.prog.Quit()
		<-s.done

		activeMu.Lock()
		if active == s {
			active = nil
		}
		activeMu.Unlock()
	})
}

func IsTerminal() bool {
	return term.IsTerminal(os.Stderr.Fd())
}

func trapInterrupt() {
	trapOnce.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-ch

			activeMu.Lock()
			s := active
			activeMu.Unlock()

			s.Stop()
			os.Exit(130)
		}()
	})
}

func (m spinnerModel) Init() tea.Cmd {
	return m.sp.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case titleMsg:
		m.title = string(msg)
		return m, nil
	case detailMsg:
		m.detail = string(msg)
		return m, nil
	}

	var cmd tea.Cmd
	m.sp, cmd = m.sp.Update(msg)

	return m, cmd
}

func (m spinnerModel) View() string {
	line := m.sp.View() + m.title

	if m.detail != "" {
		line += " " + dimStyle.Render(m.detail)
	}

	return line
}
