package docs_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"terragrunt-ls/internal/docs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetchFrom fetches /page.md with etag from a server that answers with status,
// the ETag header responseETag, and a Markdown body.
func fetchFrom(t *testing.T, status int, responseETag, etag string) (docs.Page, error) {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", responseETag)
		w.Header().Set("Content-Type", "text/markdown")
		w.WriteHeader(status)

		if status == http.StatusOK {
			_, err := w.Write([]byte("# page\n"))
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(s.Close)

	base, err := url.Parse(s.URL)
	require.NoError(t, err)

	return docs.Fetch(t.Context(), s.Client(), base, "/page.md", etag)
}

func TestFetchReturnsTheETag(t *testing.T) {
	t.Parallel()

	page, err := fetchFrom(t, http.StatusOK, `W/"abc123"`, "")
	require.NoError(t, err)

	assert.Equal(t, "# page\n", string(page.Body))
	assert.Equal(t, `W/"abc123"`, page.ETag)
}

func TestFetchDropsAnInvalidETag(t *testing.T) {
	t.Parallel()

	for name, etag := range map[string]string{
		"space":    `"a b"`,
		"too long": `"` + strings.Repeat("a", docs.MaxETagBytes) + `"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			page, err := fetchFrom(t, http.StatusOK, etag, "")
			require.NoError(t, err)
			assert.Empty(t, page.ETag)
		})
	}
}

func TestFetchNotModified(t *testing.T) {
	t.Parallel()

	_, err := fetchFrom(t, http.StatusNotModified, `"abc123"`, `"abc123"`)
	require.ErrorIs(t, err, docs.ErrNotModified)
}

func TestFetchRejectsAnErrorStatus(t *testing.T) {
	t.Parallel()

	_, err := fetchFrom(t, http.StatusNotFound, "", "")

	var statusErr *docs.StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
}
