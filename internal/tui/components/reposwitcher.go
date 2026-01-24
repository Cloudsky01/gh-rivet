package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

type RepoSwitcher struct {
	active       bool
	input        string
	repositories []config.Repository
	filtered     []config.Repository
	currentRepo  string
	cursor       int
	width        int
	height       int
	theme        *theme.Theme
}

func NewRepoSwitcher(t *theme.Theme) RepoSwitcher {
	return RepoSwitcher{
		theme:        t,
		repositories: []config.Repository{},
		filtered:     []config.Repository{},
	}
}

func (r *RepoSwitcher) SetRepositories(repos []config.Repository, current string) {
	r.repositories = repos
	r.currentRepo = current
	r.applyFilter()
}

func (r *RepoSwitcher) SetSize(width, height int) {
	r.width = width
	r.height = height
}

func (r *RepoSwitcher) IsActive() bool {
	return r.active
}

func (r *RepoSwitcher) Open() {
	r.active = true
	r.input = ""
	r.cursor = 0
	r.applyFilter()
}

func (r *RepoSwitcher) Close() {
	r.active = false
	r.input = ""
	r.cursor = 0
}

func (r *RepoSwitcher) applyFilter() {
	if r.input == "" {
		r.filtered = r.repositories
		return
	}

	matches := fuzzy.FindFrom(r.input, repositorySource(r.repositories))
	r.filtered = make([]config.Repository, len(matches))
	for i, match := range matches {
		r.filtered[i] = r.repositories[match.Index]
	}
	r.cursor = 0
}

type repositorySource []config.Repository

func (rs repositorySource) String(i int) string {
	repo := rs[i]
	searchStr := repo.Repository
	if repo.Alias != "" {
		searchStr += " " + repo.Alias
	}
	return searchStr
}

func (rs repositorySource) Len() int {
	return len(rs)
}

func (r *RepoSwitcher) Update(msg tea.Msg) (*config.Repository, tea.Cmd) {
	if !r.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			r.Close()
			return nil, nil
		case "enter":
			if r.cursor >= 0 && r.cursor < len(r.filtered) {
				selected := r.filtered[r.cursor]
				r.Close()
				return &selected, nil
			}
			r.Close()
			return nil, nil
		case "up", "ctrl+p":
			if r.cursor > 0 {
				r.cursor--
			}
			return nil, nil
		case "down", "ctrl+n":
			if r.cursor < len(r.filtered)-1 {
				r.cursor++
			}
			return nil, nil
		case "backspace":
			if len(r.input) > 0 {
				r.input = r.input[:len(r.input)-1]
				r.applyFilter()
			}
			return nil, nil
		case "tab":
			if r.cursor >= 0 && r.cursor < len(r.filtered) {
				r.input = r.filtered[r.cursor].Repository
				r.applyFilter()
			}
			return nil, nil
		default:
			key := msg.String()
			if len(key) == 1 {
				r.input += key
				r.applyFilter()
			}
			return nil, nil
		}
	}
	return nil, nil
}

func (r *RepoSwitcher) View() string {
	if !r.active {
		return ""
	}

	overlayWidth := max(50, r.width*50/100)
	overlayHeight := max(10, min(20, r.height*50/100))

	var b strings.Builder

	promptStyle := r.theme.FilterPrompt
	inputStyle := r.theme.FilterInput

	headerText := "Switch Repository"
	b.WriteString(r.theme.Text.Render(headerText))
	b.WriteString("\n\n")

	b.WriteString(promptStyle.Render(">"))
	b.WriteString(inputStyle.Render(r.input + "█"))
	b.WriteString("\n")
	b.WriteString(r.theme.Divider(overlayWidth - 4))
	b.WriteString("\n")

	if len(r.filtered) == 0 {
		b.WriteString(r.theme.TextMuted.Render("  No matching repositories"))
	} else {
		maxVisible := overlayHeight - 7
		visibleStart := 0
		visibleEnd := min(len(r.filtered), maxVisible)

		if r.cursor >= visibleEnd {
			visibleStart = r.cursor - maxVisible + 1
			visibleEnd = r.cursor + 1
		}

		for i := visibleStart; i < visibleEnd; i++ {
			repo := r.filtered[i]
			isSelected := i == r.cursor
			isCurrent := repo.Repository == r.currentRepo

			prefix := r.theme.ItemPrefix(isSelected)

			displayName := repo.Repository
			if repo.Alias != "" {
				displayName = repo.Alias + " (" + repo.Repository + ")"
			}

			if isCurrent {
				displayName += " ⭐"
			}

			var line string
			if isSelected {
				line = r.theme.Selected.Render(prefix + displayName)
			} else {
				line = r.theme.Text.Render(prefix + displayName)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(r.theme.TextMuted.Render("[enter] switch [esc] cancel"))

	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth-4).
		Height(overlayHeight-2).
		Padding(1, 2).
		Render(b.String())

	overlayBox := r.theme.BorderActive.
		Render(overlayContent)

	return lipgloss.Place(
		r.width,
		r.height,
		lipgloss.Center,
		lipgloss.Top,
		overlayBox,
	)
}
