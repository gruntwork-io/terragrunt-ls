// Package ast provides utilities for parsing and traversing the HCL AST for Terragrunt configurations.
package ast

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// IndexFileAST parses a Terragrunt HCL file
// using the official hcl2 parser, then walks the
// AST and builds an IndexedAST where nodes are indexed by their line numbers.
func IndexFileAST(fileName string, contents []byte) (*IndexedAST, error) {
	hclFile, diags := hclsyntax.ParseConfig(contents, fileName, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags != nil && diags.HasErrors() {
		return indexAST(hclFile), diags
	}

	return indexAST(hclFile), nil
}

// IndexedNode wraps a hclsyntax.Node with a reference to its parent in the AST.
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

// IndexedAST contains an indexed version of the HCL AST.
type IndexedAST struct {
	// HCLFile is the original HCL file that was parsed.
	HCLFile *hcl.File
	// Index is a map of line numbers to nodes that start on that line.
	Index NodeIndex
	// Locals is a map of local variable names to their nodes.
	Locals Scope
	// Includes is a map of include block names to their nodes.
	Includes Scope
}

// FindNodeAt attempts to find the node at the given position in the AST.
// If no node is found, returns nil.
func (d *IndexedAST) FindNodeAt(pos hcl.Pos) *IndexedNode {
	// Iterate backwards to find a node that starts before the position.
	nodes, ok := d.Index[pos.Line]
	if !ok {
		return nil
	}

	var closest *IndexedNode

	// First try finding a matching node on the same line.
	for _, node := range nodes {
		if node.Range().Start.Column <= pos.Column {
			closest = node
		}
	}

	if closest == nil {
		// Iterate backwards, line by line.
		for i := pos.Line - 1; i >= 1; i-- {
			nodes, ok = d.Index[i]
			if !ok || len(nodes) == 0 {
				continue
			}

			closest = nodes[len(nodes)-1]

			break
		}

		// If we still haven't found a node, return nil.
		if closest == nil {
			return nil
		}
	}

	// Navigate up the AST to find the first node that contains the position.
	node := closest
	for node != nil {
		end := node.Range().End
		if isPosBeforeEnd(pos, end) {
			return node
		}

		node = node.Parent
	}

	return nil
}

func isPosBeforeEnd(pos hcl.Pos, end hcl.Pos) bool {
	return pos.Line < end.Line || pos.Line == end.Line && pos.Column < end.Column
}

type Scope map[string]*IndexedNode

// Add adds a node to the scope.
func (s Scope) Add(node *IndexedNode) error {
	switch n := node.Node.(type) {
	case *hclsyntax.Attribute:
		s[n.Name] = node
	case *hclsyntax.Block:
		key := n.Type

		if len(n.Labels) > 0 {
			labels := strings.Join(n.Labels, ".")
			key += "." + labels
		}

		s[key] = node
	default:
		return fmt.Errorf("invalid node type %s", reflect.TypeOf(node.Node).String())
	}

	return nil
}

// NodeIndex is a map of line numbers to nodes that start on that line.
//
// NOTE: I wonder if it's even advisable to use a map here.
// I think a slice would be more appropriate, since we're
// accessing the nodes by line number anyways.
// It would result in a less compact representation, but
// it would potentially be faster, and would only require a
// single allocation. Requires benchmarking.
type NodeIndex map[int][]*IndexedNode

type nodeIndexBuilder struct {
	stack    []*IndexedNode
	index    NodeIndex
	locals   Scope
	includes Scope
}

func newNodeIndexBuilider() *nodeIndexBuilder {
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
		err := w.locals.Add(inode)
		if err != nil {
			subject := node.Range()

			return hcl.Diagnostics{{
				Summary:  "Failed to add local attribute to scope",
				Detail:   err.Error(),
				Severity: hcl.DiagError,
				Subject:  &subject,
			}}
		}

		return nil
	}

	if block, ok := node.(*hclsyntax.Block); ok && block.Type == "include" && len(block.Labels) > 0 {
		err := w.includes.Add(inode)
		if err != nil {
			subject := node.Range()

			return hcl.Diagnostics{{
				Summary:  "Failed to add include block to scope",
				Detail:   err.Error(),
				Severity: hcl.DiagError,
				Subject:  &subject,
			}}
		}

		return nil
	}

	return nil
}

func (w *nodeIndexBuilder) Exit(node hclsyntax.Node) hcl.Diagnostics {
	w.stack = w.stack[0 : len(w.stack)-1]
	return nil
}

// IsLocalAttribute is true if the node is an hclsyntax.Attribute within a locals {} block.
func IsLocalAttribute(node *IndexedNode) bool {
	if node.Parent == nil || node.Parent.Parent == nil || node.Parent.Parent.Parent == nil {
		return false
	}

	if _, ok := node.Parent.Node.(hclsyntax.Attributes); !ok {
		return false
	}

	if _, ok := node.Parent.Parent.Node.(*hclsyntax.Body); !ok {
		return false
	}

	return IsLocalsBlock(node.Parent.Parent.Parent.Node)
}

// IsLocalsBlock is true if the node is an HCL block of type "locals".
func IsLocalsBlock(node hclsyntax.Node) bool {
	block, ok := node.(*hclsyntax.Block)
	return ok && block.Type == "locals"
}

// IsIncludeBlock is true if the node is an HCL block of type "include".
func IsIncludeBlock(node hclsyntax.Node) bool {
	block, ok := node.(*hclsyntax.Block)
	return ok && block.Type == "include"
}

// IsAttribute is true if the node is an HCL attribute.
func IsAttribute(node hclsyntax.Node) bool {
	_, ok := node.(*hclsyntax.Attribute)
	return ok
}

// GetNodeIncludePath returns the include path of the given node, if it is an include block.
// If the node is not an include block, returns an empty string and false.
func GetNodeIncludePath(inode *IndexedNode) (string, bool) {
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

// FindFirstParentMatch finds the first parent node that matches the given matcher function.
func FindFirstParentMatch(inode *IndexedNode, matcher func(node hclsyntax.Node) bool) *IndexedNode {
	for cur := inode; cur != nil; cur = cur.Parent {
		if matcher(cur.Node) {
			return cur
		}
	}

	return nil
}

var _ hclsyntax.Walker = &nodeIndexBuilder{}

func indexAST(hclFile *hcl.File) *IndexedAST {
	body := hclFile.Body.(*hclsyntax.Body)
	builder := newNodeIndexBuilider()
	_ = hclsyntax.Walk(body, builder)

	return &IndexedAST{
		Index:    builder.index,
		Locals:   builder.locals,
		Includes: builder.includes,
		HCLFile:  hclFile,
	}
}
