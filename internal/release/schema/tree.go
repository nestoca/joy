package schema

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
)

// Kind classifies a schema tree node.
type Kind int

const (
	KindLeaf Kind = iota
	KindStruct
	KindList
)

func (k Kind) String() string {
	switch k {
	case KindStruct:
		return "struct"
	case KindList:
		return "list"
	default:
		return "leaf"
	}
}

// Node is a navigable schema field derived from a cue.Value.
type Node struct {
	Name     string
	Path     string
	Kind     Kind
	Type     string
	Optional bool
	Default  string
	Docs     string
	Children []*Node

	value    cue.Value
	expanded bool
	hasKids  bool
	parent   *Node
}

// BuildTree creates a navigable tree rooted at value.
// Children of non-leaf nodes are populated lazily via Expand.
func BuildTree(value cue.Value, rootName string) (*Node, error) {
	if !value.Exists() {
		return nil, fmt.Errorf("schema value does not exist")
	}
	if err := value.Err(); err != nil {
		return nil, fmt.Errorf("invalid schema value: %w", err)
	}
	if rootName == "" {
		rootName = "#values"
	}
	root := newNode(rootName, rootName, value, false, nil)
	root.Expand()
	return root, nil
}

func (n *Node) Expand() {
	if n == nil || n.expanded || !n.hasKids {
		return
	}
	n.expanded = true
	n.Children = buildChildren(n)
}

func (n *Node) Collapse() {
	if n != nil {
		n.expanded = false
	}
}

func (n *Node) IsExpanded() bool { return n != nil && n.expanded }
func (n *Node) CanExpand() bool  { return n != nil && n.hasKids }
func (n *Node) Parent() *Node {
	if n == nil {
		return nil
	}
	return n.parent
}

func newNode(name, path string, value cue.Value, optional bool, parent *Node) *Node {
	kind, hasKids := classify(value)
	return &Node{
		Name:     name,
		Path:     path,
		Kind:     kind,
		Type:     typeString(value),
		Optional: optional,
		Default:  defaultString(value),
		Docs:     docsString(value),
		value:    value,
		hasKids:  hasKids,
		parent:   parent,
	}
}

func classify(value cue.Value) (Kind, bool) {
	if fields := structFields(value); len(fields) > 0 {
		return KindStruct, true
	}
	if value.IncompleteKind()&cue.ListKind != 0 {
		return KindList, len(listFields(value)) > 0
	}
	return KindLeaf, false
}

// fieldOpts includes optional fields and [string]/[...T] pattern constraints.
var fieldOpts = []cue.Option{
	cue.Optional(true),
	cue.Definitions(false),
	cue.Patterns(true),
}

type field struct {
	name     string
	value    cue.Value
	optional bool
}

// structFields returns browsable struct fields.
// Handles defaults (false | *#T), optional refs (api?: #api), pattern maps
// ([string]: T), disjunctions that include a struct branch, and plain structs.
//
// Fields are merged across every struct candidate so that a disjunction such as
// `*{port: int} | {socket: string}` exposes both branches rather than only the
// default one. The first candidate to define a field wins, preserving the
// default branch's value and optionality when branches overlap.
func structFields(value cue.Value) []field {
	var fields []field
	seen := make(map[string]struct{})
	for _, candidate := range structCandidates(value) {
		iter, err := candidate.Fields(fieldOpts...)
		if err != nil {
			continue
		}
		for iter.Next() {
			name := fieldName(iter.Selector())
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			fields = append(fields, field{
				name:     name,
				value:    iter.Value(),
				optional: iter.IsOptional(),
			})
		}
	}
	return fields
}

func structCandidates(value cue.Value) []cue.Value {
	raw := make([]cue.Value, 0, 4)
	// Keep the default first so its field values and optionality take precedence
	// when branches overlap.
	if def, ok := value.Default(); ok {
		raw = append(raw, def)
	}
	// Decompose disjunctions (e.g. *{port: int} | {socket: string}) so that
	// every struct branch is visited. Visiting value.Default() alone would hide
	// the non-default branches because Value.Fields reapplies the default.
	if op, args := value.Expr(); op == cue.OrOp {
		raw = append(raw, args...)
	}
	raw = append(raw, value)
	if value.IncompleteKind()&cue.StructKind != 0 {
		raw = append(raw, value.Unify(value.Context().CompileString("{}")))
	}

	out := make([]cue.Value, 0, len(raw))
	for _, c := range raw {
		if c.Exists() && c.Err() == nil && c.IncompleteKind()&cue.StructKind != 0 {
			out = append(out, c)
		}
	}
	return out
}

func listFields(value cue.Value) []field {
	var fields []field
	if iter, err := value.List(); err == nil {
		for i := 0; iter.Next(); i++ {
			name := fmt.Sprintf("[%d]", i)
			fields = append(fields, field{name: name, value: iter.Value()})
		}
	}
	// Element type for open lists, e.g. [...string] -> [>=0].
	if iter, err := value.Fields(cue.Patterns(true)); err == nil {
		for iter.Next() {
			if iter.Selector().ConstraintType() != cue.PatternConstraint {
				continue
			}
			fields = append(fields, field{name: fieldName(iter.Selector()), value: iter.Value()})
		}
	}
	return fields
}

func buildChildren(parent *Node) []*Node {
	var fields []field
	switch parent.Kind {
	case KindStruct:
		fields = structFields(parent.value)
	case KindList:
		fields = listFields(parent.value)
	default:
		return nil
	}
	children := make([]*Node, 0, len(fields))
	for _, f := range fields {
		children = append(children, newNode(f.name, joinPath(parent.Path, f.name), f.value, f.optional, parent))
	}
	return children
}

func fieldName(sel cue.Selector) string {
	return strings.TrimRight(sel.String(), "?!")
}

func joinPath(parent, name string) string {
	switch parent {
	case "", "#values", "#release":
		return name
	default:
		return parent + "." + name
	}
}

func typeString(value cue.Value) string {
	s := formatValue(value, false)
	kind := value.IncompleteKind()
	switch {
	case kind&cue.StructKind != 0 && (strings.HasPrefix(s, "{") || strings.Contains(s, "\n")):
		if kind == cue.StructKind {
			return "struct"
		}
		// Keep disjunctions like `false | *{...}` readable when compact.
		if len(s) > 80 || strings.Contains(s, "\n") {
			return kind.String()
		}
		return s
	case kind&cue.ListKind != 0 && strings.Contains(s, "\n"):
		return "list"
	case len(s) > 80:
		return kind.String()
	default:
		return s
	}
}

func defaultString(value cue.Value) string {
	def, ok := value.Default()
	if !ok || !def.IsConcrete() {
		return ""
	}
	return formatValue(def, true)
}

func formatValue(value cue.Value, concrete bool) string {
	node := value.Syntax(
		cue.Concrete(concrete),
		cue.Definitions(false),
		cue.Hidden(true),
		cue.Optional(true),
		cue.Attributes(false),
		cue.Docs(false),
	)
	out, err := format.Node(node, format.Simplify())
	if err != nil {
		return value.IncompleteKind().String()
	}
	return strings.TrimSpace(string(out))
}

func docsString(value cue.Value) string {
	groups := value.Doc()
	if len(groups) == 0 {
		return ""
	}
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		if text := commentGroupText(g); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func commentGroupText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	lines := make([]string, 0, len(g.List))
	for _, c := range g.List {
		text := strings.TrimSpace(c.Text)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		lines = append(lines, strings.TrimSpace(text))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
