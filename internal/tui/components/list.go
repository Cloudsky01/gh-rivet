// Package components provides reusable TUI components built on Bubble Tea.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

// ListItem represents an item in the list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Data        interface{} // Arbitrary data attached to the item
}

// FilterValue implements fuzzy.Source
func (i ListItem) FilterValue() string {
	return i.Title
}

// List is a filterable, navigable list component
type List struct {
	items         []ListItem
	filteredItems []ListItem
	cursor        int
	filterInput   string
	filterActive  bool
	width         int
	height        int
	title         string
	focused       bool
	theme         *theme.Theme
}

// NewList creates a new list component
func NewList(t *theme.Theme, title string) List {
	return List{
		items:         []ListItem{},
		filteredItems: []ListItem{},
		cursor:        0,
		title:         title,
		theme:         t,
	}
}

// SetItems sets the list items
func (l *List) SetItems(items []ListItem) {
	l.items = items
	l.applyFilter()
	// Reset cursor if out of bounds
	if l.cursor >= len(l.filteredItems) {
		l.cursor = max(0, len(l.filteredItems)-1)
	}
}

// Items returns all items
func (l *List) Items() []ListItem {
	return l.items
}

// FilteredItems returns the currently visible (filtered) items
func (l *List) FilteredItems() []ListItem {
	return l.filteredItems
}

// SetSize sets the dimensions
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetFocused sets the focus state
func (l *List) SetFocused(focused bool) {
	l.focused = focused
}

// IsFocused returns the focus state
func (l *List) IsFocused() bool {
	return l.focused
}

// SetTitle sets the list title
func (l *List) SetTitle(title string) {
	l.title = title
}

// Cursor returns the current cursor position
func (l *List) Cursor() int {
	return l.cursor
}

// SetCursor sets the cursor position
func (l *List) SetCursor(pos int) {
	if pos >= 0 && pos < len(l.filteredItems) {
		l.cursor = pos
	}
}

// SelectedItem returns the currently selected item, or nil if none
func (l *List) SelectedItem() *ListItem {
	if l.cursor >= 0 && l.cursor < len(l.filteredItems) {
		return &l.filteredItems[l.cursor]
	}
	return nil
}

// IsFiltering returns whether filter mode is active
func (l *List) IsFiltering() bool {
	return l.filterActive
}

// HasFilter returns whether a filter is applied
func (l *List) HasFilter() bool {
	return l.filterInput != ""
}

// FilterInput returns the current filter text
func (l *List) FilterInput() string {
	return l.filterInput
}

// ClearFilter clears the filter
func (l *List) ClearFilter() {
	l.filterInput = ""
	l.filterActive = false
	l.applyFilter()
}

// StartFilter begins filter input mode
func (l *List) StartFilter() {
	l.filterActive = true
	l.filterInput = ""
}

// applyFilter filters the items based on current filter input
func (l *List) applyFilter() {
	if l.filterInput == "" {
		l.filteredItems = l.items
		return
	}

	// Use fuzzy matching
	matches := fuzzy.FindFrom(l.filterInput, listItemSource(l.items))
	l.filteredItems = make([]ListItem, len(matches))
	for i, match := range matches {
		l.filteredItems[i] = l.items[match.Index]
	}
}

// listItemSource implements fuzzy.Source for ListItems
type listItemSource []ListItem

func (s listItemSource) String(i int) string {
	return s[i].Title
}

func (s listItemSource) Len() int {
	return len(s)
}

// Update handles input messages
func (l *List) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return l.handleKey(msg)
	}
	return nil
}

func (l *List) handleKey(msg tea.KeyMsg) tea.Cmd {
	if l.filterActive {
		switch msg.String() {
		case "enter":
			l.filterActive = false
			return nil
		case "esc":
			l.filterActive = false
			l.filterInput = ""
			l.applyFilter()
			l.cursor = 0
			return nil
		case "ctrl+n", "down":
			l.moveDown()
			return nil
		case "ctrl+p", "up":
			l.moveUp()
			return nil
		case "backspace":
			if len(l.filterInput) > 0 {
				l.filterInput = l.filterInput[:len(l.filterInput)-1]
				l.applyFilter()
				if l.cursor >= len(l.filteredItems) {
					l.cursor = max(0, len(l.filteredItems)-1)
				}
			}
			return nil
		default:
			key := msg.String()
			if len(key) == 1 {
				l.filterInput += key
				l.applyFilter()
				if l.cursor >= len(l.filteredItems) {
					l.cursor = max(0, len(l.filteredItems)-1)
				}
			}
			return nil
		}
	}

	switch msg.String() {
	case "/":
		l.StartFilter()
		return nil
	case "j", "down":
		l.moveDown()
		return nil
	case "k", "up":
		l.moveUp()
		return nil
	case "g":
		l.cursor = 0
		return nil
	case "G":
		l.cursor = max(0, len(l.filteredItems)-1)
		return nil
	case "esc":
		if l.filterInput != "" {
			l.ClearFilter()
		}
		return nil
	case "n":
		if l.filterInput != "" {
			l.moveDown()
		}
		return nil
	case "N":
		if l.filterInput != "" {
			l.moveUp()
		}
		return nil
	}

	return nil
}

func (l *List) moveDown() {
	if l.cursor < len(l.filteredItems)-1 {
		l.cursor++
	}
}

func (l *List) moveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

// View renders the list
func (l *List) View() string {
	var b strings.Builder

	// Calculate available height for items
	headerHeight := 2 // title + divider
	if l.filterActive || l.filterInput != "" {
		headerHeight += 1 // filter line
	}
	footerHeight := 1 // help hints
	availableHeight := l.height - headerHeight - footerHeight

	// Render header with title
	titleStyle := l.theme.Title
	if l.focused {
		titleStyle = l.theme.TitleActive
	}
	b.WriteString(titleStyle.Render(l.title))
	b.WriteString("\n")
	b.WriteString(l.theme.Divider(l.width - 2))
	b.WriteString("\n")

	// Render filter input if active
	if l.filterActive {
		cursor := "█"
		filterLine := l.theme.FilterPrompt.Render(l.theme.Icons.Filter+" ") +
			l.theme.FilterInput.Render(l.filterInput+cursor)
		b.WriteString(filterLine)
		b.WriteString("\n")
	} else if l.filterInput != "" {
		filterInfo := l.theme.TextMuted.Render(
			fmt.Sprintf("%s Filtered: %q [esc] clear", l.theme.Icons.Search, l.filterInput))
		b.WriteString(filterInfo)
		b.WriteString("\n")
	}

	// Calculate visible window
	visibleCount := max(1, availableHeight/2) // 2 lines per item (title + desc)
	visibleStart, visibleEnd := l.calculateVisibleWindow(visibleCount)

	// Render items
	if len(l.filteredItems) == 0 {
		emptyMsg := "No items"
		if l.filterInput != "" {
			emptyMsg = "No matches"
		}
		b.WriteString(l.theme.TextMuted.Render("  " + emptyMsg))
		b.WriteString("\n")
	} else {
		// Scroll indicator if needed
		if len(l.filteredItems) > visibleCount {
			scrollInfo := l.theme.TextMuted.Render(
				fmt.Sprintf("  (%d-%d of %d)", visibleStart+1, visibleEnd, len(l.filteredItems)))
			b.WriteString(scrollInfo)
			b.WriteString("\n")
		}

		for i := visibleStart; i < visibleEnd; i++ {
			item := l.filteredItems[i]
			isSelected := i == l.cursor && l.focused

			// Prefix
			prefix := l.theme.ItemPrefix(isSelected)

			// Title with icon
			titleText := item.Title
			if item.Icon != "" {
				titleText = item.Icon + " " + titleText
			}

			// Truncate if needed
			maxWidth := l.width - 6
			if len(titleText) > maxWidth {
				titleText = titleText[:maxWidth-3] + "..."
			}

			// Style based on selection
			var titleLine string
			if isSelected {
				titleLine = l.theme.Selected.Render(prefix + titleText)
			} else {
				titleLine = l.theme.Text.Render(prefix + titleText)
			}
			b.WriteString(titleLine)
			b.WriteString("\n")

			// Description
			if item.Description != "" {
				desc := item.Description
				if len(desc) > maxWidth-2 {
					desc = desc[:maxWidth-5] + "..."
				}
				descLine := l.theme.TextDim.Render("    " + desc)
				b.WriteString(descLine)
				b.WriteString("\n")
			}
		}
	}

	// Pad remaining height
	contentLines := strings.Count(b.String(), "\n")
	remaining := l.height - contentLines - 1
	if remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	var hints string
	if l.filterActive {
		hints = "[↑/↓] navigate [enter] done [esc] clear"
	} else if l.filterInput != "" {
		hints = "[n/N] next/prev [esc] clear"
	} else {
		hints = "[j/k] nav [/] filter"
	}
	b.WriteString(l.theme.TextMuted.Render(hints))

	return lipgloss.NewStyle().
		Width(l.width).
		Height(l.height).
		Render(b.String())
}

func (l *List) calculateVisibleWindow(maxVisible int) (start, end int) {
	total := len(l.filteredItems)
	if total <= maxVisible {
		return 0, total
	}

	// Center cursor in viewport
	start = max(0, l.cursor-maxVisible/2)
	end = min(total, start+maxVisible)

	// Adjust if at the end
	if end == total && end-start < maxVisible {
		start = max(0, end-maxVisible)
	}

	return start, end
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
