package wizard

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type spinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
	success bool
	err     error
	result  any
}

type spinnerCompleteMsg struct {
	result any
	err    error
}

func newSpinnerModel(message string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Globe
	s.Style = spinnerStyle
	return spinnerModel{
		spinner: s,
		message: message,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil

	case spinnerCompleteMsg:
		m.done = true
		m.err = msg.err
		m.result = msg.result
		m.success = msg.err == nil
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m spinnerModel) View() string {
	if m.done {
		if m.success {
			return successStyle.Render("✓ " + m.message + " complete\n")
		}
		return errorStyle.Render("✗ " + m.message + " failed: " + m.err.Error() + "\n")
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), messageStyle.Render(m.message))
}

func RunWithSpinner(ctx context.Context, message string, fn func() (any, error)) (any, error) {
	if !isTTY() {
		fmt.Println(messageStyle.Render(message + "..."))
		result, err := fn()
		if err != nil {
			fmt.Println(errorStyle.Render("✗ " + message + " failed: " + err.Error()))
			return nil, err
		}
		fmt.Println(successStyle.Render("✓ " + message + " complete"))
		return result, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m := newSpinnerModel(message)
	p := tea.NewProgram(m)

	go func() {
		time.Sleep(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			return
		default:
			result, err := fn()
			p.Send(spinnerCompleteMsg{result: result, err: err})
		}
	}()

	finalModel, err := p.Run()
	cancel()

	if err != nil {
		return nil, err
	}

	if sm, ok := finalModel.(spinnerModel); ok {
		return sm.result, sm.err
	}

	return nil, fmt.Errorf("unexpected model type")
}
