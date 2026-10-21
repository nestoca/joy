package schema

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nestoca/joy/internal"
)

const narrowWidth = 100

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	branchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true) // bright blue ▶/▼
	leafStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	borderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// RunBrowser launches an interactive schema tree browser on the given IO streams.
func RunBrowser(streams internal.IO, root *Node) error {
	if root == nil {
		return fmt.Errorf("schema tree is nil")
	}
	m := newBrowserModel(root)
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
	}
	if streams.In != nil {
		opts = append(opts, tea.WithInput(streams.In))
	}
	if streams.Out != nil {
		opts = append(opts, tea.WithOutput(streams.Out))
	}
	_, err := tea.NewProgram(m, opts...).Run()
	return err
}

type flatRow struct {
	node  *Node
	depth int
}

type browserModel struct {
	root *Node

	rows      []flatRow
	cursor    int
	offset    int
	width     int
	height    int
	filter    string
	filtering bool
	err       error
}

func newBrowserModel(root *Node) browserModel {
	m := browserModel{root: root, width: 120, height: 30}
	m.refreshRows()
	return m
}

func (m browserModel) Init() tea.Cmd {
	return nil
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "right", "l", "enter":
			m.expandOrDrill()
		case "left", "h":
			m.collapseOrParent()
		case "/":
			m.filtering = true
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.refreshRows()
			}
		}
	}
	return m, nil
}

func (m browserModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.refreshRows()
	case "enter":
		m.filtering = false
		m.refreshRows()
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.refreshRows()
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= 32 && r != 127 {
				m.filter += string(r)
				m.refreshRows()
			}
		}
	}
	return m, nil
}

func (m *browserModel) expandOrDrill() {
	n := m.selected()
	if n == nil {
		return
	}
	if n.CanExpand() && !n.IsExpanded() {
		n.Expand()
		m.refreshRows()
		return
	}
	if n.CanExpand() && n.IsExpanded() && len(n.Children) > 0 {
		// Move to first child.
		for i, row := range m.rows {
			if row.node == n.Children[0] {
				m.cursor = i
				m.ensureVisible()
				return
			}
		}
	}
}

func (m *browserModel) collapseOrParent() {
	n := m.selected()
	if n == nil {
		return
	}
	if n.CanExpand() && n.IsExpanded() {
		n.Collapse()
		m.refreshRows()
		return
	}
	if parent := n.Parent(); parent != nil {
		for i, row := range m.rows {
			if row.node == parent {
				m.cursor = i
				m.ensureVisible()
				return
			}
		}
	}
}

func (m *browserModel) selected() *Node {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].node
}

func (m *browserModel) refreshRows() {
	prev := m.selected()
	m.rows = m.rows[:0]
	// Root is shown as the pane title; list its children (and deeper expanded nodes).
	if m.root.IsExpanded() {
		for _, child := range m.root.Children {
			m.walk(child, 0)
		}
	}
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if prev != nil {
		for i, row := range m.rows {
			if row.node == prev {
				m.cursor = i
				m.ensureVisible()
				return
			}
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	m.ensureVisible()
}

func (m *browserModel) walk(n *Node, depth int) {
	if n == nil {
		return
	}
	if m.matches(n) || hasMatchingDescendant(n, m.filter) {
		// When filtering, auto-expand ancestors of matches so paths are reachable.
		if m.filter != "" && n.CanExpand() && !n.IsExpanded() && hasMatchingDescendant(n, m.filter) {
			n.Expand()
		}
		if m.matches(n) || m.filter == "" {
			m.rows = append(m.rows, flatRow{node: n, depth: depth})
		} else if m.filter != "" && hasMatchingDescendant(n, m.filter) {
			// Keep ancestor visible while filtering so structure remains clear.
			m.rows = append(m.rows, flatRow{node: n, depth: depth})
		}
	}
	if n.IsExpanded() {
		for _, child := range n.Children {
			m.walk(child, depth+1)
		}
	}
}

func (m *browserModel) matches(n *Node) bool {
	if m.filter == "" {
		return true
	}
	q := strings.ToLower(m.filter)
	return strings.Contains(strings.ToLower(n.Name), q) ||
		strings.Contains(strings.ToLower(n.Path), q) ||
		strings.Contains(strings.ToLower(n.Type), q)
}

func hasMatchingDescendant(n *Node, filter string) bool {
	if filter == "" {
		return true
	}
	q := strings.ToLower(filter)
	var check func(*Node) bool
	check = func(node *Node) bool {
		if node == nil {
			return false
		}
		if strings.Contains(strings.ToLower(node.Name), q) ||
			strings.Contains(strings.ToLower(node.Path), q) ||
			strings.Contains(strings.ToLower(node.Type), q) {
			return true
		}
		if !node.CanExpand() {
			return false
		}
		wasExpanded := node.IsExpanded()
		node.Expand()
		if slices.ContainsFunc(node.Children, check) {
			if !wasExpanded {
				// Leave expanded so the user can navigate into matches.
			}
			return true
		}
		if !wasExpanded {
			node.Collapse()
		}
		return false
	}
	return check(n)
}

func (m *browserModel) ensureVisible() {
	treeHeight := m.treeViewportHeight()
	if treeHeight <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+treeHeight {
		m.offset = m.cursor - treeHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m browserModel) treeViewportHeight() int {
	help := 2
	filterLine := 0
	if m.filtering || m.filter != "" {
		filterLine = 1
	}
	if m.width < narrowWidth {
		// Stacked: tree gets roughly half minus chrome.
		return max(3, (m.height-help-filterLine)/2-2)
	}
	return max(3, m.height-help-filterLine-2)
}

func (m browserModel) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	tree := m.renderTree()
	detail := m.renderDetail()
	help := helpStyle.Render("↑↓/jk move  ←→/hl expand  / filter  q quit")
	filterLine := ""
	if m.filtering || m.filter != "" {
		prefix := "filter"
		if m.filtering {
			prefix = "/ filter"
		}
		filterLine = mutedStyle.Render(prefix+": ") + m.filter
		if m.filtering {
			filterLine += "█"
		}
		filterLine += "\n"
	}

	var body string
	if m.width < narrowWidth {
		body = lipgloss.JoinVertical(lipgloss.Left, tree, detail)
	} else {
		treeWidth := max(24, m.width/3)
		detailWidth := max(20, m.width-treeWidth-1)
		tree = borderStyle.Width(treeWidth - 2).Height(m.height - 4).Render(tree)
		detail = borderStyle.Width(detailWidth - 2).Height(m.height - 4).Render(detail)
		body = lipgloss.JoinHorizontal(lipgloss.Top, tree, detail)
	}
	return filterLine + body + "\n" + help
}

func (m browserModel) renderTree() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.root.Name))
	b.WriteByte('\n')

	height := m.treeViewportHeight()
	end := min(len(m.rows), m.offset+height)
	for i := m.offset; i < end; i++ {
		row := m.rows[i]
		indent := strings.Repeat("  ", row.depth)
		arrow, name := "· ", row.node.Name
		if row.node.CanExpand() {
			arrow = "▶ "
			if row.node.IsExpanded() {
				arrow = "▼ "
			}
		}
		var line string
		if i == m.cursor {
			line = selStyle.Render(indent + arrow + name)
		} else if row.node.CanExpand() {
			line = indent + branchStyle.Render(arrow) + name
		} else {
			line = indent + leafStyle.Render(arrow) + mutedStyle.Render(name)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m browserModel) renderDetail() string {
	n := m.selected()
	if n == nil {
		n = m.root
	}
	if n == nil {
		return mutedStyle.Render("no selection")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", mutedStyle.Render("path:"), n.Path)
	fmt.Fprintf(&b, "%s %s\n", mutedStyle.Render("type:"), n.Type)
	if n.Default != "" {
		fmt.Fprintf(&b, "%s %s\n", mutedStyle.Render("default:"), n.Default)
	}
	fmt.Fprintf(&b, "%s %v\n", mutedStyle.Render("optional:"), n.Optional)
	if n.Docs != "" {
		b.WriteByte('\n')
		b.WriteString(n.Docs)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
