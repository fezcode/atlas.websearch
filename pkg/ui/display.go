package ui

import (
	"fmt"
	"strings"

	"atlas.websearch/pkg/search"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
)

var (
	// Colors
	purple       = lipgloss.Color("#7D56F4")
	green        = lipgloss.Color("#04B575")
	gray         = lipgloss.Color("#888888")
	white        = lipgloss.Color("#FFFFFF")
	brightPurple = lipgloss.Color("#9D76FF")

	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(gray).
			Italic(true).
			Padding(0, 1)

	resultBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gray).
			Padding(0, 1).
			MarginBottom(1).
			Width(80)

	selectedBoxStyle = resultBoxStyle.Copy().
				BorderForeground(brightPurple).
				Background(lipgloss.Color("#2D2D2D"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)

	urlStyle = lipgloss.NewStyle().
			Foreground(green).
			Underline(true)

	snippetStyle = lipgloss.NewStyle().
			Foreground(white)

	helpStyle = lipgloss.NewStyle().
			Foreground(gray)
)

type model struct {
	results    []search.Result
	engineName string
	query      string
	cursor     int
	width      int
	height     int
	viewport   viewport.Model
	ready      bool
}

func initialModel(results []search.Result, engineName string, query string) model {
	return model{
		results:    results,
		engineName: engineName,
		query:      query,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		m.width = msg.Width
		m.height = msg.Height

		// Update content width for boxes
		newWidth := m.width - 4
		if newWidth > 100 {
			newWidth = 100
		}
		resultBoxStyle.Width(newWidth)
		selectedBoxStyle.Width(newWidth)

		// Re-render content with new widths
		m.viewport.SetContent(m.renderContent())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.syncViewport()
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.syncViewport()
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.results) {
				_ = browser.OpenURL(m.results[m.cursor].URL)
			}
		}

		// Always update content on keypress to reflect cursor change
		m.viewport.SetContent(m.renderContent())
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) syncViewport() {
	if !m.ready {
		return
	}

	linesBefore := 0
	for i := 0; i < m.cursor; i++ {
		content := m.renderItem(i)
		linesBefore += lipgloss.Height(content)
	}

	currentItemHeight := lipgloss.Height(m.renderItem(m.cursor))

	// Scroll up if cursor is above view
	if linesBefore < m.viewport.YOffset {
		m.viewport.SetYOffset(linesBefore)
	}

	// Scroll down if cursor is below view
	if linesBefore+currentItemHeight > m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(linesBefore + currentItemHeight - m.viewport.Height)
	}
}

func (m model) renderItem(index int) string {
	res := m.results[index]
	var content strings.Builder

	title := titleStyle.Render(res.Title)
	url := urlStyle.Render(res.URL)
	snippet := ""
	if res.Snippet != "" {
		snippet = "\n" + snippetStyle.Render(res.Snippet)
	}

	content.WriteString(fmt.Sprintf("%s\n%s%s", title, url, snippet))

	style := resultBoxStyle
	if m.cursor == index {
		style = selectedBoxStyle
	}

	return style.Render(content.String()) + "\n"
}

func (m model) renderContent() string {
	var s strings.Builder
	for i := range m.results {
		s.WriteString(m.renderItem(i))
	}
	return s.String()
}

func (m model) headerView() string {
	title := fmt.Sprintf(" ATLAS: %s | ENGINE: %s ", strings.ToUpper(m.query), strings.ToUpper(m.engineName))
	return headerStyle.Render(title) + "\n"
}

func (m model) footerView() string {
	help := helpStyle.Render("↑/↓: navigate • enter: open • q: quit")
	return "\n" + footerStyle.Render(help)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s%s%s", m.headerView(), m.viewport.View(), m.footerView())
}

// RenderResults starts the Bubble Tea UI with the given search results.

func RenderResults(resp *search.Response, engineName string, query string) error {
	if resp == nil || len(resp.Results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	p := tea.NewProgram(initialModel(resp.Results, engineName, query), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err

}
