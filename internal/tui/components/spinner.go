package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

type Spinner struct {
	spinner spinner.Model
	active  bool
	label   string
	theme   *theme.Theme
}

func NewSpinner(t *theme.Theme) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().
		Foreground(t.Colors.Primary).
		Bold(true)

	return Spinner{
		spinner: s,
		theme:   t,
	}
}

func (s *Spinner) Start(label string) tea.Cmd {
	s.active = true
	s.label = label
	return s.spinner.Tick
}

func (s *Spinner) Stop() {
	s.active = false
	s.label = ""
}

func (s *Spinner) IsActive() bool {
	return s.active
}

func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	if !s.active {
		return nil
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return cmd
}

func (s *Spinner) View() string {
	if !s.active {
		return ""
	}

	labelStyle := s.theme.TextDim
	return s.spinner.View() + " " + labelStyle.Render(s.label)
}
