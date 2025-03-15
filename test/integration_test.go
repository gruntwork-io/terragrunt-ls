// Package test_test is a package for integration testing.
package test_test

import (
	"os"
	"strings"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	terragruntVersion = "v0.75.10"
)

// TestEveryFixtureInTerragrunt performs a parse of every fixture in the Terragrunt test/fixtures directory.
// Only expected diagnostics should be returned for all of them.
func TestEveryFixtureInTerragrunt(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	// Clone the Terragrunt repo
	terragruntRepo := testutils.CloneGitRepo(t, "https://github.com/gruntwork-io/terragrunt.git", terragruntVersion)
	hclFiles := testutils.HCLFilesInDir(t, terragruntRepo)

	l := testutils.NewTestLogger(t)

	allowedDiags := map[string][]string{}

	for _, hclFile := range hclFiles {
		testName := strings.TrimPrefix(hclFile, terragruntRepo+string(os.PathSeparator))
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			// Parse the HCL file
			cfg, diags := tg.ParseTerragruntBuffer(l, hclFile, testutils.ReadFile(t, hclFile))

			assert.NotNil(t, cfg)

			// Check if the diagnostics are allowed
			allowed, ok := allowedDiags[testName]
			if ok {
				assert.ElementsMatch(t, allowed, diags)
			} else {
				assert.Empty(t, diags)
			}
		})
	}
}
