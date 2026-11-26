package tg

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	tgLog "github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/pkg/log/format"
	"github.com/gruntwork-io/terragrunt/pkg/options"
	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"go.lsp.dev/protocol"
)

const defaultMaxFoldersToCheck = 100

func newTGLogger(l logger.Logger) tgLog.Logger {
	return tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)
}

// restrict returns a copy of v that is limited to p, and a recorder that
// collects the denials from that copy.
func restrict(v *venv.Venv, p venv.Policy) (*venv.Venv, *denialRecorder) {
	rec := newDenialRecorder()
	p.Observe = rec.observe

	// When p grants env, venv.Restrict shares v's environment map with the
	// copy. v is reused for every parse, so each parse gets its own clone.
	return venv.Restrict(v.WithEnvCloned(), p), rec
}

// newParsingContext builds a terragrunt ParsingContext for the given file and
// returns a pointer to the HCL diagnostics slice that the parser will populate.
func newParsingContext(
	ctx context.Context,
	filename string,
) (context.Context, *config.ParsingContext, *hcl.Diagnostics, error) {
	parseDiags := &hcl.Diagnostics{}

	opts, err := options.NewTerragruntOptionsWithConfigPath(filename)
	if err != nil {
		return ctx, nil, nil, err
	}

	opts.Experiments.ExperimentMode()

	pctx := config.NewParsingContext(config.WithStrictControls(opts.StrictControls))
	pctx.Experiments = opts.Experiments
	pctx.TerragruntConfigPath = filename
	pctx.OriginalTerragruntConfigPath = filename
	pctx.WorkingDir = filepath.Dir(filename)
	pctx.DownloadDir = opts.DownloadDir
	pctx.SkipOutput = true
	pctx.MaxFoldersToCheck = defaultMaxFoldersToCheck
	pctx.Parser.DiagnosticsHandler = func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
		*parseDiags = append(*parseDiags, hclDiags...)
		return hclDiags, nil
	}

	return config.WithCaches(ctx), pctx, parseDiags, nil
}

// ParseTerragruntBuffer parses text as the unit configuration at filename. The
// parse runs in a copy of v that is limited to p, with text in place of the file at filename.
//
// The returned diagnostics include an Information diagnostic for each distinct
// command or SOPS decryption that p denied. The parse replaces p.Observe to
// collect the denials, so an Observe set by the caller is never called.
func ParseTerragruntBuffer(
	ctx context.Context,
	l logger.Logger,
	v *venv.Venv,
	p venv.Policy,
	filename, text string,
) (*config.TerragruntConfig, []protocol.Diagnostic) {
	tgLogger := newTGLogger(l)
	v, denials := restrict(v, p)

	ctx, pctx, parseDiags, err := newParsingContext(ctx, filename)
	if err != nil {
		l.Error("Error building the Terragrunt parsing context", "error", err)

		return nil, []protocol.Diagnostic{}
	}

	cfg, err := new(config.Store).ResolveRunConfigFromBytes(
		ctx,
		tgLogger,
		v,
		pctx,
		filename,
		[]byte(text),
	)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing Terragrunt config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, *parseDiags, filename, text)

	diags := append(hclDiagsToLSPDiags(filteredDiags), pathDownloadDiagnostics(l, err, filename, text)...)

	return cfg, append(diags, denials.diagnostics()...)
}

// pathDownloadDiagnostics returns an Error diagnostic at the local of a [config.PathDownloadInLocalsError] in err.
func pathDownloadDiagnostics(l logger.Logger, err error, filename, text string) []protocol.Diagnostic {
	var pathDownload config.PathDownloadInLocalsError
	if !errors.As(err, &pathDownload) {
		return nil
	}

	diag := protocol.Diagnostic{
		Severity: protocol.DiagnosticSeverityError,
		Source:   protocol.NewOptional("terragrunt-ls"),
		Message:  protocol.String(pathDownload.Error()),
	}

	if pathDownload.ConfigPath != filename {
		return []protocol.Diagnostic{diag}
	}

	indexed, parseErr := ast.ParseHCLFile(filename, []byte(text))
	if parseErr != nil {
		l.Debug("Error indexing the buffer for the path.download diagnostic", "error", parseErr)
	}

	if local, ok := indexed.Locals[pathDownload.Local]; ok {
		diag.Range = ast.FromHCLRange(local.Range())
	}

	return []protocol.Diagnostic{diag}
}

func filterHCLDiags(l logger.Logger, diags hcl.Diagnostics, filename, text string) hcl.Diagnostics {
	filtered := hcl.Diagnostics{}
	seen := map[diagKey]struct{}{}

	var sharedPathSubjects []hcl.Range

	for _, diag := range diags {
		if diag.Subject == nil {
			filtered = append(filtered, diag)

			continue
		}

		l.Debug(
			"Checking to see diag can be filtered.",
			"diag", diag,
			"filename", filename,
		)

		if diag.Subject.Filename != filename {
			continue
		}

		key := diagKey{severity: diag.Severity, summary: diag.Summary, detail: diag.Detail, subject: *diag.Subject}
		if _, ok := seen[key]; ok {
			l.Debug(
				"Filtering repeated diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		seen[key] = struct{}{}

		if isMissingOutputDiag(diag) {
			l.Debug(
				"Filtering output missing diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if isParentFileNotFoundDiag(diag) {
			l.Debug(
				"Filtering parent file not found diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if isUnresolvableAttributeDiag(diag, text) {
			l.Debug(
				"Filtering unresolvable attribute diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if isSharedPathDiag(diag) {
			l.Debug(
				"Filtering shared path field diag",
				"diag", diag,
				"filename", filename,
			)

			sharedPathSubjects = append(sharedPathSubjects, *diag.Subject)

			continue
		}

		filtered = append(filtered, diag)
	}

	if len(sharedPathSubjects) == 0 {
		return filtered
	}

	subjects := withLocalReferences(l, sharedPathSubjects, filename, text)

	return slices.DeleteFunc(filtered, func(diag *hcl.Diagnostic) bool {
		return followsSharedPathDiag(diag, subjects)
	})
}

// diagKey holds the fields that make two diagnostics the same.
type diagKey struct {
	summary  string
	detail   string
	subject  hcl.Range
	severity hcl.DiagnosticSeverity
}

const (
	// UnsupportedAttributeSummary is the summary for an unsupported attribute diagnostic.
	UnsupportedAttributeSummary = "Unsupported attribute"

	// OutputsMissingDetail is the detail for a missing outputs attribute diagnostic.
	OutputsMissingDetail = "This object does not have an attribute named \"outputs\"."
)

func isMissingOutputDiag(diag *hcl.Diagnostic) bool {
	if diag.Summary != UnsupportedAttributeSummary {
		return false
	}

	if filepath.Base(diag.Subject.Filename) == "terragrunt.hcl" {
		return false
	}

	return diag.Detail == OutputsMissingDetail
}

// sharedPathFields are the path fields that Terragrunt sets only when another file includes or reads the file.
var sharedPathFields = []string{"include", "read", "stack"}

func isSharedPathDiag(diag *hcl.Diagnostic) bool {
	if diag.Expression == nil || shadowsPath(diag.EvalContext) {
		return false
	}

	fileType := DetectFileType(diag.Subject.Filename)
	if fileType != store.FileTypeShared && fileType != store.FileTypeUnknown {
		return false
	}

	switch diag.Summary {
	case UnsupportedAttributeSummary, invalidIndexSummary:
		traversal, diags := hcl.AbsTraversalForExpr(diag.Expression)

		return !diags.HasErrors() && readsSharedPathField(traversal)
	case ErrorInFunctionCallSummary:
		call, ok := diag.Expression.(*hclsyntax.FunctionCallExpr)

		return ok && call.Name == "try" && slices.ContainsFunc(call.Variables(), readsSharedPathField)
	default:
		return false
	}
}

// shadowsPath reports whether an expression in the chain of scope declares a path variable of its own.
func shadowsPath(scope *hcl.EvalContext) bool {
	declarations := 0

	for ; scope != nil; scope = scope.Parent() {
		if _, ok := scope.Variables[config.MetadataPath]; ok {
			declarations++
		}
	}

	return declarations > 1
}

// readsSharedPathField reports whether traversal reads a field of sharedPathFields and nothing in that field.
func readsSharedPathField(traversal hcl.Traversal) bool {
	steps := traversal.SimpleSplit().Rel
	if traversal.RootName() != config.MetadataPath || len(steps) != 1 {
		return false
	}

	switch step := steps[0].(type) {
	case hcl.TraverseAttr:
		return slices.Contains(sharedPathFields, step.Name)
	case hcl.TraverseIndex:
		return step.Key.Type() == cty.String && slices.Contains(sharedPathFields, step.Key.AsString())
	default:
		return false
	}
}

const (
	// invalidIndexSummary is the summary for an index that an object does not have.
	invalidIndexSummary = "Invalid index"

	// unsuitableValueSummary is the summary for a value that an attribute cannot hold.
	unsuitableValueSummary = "Unsuitable value type"

	// unknownValueDetail is the detail for an attribute whose expression gave no value.
	unknownValueDetail = "Unsuitable value: value must be known"
)

// withLocalReferences returns subjects with each reference in text to a local whose value holds one of them.
func withLocalReferences(l logger.Logger, subjects []hcl.Range, filename, text string) []hcl.Range {
	indexed, err := ast.ParseHCLFile(filename, []byte(text))
	if err != nil {
		l.Debug("Error indexing the buffer for the shared path field diags", "error", err)
	}

	body := indexed.HCLFile.Body.(*hclsyntax.Body)
	found := map[string]struct{}{}

	for range len(indexed.Locals) {
		before := len(found)

		for name, local := range indexed.Locals {
			if _, ok := found[name]; ok || !slices.ContainsFunc(subjects, local.Range().Overlaps) {
				continue
			}

			found[name] = struct{}{}

			ast.WalkReferences(body, config.MetadataLocal, name, func(expr *hclsyntax.ScopeTraversalExpr, _ hcl.Range) {
				subjects = append(subjects, expr.Range())
			})
		}

		if len(found) == before {
			break
		}
	}

	return subjects
}

// followsSharedPathDiag reports whether diag is the unknown value error of an expression that holds one of subjects.
func followsSharedPathDiag(diag *hcl.Diagnostic, subjects []hcl.Range) bool {
	if diag.Summary != unsuitableValueSummary || diag.Detail != unknownValueDetail || diag.Context == nil {
		return false
	}

	return slices.ContainsFunc(subjects, diag.Context.Overlaps)
}

const (
	// ErrorInFunctionCallSummary is the summary for an error in a function call diagnostic.
	ErrorInFunctionCallSummary = "Error in function call"

	// ParentFileNotFoundErrorDetailPartial is the partial detail for a parent file not found diagnostic.
	ParentFileNotFoundErrorDetailPartial = `Call to function "find_in_parent_folders" failed: ParentFileNotFoundError`
)

func isParentFileNotFoundDiag(diag *hcl.Diagnostic) bool {
	if diag.Summary != ErrorInFunctionCallSummary {
		return false
	}

	return strings.HasPrefix(diag.Detail, ParentFileNotFoundErrorDetailPartial)
}

const (
	// UnknownVariableSummary is the summary for an unknown variable diagnostic.
	UnknownVariableSummary = "Unknown variable"

	// ValuesUnknownVariableDetail is the detail for a missing "values" variable diagnostic.
	ValuesUnknownVariableDetail = `There is no variable named "values".`
)

// unresolvableKeywords lists object names whose attributes may not be
// available during LS parsing because they depend on runtime state:
//   - "values": populated from terragrunt.values.hcl at runtime.
//   - "local": locals that reference unresolvable values cascade failures.
var unresolvableKeywords = []string{"values", "local"}

// isUnresolvableAttributeDiag checks whether the diagnostic is an "Unsupported
// attribute" or "Unknown variable" error caused by referencing an object that
// cannot be fully resolved during LS parsing.
func isUnresolvableAttributeDiag(diag *hcl.Diagnostic, text string) bool {
	if diag.Summary == UnknownVariableSummary && diag.Detail == ValuesUnknownVariableDetail {
		return true
	}

	if diag.Summary != UnsupportedAttributeSummary {
		return false
	}

	lines := strings.Split(text, "\n")
	line := diag.Subject.Start.Line - 1 // HCL lines are 1-based

	if line < 0 || line >= len(lines) {
		return false
	}

	// The diagnostic start column points to the "." in "keyword.attr".
	// Check that the characters immediately before the dot match a known keyword.
	col := diag.Subject.Start.Column - 1 // HCL columns are 1-based, convert to 0-based

	for _, keyword := range unresolvableKeywords {
		kLen := len(keyword)

		if col < kLen {
			continue
		}

		if lines[line][col-kLen:col] == keyword {
			return true
		}
	}

	return false
}

func hclDiagsToLSPDiags(hclDiags hcl.Diagnostics) []protocol.Diagnostic {
	diags := make([]protocol.Diagnostic, 0, len(hclDiags))

	for _, diag := range hclDiags {
		var diagRange protocol.Range

		if diag.Subject != nil {
			diagRange = protocol.Range{
				Start: protocol.Position{
					Line:      uint32(diag.Subject.Start.Line) - 1,
					Character: uint32(diag.Subject.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(diag.Subject.End.Line) - 1,
					Character: uint32(diag.Subject.End.Column) - 1,
				},
			}
		}

		diags = append(diags, protocol.Diagnostic{
			Range:    diagRange,
			Severity: protocol.DiagnosticSeverity(diag.Severity),
			Source:   protocol.NewOptional("HCL"),
			Message:  protocol.String(diag.Summary + ": " + diag.Detail),
		})
	}

	return diags
}

// DetectFileType returns the FileType for the given filename based on its base name.
func DetectFileType(filename string) store.FileType {
	base := filepath.Base(filename)

	switch base {
	case "terragrunt.hcl":
		return store.FileTypeUnit
	case "terragrunt.stack.hcl":
		return store.FileTypeStack
	case "terragrunt.autoinclude.hcl":
		return store.FileTypeShared
	case "terragrunt.values.hcl":
		return store.FileTypeValues
	default:
		return store.FileTypeUnknown
	}
}

// ParseStackBuffer parses a terragrunt.stack.hcl file and returns the stack
// config and diagnostics. It limits v to p and reports denials in the same way
// as [ParseTerragruntBuffer].
func ParseStackBuffer(
	ctx context.Context,
	l logger.Logger,
	v *venv.Venv,
	p venv.Policy,
	filename, text string,
) (*config.StackConfig, []protocol.Diagnostic) {
	tgLogger := newTGLogger(l)
	v, denials := restrict(v, p)

	ctx, pctx, parseDiags, err := newParsingContext(ctx, filename)
	if err != nil {
		l.Error("Error building the Terragrunt parsing context", "error", err)

		return nil, []protocol.Diagnostic{}
	}

	cfg, err := config.ReadStackConfigString(ctx, tgLogger, v, new(config.Store), pctx, filename, text)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing stack config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, *parseDiags, filename, text)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, append(diags, denials.diagnostics()...)
}

// unitSchema holds the names of the attributes and blocks that Terragrunt accepts at the top level of a unit
// configuration.
type unitSchema struct {
	attributes map[string]struct{}
	blocks     map[string]struct{}
}

func newUnitSchema() unitSchema {
	implied, _ := gohcl.ImpliedBodySchema(&config.TerragruntConfigFile{})

	schema := unitSchema{
		attributes: make(map[string]struct{}, len(implied.Attributes)),
		blocks:     make(map[string]struct{}, len(implied.Blocks)),
	}

	for _, attr := range implied.Attributes {
		schema.attributes[attr.Name] = struct{}{}
	}

	for _, block := range implied.Blocks {
		schema.blocks[block.Type] = struct{}{}
	}

	return schema
}

// accepts reports whether every top-level attribute and block of file is one that a unit configuration can hold. A
// file with none is accepted.
func (u unitSchema) accepts(file *ast.IndexedAST) bool {
	body, ok := file.HCLFile.Body.(*hclsyntax.Body)
	if !ok {
		return false
	}

	for name := range body.Attributes {
		if _, ok := u.attributes[name]; !ok {
			return false
		}
	}

	for _, block := range body.Blocks {
		if _, ok := u.blocks[block.Type]; !ok {
			return false
		}
	}

	return true
}
