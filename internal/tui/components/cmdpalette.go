package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Cloudsky01/gh-rivet/internal/tui/theme"
)

type Command struct {
	Name        string
	Aliases     []string
	Description string
	Action      func() tea.Cmd
}

type CmdPalette struct {
	active   bool
	input    string
	commands []Command
	filtered []Command
	cursor   int
	width    int
	height   int
	theme    *theme.Theme
}

func NewCmdPalette(t *theme.Theme) CmdPalette {
	return CmdPalette{
		theme:    t,
		commands: []Command{},
		filtered: []Command{},
	}
}

func (c *CmdPalette) SetCommands(cmds []Command) {
	c.commands = cmds
	c.applyFilter()
}

func (c *CmdPalette) SetSize(width, height int) {
	c.width = width
	c.height = height
}

func (c *CmdPalette) IsActive() bool {
	return c.active
}

func (c *CmdPalette) Open() {
	c.active = true
	c.input = ""
	c.cursor = 0
	c.applyFilter()
}

func (c *CmdPalette) Close() {
	c.active = false
	c.input = ""
	c.cursor = 0
}

func (c *CmdPalette) applyFilter() {
	if c.input == "" {
		c.filtered = c.commands
		return
	}

	matches := fuzzy.FindFrom(c.input, commandSource(c.commands))
	c.filtered = make([]Command, len(matches))
	for i, match := range matches {
		c.filtered[i] = c.commands[match.Index]
	}
	c.cursor = 0
}

type commandSource []Command

func (cs commandSource) String(i int) string {
	cmd := cs[i]
	return cmd.Name + " " + strings.Join(cmd.Aliases, " ")
}

func (cs commandSource) Len() int {
	return len(cs)
}

func (c *CmdPalette) Update(msg tea.Msg) (selectedCmd *Command, cmd tea.Cmd) {
	if !c.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.Close()
			return nil, nil
		case "enter":
			if c.cursor >= 0 && c.cursor < len(c.filtered) {
				selected := c.filtered[c.cursor]
				c.Close()
				return &selected, nil
			}
			c.Close()
			return nil, nil
		case "up", "ctrl+p":
			if c.cursor > 0 {
				c.cursor--
			}
			return nil, nil
		case "down", "ctrl+n":
			if c.cursor < len(c.filtered)-1 {
				c.cursor++
			}
			return nil, nil
		case "backspace":
			if len(c.input) > 0 {
				c.input = c.input[:len(c.input)-1]
				c.applyFilter()
			}
			return nil, nil
		case "tab":
			if c.cursor >= 0 && c.cursor < len(c.filtered) {
				c.input = c.filtered[c.cursor].Name
				c.applyFilter()
			}
			return nil, nil
		default:
			key := msg.String()
			if len(key) == 1 {
				c.input += key
				c.applyFilter()
			}
			return nil, nil
		}
	}
	return nil, nil
}

func (c *CmdPalette) View() string {
	if !c.active {
		return ""
	}

	overlayWidth := max(40, c.width*50/100)
	overlayHeight := max(10, min(20, c.height*50/100))

	var b strings.Builder

	promptStyle := c.theme.FilterPrompt
	inputStyle := c.theme.FilterInput

	b.WriteString(promptStyle.Render(":"))
	b.WriteString(inputStyle.Render(c.input + "█"))
	b.WriteString("\n")
	b.WriteString(c.theme.Divider(overlayWidth - 4))
	b.WriteString("\n")

	if len(c.filtered) == 0 {
		b.WriteString(c.theme.TextMuted.Render("  No matching commands"))
	} else {
		maxVisible := overlayHeight - 5
		visibleStart := 0
		visibleEnd := min(len(c.filtered), maxVisible)

		if c.cursor >= visibleEnd {
			visibleStart = c.cursor - maxVisible + 1
			visibleEnd = c.cursor + 1
		}

		for i := visibleStart; i < visibleEnd; i++ {
			cmd := c.filtered[i]
			isSelected := i == c.cursor

			prefix := c.theme.ItemPrefix(isSelected)
			var line string
			if isSelected {
				line = c.theme.Selected.Render(prefix + cmd.Name)
			} else {
				line = c.theme.Text.Render(prefix + cmd.Name)
			}
			b.WriteString(line)

			if cmd.Description != "" {
				desc := c.theme.TextDim.Render(" - " + cmd.Description)
				b.WriteString(desc)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(c.theme.TextMuted.Render("[tab] complete [enter] execute [esc] cancel"))

	overlayContent := lipgloss.NewStyle().
		Width(overlayWidth-4).
		Height(overlayHeight-2).
		Padding(1, 2).
		Render(b.String())

	overlayBox := c.theme.BorderActive.
		Render(overlayContent)

	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Top,
		overlayBox,
	)
}

func DefaultCommands(actions map[string]func() tea.Cmd) []Command {
	cmds := []Command{
		{Name: "quit", Aliases: []string{"q", "exit"}, Description: "Exit the application"},
		{Name: "refresh", Aliases: []string{"r"}, Description: "Refresh current view"},
		{Name: "search", Aliases: []string{"s", "find"}, Description: "Open global search"},
		{Name: "filter", Aliases: []string{"f"}, Description: "Filter current list"},
		{Name: "help", Aliases: []string{"h", "?"}, Description: "Show help"},
		{Name: "pin", Aliases: []string{"p"}, Description: "Pin/unpin selected workflow"},
		{Name: "open", Aliases: []string{"o", "web", "browser"}, Description: "Open in browser"},
		{Name: "sidebar", Aliases: []string{"1"}, Description: "Toggle sidebar"},
		{Name: "runs", Aliases: []string{"r"}, Description: "Toggle runs panel"},
		{Name: "groups", Aliases: []string{"g"}, Description: "Go to groups view"},
		{Name: "back", Aliases: []string{"b"}, Description: "Go back"},
	}

	for i := range cmds {
		if action, ok := actions[cmds[i].Name]; ok {
			cmds[i].Action = action
		}
	}

	return cmds
}
