package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nestoca/joy/internal/yml"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var bin = os.Args[0]

func run() error {
	flag.Usage = func() {
		fmt.Println(bin)
		fmt.Println()
		fmt.Println("description: convert all yaml files within target folder to use tag locks instead of comment locks")
		fmt.Println()
		fmt.Printf("usage: %s $TARGET_FOLDER", bin)
	}

	flag.Parse()

	targetFolder := func() string {
		if arg := flag.Arg(0); arg != "" {
			return arg
		}
		return "."
	}()

	return filepath.WalkDir(targetFolder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if isHidden(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if ext := filepath.Ext(path); ext != ".yml" && ext != ".yaml" {
			return nil
		}

		if err := RewriteYAML(path); err != nil {
			return fmt.Errorf("failed to rewrite %s: %v", path, err)
		}
		return nil
	})
}

func RewriteYAML(name string) (err error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	var node Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}

	rewriteLocks(node.Node)

	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(file.Close()) }()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	return encoder.Encode(node)
}

func rewriteLocks(node *yaml.Node) {
	if yml.IsCommentLocked(node) {
		node.Tag = "!lock"
		node.LineComment = lockExpression.ReplaceAllString(node.LineComment, "")
		node.HeadComment = lockExpression.ReplaceAllString(node.LineComment, "")
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if yml.IsCommentLocked(key) || yml.IsCommentLocked(value) {
				value.Tag = "!lock"
				key.HeadComment = lockExpression.ReplaceAllString(key.HeadComment, "")
				value.LineComment = lockExpression.ReplaceAllString(value.LineComment, "")
			}
		}
	}

	for _, child := range node.Content {
		rewriteLocks(child)
	}
}

// The node type is a wrapper over a yaml.Node. The reason for this wrapper is to make Marshallinga and Unmarshalling easier.
// go-yaml fails to unmarshal document nodes, and we always end up needing to go between the document and the first content item.
// With this we store the first unmashalled node which is the content not the document.
type Node struct {
	*yaml.Node
}

func (n *Node) UnmarshalYAML(node *yaml.Node) error {
	n.Node = node
	return nil
}

func (n Node) MarshalYAML() (any, error) {
	return n.Node, nil
}

var lockExpression = regexp.MustCompile(`(?i)(?m)^#+\s*lock\s*\n?`)

func isHidden(name string) bool {
	base := filepath.Base(name)
	return base != "." && strings.HasSuffix(base, ".")
}
