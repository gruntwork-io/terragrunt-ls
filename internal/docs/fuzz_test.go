package docs_test

import (
	"net/url"
	"regexp"
	"strings"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/testutils"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var itemName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

// FuzzParseIndex tests that every item ParseIndex returns has a known kind and
// a name that is safe in a URL path and a file name.
func FuzzParseIndex(f *testing.F) {
	f.Add([]byte(testutils.DocsUnitsIndex))
	f.Add([]byte(testutils.DocsStacksIndex))
	f.Add([]byte("| [`a`](/reference/hcl/blocks/a/) | \\| |"))

	f.Fuzz(func(t *testing.T, page []byte) {
		items, err := docs.ParseIndex(page)
		if err != nil {
			return
		}

		require.LessOrEqual(t, len(items), docs.MaxIndexItems)

		for _, item := range items {
			assert.Contains(t, []docs.Kind{docs.KindBlock, docs.KindAttribute, docs.KindFunction}, item.Kind)
			assert.Regexp(t, itemName, item.Name)
		}
	})
}

// FuzzParseVariable tests that every item ParseVariable returns is the variable or one of its fields, with safe names.
func FuzzParseVariable(f *testing.F) {
	f.Add([]byte(testutils.DocsPathVariable))
	f.Add([]byte("> d\n## Fields\n- [`path.a`](/reference/hcl/variables/path/a/): \\|"))

	path := docs.Variable{Name: "path", Experiment: "path-variable"}

	f.Fuzz(func(t *testing.T, page []byte) {
		items, err := docs.ParseVariable(path, page)
		if err != nil {
			return
		}

		require.LessOrEqual(t, len(items), docs.MaxIndexItems+1)
		assert.Equal(t, "path", items[0].Name)

		for _, item := range items[1:] {
			assert.Equal(t, docs.KindVariable, item.Kind)

			field, ok := strings.CutPrefix(item.Name, "path.")
			require.True(t, ok, item.Name)
			assert.Regexp(t, itemName, field)
		}
	})
}

// FuzzSummarize tests that Summarize handles any page.
func FuzzSummarize(f *testing.F) {
	base, err := url.Parse(docs.DefaultBaseURL)
	require.NoError(f, err)

	f.Add([]byte(getEnvPage))
	f.Add([]byte("# x\n\n> d\n\n## Usage\n\n```\n## y\n[a](/b#c)\n"))
	f.Add([]byte("[a](/%zz)"))

	f.Fuzz(func(t *testing.T, page []byte) {
		docs.Summarize(page, base)
	})
}
