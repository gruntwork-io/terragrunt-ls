package tg

import (
	"os"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/ast"
	astconfig "terragrunt-ls/internal/ast/config"
	aststack "terragrunt-ls/internal/ast/stack"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type State struct {
	// Map of file names to Terragrunt configs
	Configs map[string]store.Store
	// Map of file names to stack configs
	StackConfigs map[string]store.StackStore
	// Map of file names to values configs
	ValuesConfigs map[string]store.ValuesStore
}

func NewState() State {
	return State{Configs: map[string]store.Store{}, StackConfigs: map[string]store.StackStore{}, ValuesConfigs: map[string]store.ValuesStore{}}
}

func (s *State) OpenDocument(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Opening document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(l, docURI, text)
}

func (s *State) UpdateDocument(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Updating document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(l, docURI, text)
}

func (s *State) updateState(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	// Ignore errors from AST indexing since we'll get the same errors from the Terragrunt parser just below
	astTree, _ := ast.ParseHCLFile(docURI.Filename(), []byte(text))

	filename := docURI.Filename()
	fileType := GetTerragruntFileType(filename)

	switch fileType {
	case TerragruntFileTypeConfig:
		return s.updateConfigState(l, docURI, text, astTree)
	case TerragruntFileTypeStack:
		return s.updateStackState(l, docURI, text, astTree)
	case TerragruntFileTypeValues:
		return s.updateValuesState(l, docURI, text, astTree)
	case TerragruntFileTypeUnknown:
		l.Debug("Unknown file type", "filename", filename, "fileType", fileType)
		return []protocol.Diagnostic{}
	default:
		l.Debug("Unknown file type", "filename", filename, "fileType", fileType)
		return []protocol.Diagnostic{}
	}
}

func (s *State) updateConfigState(l logger.Logger, docURI protocol.DocumentURI, text string, astTree *ast.IndexedAST) []protocol.Diagnostic {
	cfg, diags := ParseTerragruntConfigBuffer(l, docURI.Filename(), text)

	l.Debug(
		"Config",
		"uri", docURI,
		"config", cfg,
	)

	cfgAsCty := cty.NilVal

	if cfg != nil {
		if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
			cfgAsCty = converted
		}
	}

	s.Configs[docURI.Filename()] = store.Store{
		AST:      astconfig.NewConfigAST(astTree),
		Cfg:      cfg,
		CfgAsCty: cfgAsCty,
		Document: text,
	}

	return diags
}

func (s *State) updateStackState(l logger.Logger, docURI protocol.DocumentURI, text string, astTree *ast.IndexedAST) []protocol.Diagnostic {
	stackCfg, diags := ParseTerragruntStackBuffer(l, docURI.Filename(), text)

	l.Debug(
		"Stack Config",
		"uri", docURI,
		"config", stackCfg,
	)

	s.StackConfigs[docURI.Filename()] = store.StackStore{
		AST:      aststack.NewStackAST(astTree),
		StackCfg: stackCfg,
		Document: text,
	}

	return diags
}

func (s *State) updateValuesState(l logger.Logger, docURI protocol.DocumentURI, text string, astTree *ast.IndexedAST) []protocol.Diagnostic {
	valuesHCL, diags := ParseTerragruntValuesBuffer(l, docURI.Filename(), text)

	l.Debug(
		"Values Config",
		"uri", docURI,
		"valuesHCL", valuesHCL,
	)

	s.ValuesConfigs[docURI.Filename()] = store.ValuesStore{
		AST:       astTree,
		ValuesHCL: valuesHCL,
		Document:  text,
	}

	return diags
}

func (s *State) Hover(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	filename := docURI.Filename()
	fileType := GetTerragruntFileType(filename)

	l.Debug(
		"Hovering over character",
		"uri", docURI,
		"position", position,
		"fileType", fileType,
	)

	switch fileType {
	case TerragruntFileTypeConfig:
		return s.hoverConfig(l, id, docURI, position)
	case TerragruntFileTypeStack:
		return s.hoverStack(l, id, docURI, position)
	case TerragruntFileTypeValues:
		return s.hoverValues(l, id, docURI, position)
	case TerragruntFileTypeUnknown:
		return newEmptyHoverResponse(id)
	default:
		return newEmptyHoverResponse(id)
	}
}

func (s *State) hoverConfig(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store, ok := s.Configs[docURI.Filename()]
	if !ok {
		return newEmptyHoverResponse(id)
	}

	l.Debug(
		"Config",
		"uri", docURI,
		"config", store.Cfg,
	)

	word, context := hover.GetHoverTargetWithContext(l, store, position)

	l.Debug(
		"Hovering with context",
		"word", word,
		"context", context,
	)

	if word == "" {
		return newEmptyHoverResponse(id)
	}

	//nolint:gocritic
	switch context {
	case hover.HoverContextLocal:
		if store.Cfg == nil {
			return newEmptyHoverResponse(id)
		}

		if _, ok := store.Cfg.Locals[word]; !ok {
			return newEmptyHoverResponse(id)
		}

		if store.CfgAsCty.IsNull() {
			return newEmptyHoverResponse(id)
		}

		locals := store.CfgAsCty.GetAttr("locals")
		localVal := locals.GetAttr(word)

		f := hclwrite.NewEmptyFile()
		rootBody := f.Body()
		rootBody.SetAttributeValue(word, localVal)

		return lsp.HoverResponse{
			Response: lsp.Response{
				RPC: lsp.RPCVersion,
				ID:  &id,
			},
			Result: lsp.HoverResult{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: text.WrapAsHCLCodeFence(strings.TrimSpace(string(f.Bytes()))),
				},
			},
		}
	}

	return newEmptyHoverResponse(id)
}

func (s *State) hoverStack(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store, ok := s.StackConfigs[docURI.Filename()]
	if !ok {
		return newEmptyHoverResponse(id)
	}

	l.Debug(
		"Stack hover requested",
		"uri", docURI,
		"position", position,
	)

	target, context := hover.GetStackHoverTargetWithContext(l, store, position)

	l.Debug(
		"Stack hover target discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyHoverResponse(id)
	}

	switch context {
	case hover.HoverContextStackUnit:
		return newStackUnitHoverResponse(id, target)
	case hover.HoverContextStackSource:
		return newStackSourceHoverResponse(id, target)
	case hover.HoverContextStackPath:
		return newStackPathHoverResponse(id, target)
	case hover.HoverContextStackBlock:
		return newStackBlockHoverResponse(id, target)
	default:
		return newEmptyHoverResponse(id)
	}
}

func (s *State) hoverValues(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store, ok := s.ValuesConfigs[docURI.Filename()]
	if !ok {
		return newEmptyHoverResponse(id)
	}

	l.Debug(
		"Values hover requested",
		"uri", docURI,
		"position", position,
	)

	target, context := hover.GetValuesHoverTargetWithContext(l, store, position)

	l.Debug(
		"Values hover target discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyHoverResponse(id)
	}

	switch context {
	case hover.HoverContextValuesVariable:
		return newValuesVariableHoverResponse(id, target)
	case hover.HoverContextValuesDependency:
		return newValuesDependencyHoverResponse(id, target)
	default:
		return newEmptyHoverResponse(id)
	}
}

// Helper functions for creating hover responses

func newStackUnitHoverResponse(id int, unitName string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Unit: `" + unitName + "`**\n\nA unit block defines a single infrastructure component in a Terragrunt stack.\n\nEach unit has a source (where the Terraform code lives) and a path (where it will be deployed).",
			},
		},
	}
}

func newStackSourceHoverResponse(id int, source string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Source: `" + source + "`**\n\nThe source attribute specifies where the Terraform module or configuration is located.\n\nThis can be a local path, Git repository, or other supported Terraform module sources.",
			},
		},
	}
}

func newStackPathHoverResponse(id int, path string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Path: `" + path + "`**\n\nThe path attribute specifies the relative directory where this unit will be deployed.\n\nThis path is relative to the stack directory and determines where Terragrunt will run commands for this unit.",
			},
		},
	}
}

func newStackBlockHoverResponse(id int, stackName string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Stack: `" + stackName + "`**\n\nA stack block defines a nested stack within the current stack.\n\nNested stacks allow you to organize and compose multiple related infrastructure units together.",
			},
		},
	}
}

func newValuesVariableHoverResponse(id int, variable string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Variable: `" + variable + "`**\n\nThis appears to be a variable defined in the values block.\n\nValues files are used to define dynamic input values for units in Terragrunt stacks.",
			},
		},
	}
}

func newValuesDependencyHoverResponse(id int, dependency string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Dependency: `" + dependency + "`**\n\nA dependency reference allows you to use outputs from other units in your stack.\n\nThe dependency block defines where to find the output values and provides mock values for testing.",
			},
		},
	}
}

func newEmptyHoverResponse(id int) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
	}
}

func (s *State) Definition(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	filename := docURI.Filename()
	fileType := GetTerragruntFileType(filename)

	l.Debug(
		"Definition requested",
		"uri", docURI,
		"position", position,
		"fileType", fileType,
	)

	switch fileType {
	case TerragruntFileTypeConfig:
		return s.definitionConfig(l, id, docURI, position)
	case TerragruntFileTypeStack:
		return s.definitionStack(l, id, docURI, position)
	case TerragruntFileTypeValues:
		return s.definitionValues(l, id, docURI, position)
	case TerragruntFileTypeUnknown:
		return newEmptyDefinitionResponse(id, docURI, position)
	default:
		return newEmptyDefinitionResponse(id, docURI, position)
	}
}

func (s *State) definitionConfig(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store, ok := s.Configs[docURI.Filename()]
	if !ok {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	target, context := definition.GetDefinitionTargetWithContext(l, store, position)

	l.Debug(
		"Definition discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	//nolint:gocritic
	switch context {
	case definition.DefinitionContextInclude:
		l.Debug(
			"Store content",
			"store", store,
		)

		if store.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Includes",
			"includes", store.Cfg.ProcessedIncludes,
		)

		for _, include := range store.Cfg.ProcessedIncludes {
			if include.Name == target {
				l.Debug(
					"Jumping to target",
					"include", include,
				)

				defURI := uri.File(include.Path)

				l.Debug(
					"URI of target",
					"URI", defURI,
				)

				return lsp.DefinitionResponse{
					Response: lsp.Response{
						RPC: lsp.RPCVersion,
						ID:  &id,
					},
					Result: protocol.Location{
						URI: defURI,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				}
			}
		}
	case definition.DefinitionContextDependency:
		l.Debug(
			"Store content",
			"store", store,
		)

		if store.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Dependencies",
			"dependencies", store.Cfg.TerragruntDependencies,
		)

		for _, dep := range store.Cfg.TerragruntDependencies {
			if dep.Name == target {
				l.Debug(
					"Jumping to target",
					"dependency", dep,
				)

				path := dep.ConfigPath.AsString()

				defURI := uri.File(path)
				if !filepath.IsAbs(path) {
					defURI = uri.File(filepath.Join(filepath.Dir(docURI.Filename()), path, "terragrunt.hcl"))
				}

				_, err := os.Stat(defURI.Filename())
				if err != nil {
					l.Warn(
						"Dependency does not exist",
						"dependency", dep,
						"error", err,
					)

					return newEmptyDefinitionResponse(id, docURI, position)
				}

				l.Debug(
					"URI of target",
					"URI", defURI,
				)

				return lsp.DefinitionResponse{
					Response: lsp.Response{
						RPC: lsp.RPCVersion,
						ID:  &id,
					},
					Result: protocol.Location{
						URI: defURI,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				}
			}
		}
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func (s *State) definitionStack(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store, ok := s.StackConfigs[docURI.Filename()]
	if !ok {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	l.Debug(
		"Stack definition requested",
		"uri", docURI,
		"position", position,
	)

	currentDir := filepath.Dir(docURI.Filename())
	target, context := definition.GetStackDefinitionTargetWithContext(l, store, position, currentDir)

	l.Debug(
		"Stack definition target discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	switch context {
	case definition.DefinitionContextStackSource:
		if resolved, ok := definition.ResolveStackSourceLocation(target, currentDir); ok {
			defURI := uri.File(resolved)
			l.Debug("Navigating to source", "source", target, "resolved", resolved)

			return lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: lsp.RPCVersion,
					ID:  &id,
				},
				Result: protocol.Location{
					URI: defURI,
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 0},
					},
				},
			}
		}

		l.Debug("Could not resolve source location", "source", target)

	case definition.DefinitionContextStackPath:
		defURI := uri.File(target)
		l.Debug("Navigating to unit path", "resolved", target)

		return lsp.DefinitionResponse{
			Response: lsp.Response{
				RPC: lsp.RPCVersion,
				ID:  &id,
			},
			Result: protocol.Location{
				URI: defURI,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
			},
		}
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func (s *State) definitionValues(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store, ok := s.ValuesConfigs[docURI.Filename()]
	if !ok {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	l.Debug(
		"Values definition requested",
		"uri", docURI,
		"position", position,
	)

	target, context := definition.GetValuesDefinitionTargetWithContext(l, store, position)

	l.Debug(
		"Values definition target discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	if context == definition.DefinitionContextValuesDependency {
		if resolved, ok := definition.ResolveValuesDependencyPath(target, docURI.Filename()); ok {
			defURI := uri.File(resolved)
			l.Debug("Navigating to dependency", "dependency", target, "resolved", resolved)

			return lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: lsp.RPCVersion,
					ID:  &id,
				},
				Result: protocol.Location{
					URI: defURI,
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 0},
					},
				},
			}
		}

		l.Debug("Could not resolve dependency path", "dependency", target)
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func newEmptyDefinitionResponse(id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	return lsp.DefinitionResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: protocol.Location{
			URI: docURI,
			Range: protocol.Range{
				Start: position,
				End:   position,
			},
		},
	}
}

func (s *State) TextDocumentCompletion(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
	filename := docURI.Filename()
	fileType := GetTerragruntFileType(filename)

	var items []protocol.CompletionItem

	switch fileType {
	case TerragruntFileTypeConfig:
		if store, ok := s.Configs[filename]; ok {
			items = completion.GetCompletions(l, store, position, filename)
		}
	case TerragruntFileTypeStack:
		if stackStore, ok := s.StackConfigs[filename]; ok {
			compatStore := store.Store{
				Document: stackStore.Document,
			}
			items = completion.GetCompletions(l, compatStore, position, filename)
		}
	case TerragruntFileTypeValues:
		if valuesStore, ok := s.ValuesConfigs[filename]; ok {
			compatStore := store.Store{
				Document: valuesStore.Document,
			}
			items = completion.GetCompletions(l, compatStore, position, filename)
		}
	case TerragruntFileTypeUnknown:
		items = []protocol.CompletionItem{}
	}

	response := lsp.CompletionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: items,
	}

	return response
}

func (s *State) TextDocumentFormatting(l logger.Logger, id int, docURI protocol.DocumentURI) lsp.FormatResponse {
	filename := docURI.Filename()
	fileType := GetTerragruntFileType(filename)

	l.Debug(
		"Formatting requested",
		"uri", docURI,
		"fileType", fileType,
	)

	var document string

	var found bool

	switch fileType {
	case TerragruntFileTypeConfig:
		if store, ok := s.Configs[filename]; ok {
			document = store.Document
			found = true
		}
	case TerragruntFileTypeStack:
		if stackStore, ok := s.StackConfigs[filename]; ok {
			document = stackStore.Document
			found = true
		}
	case TerragruntFileTypeValues:
		if valuesStore, ok := s.ValuesConfigs[filename]; ok {
			document = valuesStore.Document
			found = true
		}
	case TerragruntFileTypeUnknown:
	default:
	}

	if !found {
		return lsp.FormatResponse{
			Response: lsp.Response{
				RPC: lsp.RPCVersion,
				ID:  &id,
			},
			Result: []protocol.TextEdit{},
		}
	}

	formatted := hclwrite.Format([]byte(document))

	return lsp.FormatResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: []protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: getEndOfDocument(document),
				},
				NewText: string(formatted),
			},
		},
	}
}

func getEndOfDocument(doc string) protocol.Position {
	lines := strings.Split(doc, "\n")

	return protocol.Position{
		Line:      uint32(len(lines) - 1),
		Character: uint32(len(lines[len(lines)-1])),
	}
}
