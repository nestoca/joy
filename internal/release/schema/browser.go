package schema

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	var opts []tea.ProgramOption
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

	case tea.KeyPressMsg:
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

func (m browserModel) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.refreshRows()
	case "enter":
		m.filtering = false
		m.refreshRows()
	case "backspace":
		// Remove the final rune, not a single byte, so multi-byte characters
		// (e.g. "é") do not leave invalid UTF-8 in the filter.
		if runes := []rune(m.filter); len(runes) > 0 {
			m.filter = string(runes[:len(runes)-1])
			m.refreshRows()
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		if msg.Text != "" {
			m.filter += msg.Text
			m.refreshRows()
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
	// Memoize subtree-match results for the duration of this refresh so each
	// node's descendants are inspected at most once, even though walk and the
	// auto-expand logic both need the answer.
	memo := make(map[*Node]bool)
	// Root is shown as the pane title; list its children (and deeper expanded nodes).
	if m.root.IsExpanded() {
		for _, child := range m.root.Children {
			m.walk(child, 0, memo)
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

func (m *browserModel) walk(n *Node, depth int, memo map[*Node]bool) {
	if n == nil {
		return
	}
	selfMatch := m.matches(n)
	// Compute the subtree-match result once and reuse it for both the
	// auto-expand decision and the visibility check below.
	visible := selfMatch
	if !selfMatch && m.filter != "" {
		visible = hasMatchingDescendant(n, m.filter, 0, memo)
	}
	if visible {
		// When filtering, auto-expand ancestors of matches so paths are reachable.
		if m.filter != "" && n.CanExpand() && !n.IsExpanded() {
			n.Expand()
		}
		m.rows = append(m.rows, flatRow{node: n, depth: depth})
	}
	if n.IsExpanded() {
		for _, child := range n.Children {
			m.walk(child, depth+1, memo)
		}
	}
}

func (m *browserModel) matches(n *Node) bool {
	if m.filter == "" {
		return true
	}
	return matchesFilter(n, m.filter)
}

func matchesFilter(n *Node, filter string) bool {
	q := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(n.Name), q) ||
		strings.Contains(strings.ToLower(n.Path), q) ||
		strings.Contains(strings.ToLower(n.Type), q)
}

// maxFilterDepth bounds how deep hasMatchingDescendant will recurse so that
// deeply nested or self-referencing schemas cannot cause unbounded work.
const maxFilterDepth = 64

// hasMatchingDescendant reports whether n or any of its descendants matches the
// filter. Results are memoized through memo for the duration of a single
// refresh, and recursion stops at maxFilterDepth. As a side effect, nodes along
// a path to a match are left expanded so the user can navigate into them.
func hasMatchingDescendant(n *Node, filter string, depth int, memo map[*Node]bool) bool {
	if filter == "" {
		return true
	}
	if n == nil || depth > maxFilterDepth {
		return false
	}
	if cached, ok := memo[n]; ok {
		return cached
	}
	result := matchesFilter(n, filter)
	if !result && n.CanExpand() {
		wasExpanded := n.IsExpanded()
		n.Expand()
		for _, child := range n.Children {
			if hasMatchingDescendant(child, filter, depth+1, memo) {
				result = true
				break
			}
		}
		// Restore the collapsed state only when nothing matched; a match leaves
		// the node expanded so the user can navigate into it.
		if !result && !wasExpanded {
			n.Collapse()
		}
	}
	memo[n] = result
	return result
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

func (m browserModel) View() tea.View {
	if m.err != nil {
		v := tea.NewView(m.err.Error())
		v.AltScreen = true
		return v
	}
	tree := m.renderTree()
	detail := m.renderDetail()
	help := helpStyle.Render("↑↓/jk move  ←→/hl expand  / filter  q quit")
	filterLine := ""
	filterLines := 0
	if m.filtering || m.filter != "" {
		filterLines = 1
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

	// Budget the vertical space so the rendered view never exceeds the
	// terminal height. Without a hard cap, a long detail pane (e.g. a node
	// with lots of docs) overflows and the alt screen scrolls, pushing the
	// title and filter input off the top. One line is reserved for the help
	// footer.
	const helpLines = 1

	var body string
	if m.width < narrowWidth {
		treeLines := m.treeViewportHeight() + 1 // tree rows plus the title line
		detailLines := max(1, m.height-helpLines-filterLines-treeLines)
		tree = lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(treeLines).Render(tree)
		detail = lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(detailLines).Render(detail)
		body = lipgloss.JoinVertical(lipgloss.Left, tree, detail)
	} else {
		treeWidth := max(24, m.width/3)
		detailWidth := max(20, m.width-treeWidth-1)
		innerHeight := m.treeViewportHeight() + 1 // tree rows plus the title line
		tree = borderStyle.Width(treeWidth - 2).Height(innerHeight).MaxWidth(treeWidth).MaxHeight(innerHeight + 2).Render(tree)
		detail = borderStyle.Width(detailWidth - 2).Height(innerHeight).MaxWidth(detailWidth).MaxHeight(innerHeight + 2).Render(detail)
		body = lipgloss.JoinHorizontal(lipgloss.Top, tree, detail)
	}
	v := tea.NewView(filterLine + body + "\n" + help)
	v.AltScreen = true
	return v
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
