// Package ast provides Abstract Syntax Tree indexing support for Terragrunt HCL files.
package ast

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ParseHCLFile parses a Terragrunt HCL file using the official hcl2 parser, then walks the AST and builds an IndexedAST
// where nodes are indexed by their line numbers.
func ParseHCLFile(fileName string, contents []byte) (file *IndexedAST, err error) {
	hclFile, diags := hclsyntax.ParseConfig(contents, fileName, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags != nil && diags.HasErrors() {
		return indexAST(hclFile), diags
	}

	return indexAST(hclFile), nil
}

// IndexedNode wraps a hclsyntax.Node with a reference to its parent in the AST
type IndexedNode struct {
	Parent *IndexedNode
	hclsyntax.Node
}

func (n *IndexedNode) GoString() string {
	r := n.Range()
	return fmt.Sprintf("[%d:%d-%d:%d] %s", r.Start.Line, r.Start.Column, r.End.Line, r.End.Column, reflect.TypeOf(n.Node))
}

func (n *IndexedNode) String() string {
	return n.GoString()
}

// IndexedAST contains an indexed version of the HCL AST
type IndexedAST struct {
	// The underlying HCL AST
	HCLFile *hcl.File
	// A map from line number to a list of nodes that start on that line
	Index NodeIndex
	// Locals contains the local attributes in the file, indexed by attribute key
	Locals Scope
	// Includes contains the include blocks in the file, indexed by include block name
	Includes Scope
}

// FindNodeAt returns the node at the given position in the file. If no node is found, returns nil.
func (d *IndexedAST) FindNodeAt(pos hcl.Pos) *IndexedNode {
	// Iterate backwards to find a node that starts before the position
	nodes, ok := d.Index[pos.Line]
	if !ok {
		return nil
	}

	var closest *IndexedNode
	// First try finding a matching node on the same line
	for _, node := range nodes {
		if node.Range().Start.Column <= pos.Column {
			closest = node
		}
	}

	if closest == nil {
		// Iterate backwards by line
		for i := pos.Line - 1; i >= 1; i-- {
			nodes, ok = d.Index[i]
			if !ok || len(nodes) == 0 {
				continue
			}

			closest = nodes[len(nodes)-1]

			break
		}
	}

	if closest == nil {
		return nil
	}
	// Navigate up the AST to find the first node that contains the position.
	node := closest
	for node != nil {
		end := node.Range().End
		if end.Line > pos.Line || end.Line == pos.Line && end.Column > pos.Column {
			return node
		}

		node = node.Parent
	}

	return nil
}

type Scope map[string]*IndexedNode

func (s Scope) Add(node *IndexedNode) {
	switch n := node.Node.(type) {
	case *hclsyntax.Attribute:
		s[n.Name] = node
	case *hclsyntax.Block:
		s[n.Labels[0]] = node
	default:
		panic("invalid node type " + reflect.TypeOf(node.Node).String())
	}
}

// NodeIndex is a map from line number to an ordered list of nodes that start on that line.
type NodeIndex map[int][]*IndexedNode

type nodeIndexBuilder struct {
	stack    []*IndexedNode
	index    NodeIndex
	locals   Scope
	includes Scope
}

func newNodeIndexBuilder() *nodeIndexBuilder {
	return &nodeIndexBuilder{
		index:    make(map[int][]*IndexedNode),
		locals:   make(Scope),
		includes: make(Scope),
	}
}

func (w *nodeIndexBuilder) Enter(node hclsyntax.Node) hcl.Diagnostics {
	var parent *IndexedNode
	if len(w.stack) > 0 {
		parent = w.stack[len(w.stack)-1]
	}

	line := node.Range().Start.Line
	inode := &IndexedNode{
		Parent: parent,
		Node:   node,
	}
	w.stack = append(w.stack, inode)
	w.index[line] = append(w.index[line], inode)

	if IsLocalAttribute(inode) {
		w.locals.Add(inode)
	} else if IsIncludeBlock(inode) {
		w.includes.Add(inode)
	}

	return nil
}

func (w *nodeIndexBuilder) Exit(node hclsyntax.Node) hcl.Diagnostics {
	w.stack = w.stack[0 : len(w.stack)-1]
	return nil
}

// IsLocalAttribute returns TRUE if the node is a hclsyntax.Attribute within a locals {} block.
func IsLocalAttribute(inode *IndexedNode) bool {
	if inode.Parent == nil || inode.Parent.Parent == nil || inode.Parent.Parent.Parent == nil {
		return false
	}

	if _, ok := inode.Parent.Node.(hclsyntax.Attributes); !ok {
		return false
	}

	if _, ok := inode.Parent.Parent.Node.(*hclsyntax.Body); !ok {
		return false
	}

	return IsLocalsBlock(inode.Parent.Parent.Parent)
}

// IsLocalsBlock returns TRUE if the node is an HCL block of type "locals".
func IsLocalsBlock(inode *IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "locals"
}

// IsIncludeBlock returns TRUE if the node is an HCL block of type "include".
func IsIncludeBlock(inode *IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "include" && len(block.Labels) > 0
}

// IsDependencyBlock returns TRUE if the node is an HCL block of type "dependency".
func IsDependencyBlock(inode *IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "dependency" && len(block.Labels) > 0
}

// IsAttribute returns TRUE if the node is an hclsyntax.Attribute.
func IsAttribute(inode *IndexedNode) bool {
	_, ok := inode.Node.(*hclsyntax.Attribute)
	return ok
}

// GetNodeIncludeLabel returns the label of the given node, if it is an include block.
// If the node is not an include block, returns an empty string and false.
func GetNodeIncludeLabel(inode *IndexedNode) (string, bool) {
	attr := FindFirstParentMatch(inode, IsAttribute)
	if attr == nil {
		return "", false
	}

	local := FindFirstParentMatch(attr, IsIncludeBlock)
	if local == nil {
		return "", false
	}

	name := ""
	if labels := local.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

// GetNodeDependencyLabel returns whether the node is part of a dependency block's config_path field.
// If it is, returns the name of the dependency block and TRUE, otherwise returns "" and FALSE.
func GetNodeDependencyLabel(inode *IndexedNode) (string, bool) {
	attr := FindFirstParentMatch(inode, IsAttribute)
	if attr == nil {
		return "", false
	}

	if attr.Node.(*hclsyntax.Attribute).Name != "config_path" {
		return "", false
	}

	dep := FindFirstParentMatch(attr, IsDependencyBlock)
	if dep == nil {
		return "", false
	}

	name := ""
	if labels := dep.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

func FindFirstParentMatch(inode *IndexedNode, matcher func(*IndexedNode) bool) *IndexedNode {
	for cur := inode; cur != nil; cur = cur.Parent {
		if matcher(cur) {
			return cur
		}
	}

	return nil
}

var _ hclsyntax.Walker = &nodeIndexBuilder{}

func indexAST(ast *hcl.File) *IndexedAST {
	body := ast.Body.(*hclsyntax.Body)
	builder := newNodeIndexBuilder()
	_ = hclsyntax.Walk(body, builder)

	return &IndexedAST{
		Index:    builder.index,
		Locals:   builder.locals,
		Includes: builder.includes,
		HCLFile:  ast,
	}
}
