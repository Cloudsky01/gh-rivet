package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

// PinnedItem represents a pinned workflow in the sidebar
type PinnedItem struct {
	WorkflowName string
	GroupName    string
	GroupID      string
	Data         interface{} // Reference to the group for actions
}

// Sidebar is the pinned workflows sidebar component
type Sidebar struct {
	items         []PinnedItem
	filteredItems []PinnedItem
	cursor        int
	filterInput   string
	filterActive  bool
	width         int
	height        int
	visible       bool
	focused       bool
	theme         *theme.Theme
}

// NewSidebar creates a new sidebar component
func NewSidebar(t *theme.Theme) Sidebar {
	return Sidebar{
		items:         []PinnedItem{},
		filteredItems: []PinnedItem{},
		visible:       true,
		theme:         t,
	}
}

// SetItems sets the pinned items
func (s *Sidebar) SetItems(items []PinnedItem) {
	s.items = items
	s.applyFilter()
	if s.cursor >= len(s.filteredItems) {
		s.cursor = max(0, len(s.filteredItems)-1)
	}
}

// Items returns all items
func (s *Sidebar) Items() []PinnedItem {
	return s.items
}

// SetSize sets the dimensions
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetFocused sets the focus state
func (s *Sidebar) SetFocused(focused bool) {
	s.focused = focused
}

// IsFocused returns the focus state
func (s *Sidebar) IsFocused() bool {
	return s.focused
}

// SetVisible sets visibility
func (s *Sidebar) SetVisible(visible bool) {
	s.visible = visible
}

// IsVisible returns visibility
func (s *Sidebar) IsVisible() bool {
	return s.visible
}

// Toggle toggles visibility
func (s *Sidebar) Toggle() {
	s.visible = !s.visible
}

// Cursor returns current cursor position
func (s *Sidebar) Cursor() int {
	return s.cursor
}

// SelectedItem returns the selected pinned item
func (s *Sidebar) SelectedItem() *PinnedItem {
	if s.cursor >= 0 && s.cursor < len(s.filteredItems) {
		return &s.filteredItems[s.cursor]
	}
	return nil
}

// IsFiltering returns whether filter mode is active
func (s *Sidebar) IsFiltering() bool {
	return s.filterActive
}

// HasFilter returns whether a filter is applied
func (s *Sidebar) HasFilter() bool {
	return s.filterInput != ""
}

// ClearFilter clears the filter
func (s *Sidebar) ClearFilter() {
	s.filterInput = ""
	s.filterActive = false
	s.applyFilter()
}

// StartFilter begins filter input mode
func (s *Sidebar) StartFilter() {
	s.filterActive = true
	s.filterInput = ""
}

func (s *Sidebar) applyFilter() {
	if s.filterInput == "" {
		s.filteredItems = s.items
		return
	}

	matches := fuzzy.FindFrom(s.filterInput, pinnedItemSource(s.items))
	s.filteredItems = make([]PinnedItem, len(matches))
	for i, match := range matches {
		s.filteredItems[i] = s.items[match.Index]
	}
}

type pinnedItemSource []PinnedItem

func (p pinnedItemSource) String(i int) string {
	return p[i].WorkflowName
}

func (p pinnedItemSource) Len() int {
	return len(p)
}

// Update handles input
func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKey(msg)
	}
	return nil
}

func (s *Sidebar) handleKey(msg tea.KeyMsg) tea.Cmd {
	if s.filterActive {
		switch msg.String() {
		case "enter":
			s.filterActive = false
			s.cursor = 0
			return nil
		case "esc":
			s.filterActive = false
			s.filterInput = ""
			s.applyFilter()
			return nil
		case "backspace":
			if len(s.filterInput) > 0 {
				s.filterInput = s.filterInput[:len(s.filterInput)-1]
				s.applyFilter()
				s.cursor = 0
			}
			return nil
		default:
			key := msg.String()
			if len(key) == 1 {
				s.filterInput += key
				s.applyFilter()
				s.cursor = 0
			}
			return nil
		}
	}

	switch msg.String() {
	case "/":
		s.StartFilter()
		return nil
	case "j", "down":
		if s.cursor < len(s.filteredItems)-1 {
			s.cursor++
		}
		return nil
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
		return nil
	case "esc":
		if s.filterInput != "" {
			s.ClearFilter()
		}
		return nil
	}

	return nil
}

// View renders the sidebar
func (s *Sidebar) View() string {
	if !s.visible {
		return ""
	}

	var b strings.Builder

	// Header
	titleStyle := s.theme.Title
	if s.focused {
		titleStyle = s.theme.TitleActive
	}
	title := s.theme.Icons.Pin + " Pinned"
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(s.theme.Divider(s.width - 2))
	b.WriteString("\n")

	// Filter
	headerHeight := 2
	if s.filterActive {
		cursor := "█"
		filterLine := s.theme.FilterPrompt.Render(s.theme.Icons.Filter+" ") +
			s.theme.FilterInput.Render(s.filterInput+cursor)
		b.WriteString(filterLine)
		b.WriteString("\n")
		headerHeight++
	} else if s.filterInput != "" {
		filterInfo := s.theme.TextMuted.Render(
			fmt.Sprintf("%s %q", s.theme.Icons.Search, s.filterInput))
		b.WriteString(filterInfo)
		b.WriteString("\n")
		headerHeight++
	}

	// Items
	footerHeight := 1
	availableHeight := s.height - headerHeight - footerHeight
	itemHeight := 3 // workflow name + group name + spacing
	visibleCount := max(1, availableHeight/itemHeight)

	if len(s.filteredItems) == 0 {
		emptyMsg := "No pinned workflows"
		if s.filterInput != "" {
			emptyMsg = "No matches"
		}
		b.WriteString(s.theme.TextMuted.Render("  " + emptyMsg))
		b.WriteString("\n")
	} else {
		visibleStart, visibleEnd := s.calculateVisibleWindow(visibleCount)

		// Scroll indicator
		if len(s.filteredItems) > visibleCount {
			scrollInfo := s.theme.TextMuted.Render(
				fmt.Sprintf("  (%d-%d of %d)", visibleStart+1, visibleEnd, len(s.filteredItems)))
			b.WriteString(scrollInfo)
			b.WriteString("\n")
		}

		for i := visibleStart; i < visibleEnd; i++ {
			item := s.filteredItems[i]
			isSelected := i == s.cursor && s.focused

			// Workflow name
			prefix := s.theme.ItemPrefix(isSelected)
			workflowName := item.WorkflowName
			maxWidth := s.width - 6
			if len(workflowName) > maxWidth {
				workflowName = workflowName[:maxWidth-3] + "..."
			}

			var workflowLine string
			if isSelected {
				workflowLine = s.theme.Selected.Render(prefix + workflowName)
			} else {
				workflowLine = s.theme.Text.Render(prefix + workflowName)
			}
			b.WriteString(workflowLine)
			b.WriteString("\n")

			// Group name
			groupName := item.GroupName
			if len(groupName) > maxWidth-2 {
				groupName = groupName[:maxWidth-5] + "..."
			}
			groupLine := s.theme.TextDim.Render("    " + groupName)
			b.WriteString(groupLine)
			b.WriteString("\n")

			if i < visibleEnd-1 {
				b.WriteString("\n")
			}
		}
	}

	// Pad remaining
	contentLines := strings.Count(b.String(), "\n")
	remaining := s.height - contentLines - 1
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	// Help hints
	var hints string
	if s.filterActive {
		hints = "[enter] apply [esc] cancel"
	} else {
		hints = "[/] filter [1] toggle"
	}
	b.WriteString(s.theme.TextMuted.Render(hints))

	return lipgloss.NewStyle().
		Width(s.width).
		Height(s.height).
		Render(b.String())
}

func (s *Sidebar) calculateVisibleWindow(maxVisible int) (start, end int) {
	total := len(s.filteredItems)
	if total <= maxVisible {
		return 0, total
	}

	start = max(0, s.cursor-maxVisible/2)
	end = min(total, start+maxVisible)

	if end == total && end-start < maxVisible {
		start = max(0, end-maxVisible)
	}

	return start, end
}
