package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

type Toast struct {
	Message   string
	Level     ToastLevel
	ExpiresAt time.Time
}

type ToastExpiredMsg struct {
	ID int
}

type Toaster struct {
	toasts    []Toast
	width     int
	theme     *theme.Theme
	idCounter int
}

func NewToaster(t *theme.Theme) Toaster {
	return Toaster{
		theme:  t,
		toasts: []Toast{},
	}
}

func (t *Toaster) SetWidth(width int) {
	t.width = width
}

func (t *Toaster) Show(message string, level ToastLevel, duration time.Duration) tea.Cmd {
	toast := Toast{
		Message:   message,
		Level:     level,
		ExpiresAt: time.Now().Add(duration),
	}
	t.toasts = append(t.toasts, toast)
	t.idCounter++
	id := t.idCounter

	return tea.Tick(duration, func(_ time.Time) tea.Msg {
		return ToastExpiredMsg{ID: id}
	})
}

func (t *Toaster) Info(message string) tea.Cmd {
	return t.Show(message, ToastInfo, 3*time.Second)
}

func (t *Toaster) Success(message string) tea.Cmd {
	return t.Show(message, ToastSuccess, 5*time.Second)
}

func (t *Toaster) Warning(message string) tea.Cmd {
	return t.Show(message, ToastWarning, 4*time.Second)
}

func (t *Toaster) Error(message string) tea.Cmd {
	return t.Show(message, ToastError, 5*time.Second)
}

func (t *Toaster) Update(msg tea.Msg) {
	switch msg.(type) {
	case ToastExpiredMsg:
		t.removeExpired()
	}
}

func (t *Toaster) removeExpired() {
	now := time.Now()
	active := make([]Toast, 0)
	for _, toast := range t.toasts {
		if toast.ExpiresAt.After(now) {
			active = append(active, toast)
		}
	}
	t.toasts = active
}

func (t *Toaster) HasToasts() bool {
	return len(t.toasts) > 0
}

func (t *Toaster) View() string {
	if len(t.toasts) == 0 {
		return ""
	}

	toast := t.toasts[len(t.toasts)-1]

	var style lipgloss.Style
	var icon string

	switch toast.Level {
	case ToastSuccess:
		style = t.theme.StatusSuccess
		icon = t.theme.Icons.Success + " "
	case ToastWarning:
		style = t.theme.StatusWarning
		icon = t.theme.Icons.InProgress + " "
	case ToastError:
		style = t.theme.StatusError
		icon = t.theme.Icons.Error + " "
	default:
		style = t.theme.Text
		icon = "ℹ "
	}

	content := icon + toast.Message

	toastStyle := lipgloss.NewStyle().
		Foreground(style.GetForeground()).
		Padding(0, 2).
		Bold(true)

	rendered := toastStyle.Render(content)

	return lipgloss.PlaceHorizontal(t.width, lipgloss.Right, rendered)
}
