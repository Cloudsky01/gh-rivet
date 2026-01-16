package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

// SearchResult represents a search result
type SearchResult struct {
	Type         string   // "group" or "workflow"
	Name         string   // Display name
	Description  string   // Additional info
	GroupPath    []string // Path to parent groups
	WorkflowName string   // Actual workflow filename (for workflows)
	Data         interface{}
}

// SearchFunc is a function that returns search results for a query
type SearchFunc func(query string) []SearchResult

// Search is a global search overlay component
type Search struct {
	active     bool
	input      string
	results    []SearchResult
	cursor     int
	width      int
	height     int
	theme      *theme.Theme
	searchFunc SearchFunc
}

// NewSearch creates a new search component
func NewSearch(t *theme.Theme) Search {
	return Search{
		theme:   t,
		results: []SearchResult{},
	}
}

// SetSearchFunc sets the function used to perform searches
func (s *Search) SetSearchFunc(fn SearchFunc) {
	s.searchFunc = fn
}

// SetSize sets dimensions
func (s *Search) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// IsActive returns whether search is active
func (s *Search) IsActive() bool {
	return s.active
}

// Open opens the search overlay
func (s *Search) Open() {
	s.active = true
	s.input = ""
	s.results = nil
	s.cursor = 0
}

// Close closes the search overlay
func (s *Search) Close() {
	s.active = false
	s.input = ""
	s.results = nil
	s.cursor = 0
}

// SelectedResult returns the currently selected result
func (s *Search) SelectedResult() *SearchResult {
	if s.cursor >= 0 && s.cursor < len(s.results) {
		return &s.results[s.cursor]
	}
	return nil
}

func (s *Search) doSearch() {
	if s.searchFunc == nil || s.input == "" {
		s.results = nil
		return
	}
	s.results = s.searchFunc(s.input)
	s.cursor = 0
}

// Update handles input
func (s *Search) Update(msg tea.Msg) (selected *SearchResult, cmd tea.Cmd) {
	if !s.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			s.Close()
			return nil, nil
		case "enter":
			if result := s.SelectedResult(); result != nil {
				selected := *result
				s.Close()
				return &selected, nil
			}
			return nil, nil
		case "up", "ctrl+p":
			if s.cursor > 0 {
				s.cursor--
			}
			return nil, nil
		case "down", "ctrl+n":
			if s.cursor < len(s.results)-1 {
				s.cursor++
			}
			return nil, nil
		case "backspace":
			if len(s.input) > 0 {
				s.input = s.input[:len(s.input)-1]
				s.doSearch()
			}
			return nil, nil
		default:
			key := msg.String()
			if len(key) == 1 {
				s.input += key
				s.doSearch()
			}
			return nil, nil
		}
	}

	return nil, nil
}

// View renders the search overlay
func (s *Search) View() string {
	if !s.active {
		return ""
	}

	// Calculate overlay dimensions (centered, 60% width, 70% height)
	overlayWidth := max(50, s.width*60/100)
	overlayHeight := max(15, s.height*70/100)

	var b strings.Builder

	// Title
	title := s.theme.Title.Render("Global Search")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(s.theme.Divider(overlayWidth - 4))
	b.WriteString("\n\n")

	// Search input
	cursor := "█"
	inputText := s.input + cursor
	if s.input == "" {
		inputText = s.theme.TextMuted.Render("Type to search groups and workflows...") + cursor
	}

	searchIcon := s.theme.FilterPrompt.Render(s.theme.Icons.Search + " ")
	b.WriteString(searchIcon)
	b.WriteString(s.theme.FilterInput.Render(inputText))
	b.WriteString("\n\n")

	// Results
	if s.input == "" {
		hintText := s.theme.TextMuted.Render("  Start typing to search through all groups and workflows")
		b.WriteString(hintText)
	} else if len(s.results) == 0 {
		noResultsText := s.theme.TextMuted.Render("  No results found")
		b.WriteString(noResultsText)
	} else {
		// Show results count
		countText := s.theme.TextMuted.Render(fmt.Sprintf("  %d results", len(s.results)))
		b.WriteString(countText)
		b.WriteString("\n\n")

		// Calculate how many results we can show
		linesPerResult := 2
		maxResults := max(1, (overlayHeight-10)/linesPerResult)

		// Adjust visible window based on selection
		visibleStart := 0
		visibleEnd := min(len(s.results), maxResults)

		if s.cursor >= visibleEnd {
			visibleStart = s.cursor - maxResults + 1
			visibleEnd = s.cursor + 1
		}

		for i := visibleStart; i < visibleEnd; i++ {
			result := s.results[i]
			isSelected := i == s.cursor

			// Prefix and styling based on selection
			prefix := s.theme.ItemPrefix(isSelected)

			// Icon based on type
			icon := s.theme.Icons.Folder
			if result.Type == "workflow" {
				icon = s.theme.Icons.Workflow
			}

			// Truncate name if needed
			name := result.Name
			maxNameWidth := overlayWidth - 15
			if len(name) > maxNameWidth {
				name = name[:maxNameWidth-3] + "..."
			}

			var nameLine string
			if isSelected {
				nameLine = s.theme.Selected.Render(fmt.Sprintf("%s%s %s", prefix, icon, name))
			} else {
				nameLine = s.theme.Text.Render(fmt.Sprintf("%s%s %s", prefix, icon, name))
			}
			b.WriteString(nameLine)
			b.WriteString("\n")

			// Show path
			pathText := formatPath(result.GroupPath)
			if result.Type == "workflow" && result.Description != result.Name {
				pathText = pathText + " / " + result.Description
			}
			maxPathWidth := overlayWidth - 10
			if len(pathText) > maxPathWidth {
				pathText = pathText[:maxPathWidth-3] + "..."
			}

			pathLine := s.theme.TextDim.Render(fmt.Sprintf("     %s", pathText))
			b.WriteString(pathLine)
			b.WriteString("\n")
		}

		// Show scroll indicator if needed
		if len(s.results) > maxResults {
			scrollInfo := s.theme.TextMuted.Render(
				fmt.Sprintf("\n  (%d-%d of %d)", visibleStart+1, visibleEnd, len(s.results)))
			b.WriteString(scrollInfo)
		}
	}

	// Help hint at bottom
	b.WriteString("\n\n")
	hint := s.theme.TextMuted.Render("[↑/↓] navigate | [enter] select | [esc] close")
	b.WriteString(hint)

	// Create the overlay box
	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth-4).
		Height(overlayHeight-2).
		Padding(1, 2).
		Render(b.String())

	overlayBox := s.theme.BorderActive.
		Background(s.theme.Colors.BgSecondary).
		Render(overlayContent)

	// Center the overlay
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		overlayBox,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
	)
}

func formatPath(path []string) string {
	if len(path) == 0 {
		return "(root)"
	}
	return strings.Join(path, " > ")
}

// FuzzySearchItems performs fuzzy search on a list of SearchResults
func FuzzySearchItems(items []SearchResult, query string) []SearchResult {
	if query == "" {
		return items
	}

	matches := fuzzy.FindFrom(query, searchResultSource(items))
	results := make([]SearchResult, len(matches))
	for i, match := range matches {
		results[i] = items[match.Index]
	}
	return results
}

type searchResultSource []SearchResult

func (s searchResultSource) String(i int) string {
	return s[i].Name
}

func (s searchResultSource) Len() int {
	return len(s)
}
