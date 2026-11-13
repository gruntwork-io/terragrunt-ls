package docs_test

import (
	"net/url"
	"strings"
	"terragrunt-ls/internal/docs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIndex(t *testing.T) {
	t.Parallel()

	page := "# Units\n\n" +
		"## Blocks\n\n" +
		"| Name | Description |\n" +
		"| --- | --- |\n" +
		"| [`terraform`](/reference/hcl/blocks/terraform/)   | Configures the module.   |\n\n" +
		"## Functions\n\n" +
		"| [`get_platform`](/reference/hcl/functions/get_platform/) | Returns the OS, e.g. \\`linux\\`. |\n" +
		"| [`mismatched`](/reference/hcl/functions/other/) | Names a different page. |\n" +
		"| [`Upper`](/reference/hcl/functions/Upper/) | Isn't a valid name. |\n"

	items, err := docs.ParseIndex([]byte(page))
	require.NoError(t, err)

	assert.Equal(t, []docs.Item{
		{Kind: docs.KindBlock, Name: "terraform", Description: "Configures the module."},
		{Kind: docs.KindFunction, Name: "get_platform", Description: "Returns the OS, e.g. `linux`."},
	}, items)
}

func TestParseIndexEmpty(t *testing.T) {
	t.Parallel()

	_, err := docs.ParseIndex([]byte("<!doctype html><title>Not found</title>"))
	require.ErrorIs(t, err, docs.ErrEmptyIndex)
}

func TestParseIndexTooManyItems(t *testing.T) {
	t.Parallel()

	row := "| [`get_env`](/reference/hcl/functions/get_env/) | Returns an environment variable. |\n"

	_, err := docs.ParseIndex([]byte(strings.Repeat(row, docs.MaxIndexItems+1)))
	require.ErrorIs(t, err, docs.ErrTooManyItems)
}

func TestSummarize(t *testing.T) {
	t.Parallel()

	pageURL, err := url.Parse("https://docs.example.com/reference/hcl/functions/run_cmd/")
	require.NoError(t, err)

	tc := []struct {
		name     string
		page     string
		expected string
	}{
		{
			name: "function",
			page: "# run_cmd\n\n" +
				"> Runs a command and returns its stdout.\n\n" +
				"**Returns:** `string`\n\n" +
				"## Usage\n\n" +
				"```hcl\nrun_cmd(...args: string) --> string\n```\n\n" +
				"`run_cmd` runs a command, unless you pass [`--terragrunt-quiet`](#options).\n" +
				"A non-zero exit is an error.\n\n" +
				"It runs without a shell.\n\n" +
				"terragrunt.hcl\n\n" +
				"```hcl\nlocals {}\n```\n\n" +
				"### Options\n\n" +
				"| Option | Effect |\n| --- | --- |\n| `--terragrunt-quiet` | Doesn't print. |\n\n" +
				"## Parameters\n\n" +
				"| Name | Type | Required | Description |\n" +
				"| --- | --- | --- | --- |\n" +
				"| `args` | `string` | Yes | The command, then its arguments. |\n" +
				"| `pipe` | `string` | No | A value with a \\| in it. |\n\n" +
				"## Examples\n\n" +
				"```hcl\nlocals { commit = run_cmd(\"git\") }\n```\n",
			expected: "Runs a command and returns its stdout.\n\n" +
				"```hcl\nrun_cmd(...args: string) --> string\n```\n\n" +
				"`run_cmd` runs a command, unless you pass " +
				"[`--terragrunt-quiet`](https://docs.example.com/reference/hcl/functions/run_cmd/#options).\n" +
				"A non-zero exit is an error.\n\n" +
				"**Parameters:**\n" +
				"- `args` (`string`): The command, then its arguments.\n" +
				"- `pipe` (`string`, optional): A value with a | in it.",
		},
		{
			name: "attribute",
			page: "# inputs\n\n" +
				"> Sets the input variables.\n\n" +
				"**Type:** `map(any)`\n\n" +
				"## Usage\n\n" +
				"The `inputs` attribute sets a module's variables.\n\n" +
				"* live\n  * app\n\n" +
				"## Examples\n\n" +
				"```hcl\ninputs = {}\n```\n",
			expected: "Sets the input variables.\n\n" +
				"**Type:** `map(any)`\n\n" +
				"The `inputs` attribute sets a module's variables.",
		},
		{
			name: "block",
			page: "# terraform\n\n" +
				"> Configures the module.\n\n" +
				"## Usage\n\n" +
				"It skips files unless [`include_in_copy`](/reference/hcl/blocks/terraform#include_in_copy) names them.\n\n" +
				"```hcl\n## Not a heading\nterraform {}\n```\n\n" +
				"### Module sources\n\n" +
				"A `tfr://` source pulls from a registry.\n\n" +
				"## Attributes\n\n" +
				"### `source`\n\n" +
				"```hcl\n### `not_an_attribute`\n```\n\n" +
				"### `include_in_copy`\n\n" +
				"## Blocks\n\n" +
				"### `before_hook`\n",
			expected: "Configures the module.\n\n" +
				"It skips files unless " +
				"[`include_in_copy`](https://docs.example.com/reference/hcl/blocks/terraform#include_in_copy) " +
				"names them.\n\n" +
				"**Attributes:** `source`, `include_in_copy`\n\n" +
				"**Blocks:** `before_hook`",
		},
		{
			name:     "function without parameters",
			page:     "# get_platform\n\n> Returns the OS.\n\n## Usage\n\n```hcl\nget_platform() --> string\n```\n",
			expected: "Returns the OS.\n\n```hcl\nget_platform() --> string\n```",
		},
		{
			name:     "unclosed code block",
			page:     "# x\n\n> Does x.\n\n## Usage\n\n```hcl\nx {\n",
			expected: "Does x.",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, docs.Summarize([]byte(tt.page), pageURL))
		})
	}
}

func TestSnapshotFS(t *testing.T) {
	t.Parallel()

	snapshot := docs.SnapshotFS()

	for _, cfg := range docs.Configs {
		t.Run(string(cfg), func(t *testing.T) {
			t.Parallel()

			body, err := snapshot.Open(string(cfg) + ".md")
			require.NoError(t, err)
			require.NoError(t, body.Close())
		})
	}

	base, err := url.Parse(docs.DefaultBaseURL)
	require.NoError(t, err)

	_, err = docs.New(&docs.Options{Snapshot: snapshot, BaseURL: base})
	require.NoError(t, err)
}
