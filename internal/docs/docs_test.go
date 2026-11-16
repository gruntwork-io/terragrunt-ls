package docs_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/testutils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const getEnvPage = "# get_env\n\n" +
	"> Returns the value of an environment variable.\n\n" +
	"## Usage\n\n" +
	"```hcl\nget_env(name: string, [default: string]) --> string\n```\n\n" +
	"## Examples\n\n" +
	"```hcl\ninputs = {}\n```\n"

const getEnvSummary = "Returns the value of an environment variable.\n\n" +
	"```hcl\nget_env(name: string, [default: string]) --> string\n```"

// docsServer serves pages by URL path as Markdown with an ETag of their
// content, answers a request that names a page's current ETag with a 304,
// answers any other path with a 500, and counts the requests and 304s for each
// path.
type docsServer struct {
	pages       map[string]string
	requests    map[string]int
	notModified map[string]int
	*httptest.Server
	mu sync.Mutex
}

func newDocsServer(t *testing.T, pages map[string]string) *docsServer {
	t.Helper()

	s := &docsServer{pages: pages, requests: map[string]int{}, notModified: map[string]int{}}

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.requests[r.URL.Path]++

		page, ok := s.pages[r.URL.Path]
		if !ok {
			http.Error(w, "unavailable", http.StatusInternalServerError)

			return
		}

		etag := fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(page)))
		w.Header().Set("ETag", etag)

		if r.Header.Get("If-None-Match") == etag {
			s.notModified[r.URL.Path]++

			w.WriteHeader(http.StatusNotModified)

			return
		}

		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, err := w.Write([]byte(page))
		assert.NoError(t, err)
	}))
	t.Cleanup(s.Close)

	return s
}

func (s *docsServer) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.requests[path]
}

func (s *docsServer) countNotModified(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.notModified[path]
}

func (s *docsServer) setPage(path, page string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages[path] = page
}

// ageCache makes a cached file older than the default maximum age.
func ageCache(t *testing.T, name string) {
	t.Helper()

	old := time.Now().Add(-2 * docs.DefaultMaxAge)
	require.NoError(t, os.Chtimes(name, old, old))
}

func newDocs(t *testing.T, opts *docs.Options) *docs.Docs {
	t.Helper()

	opts.Snapshot = testutils.DocsSnapshot()
	opts.Logger = testutils.NewTestLogger(t)

	d, err := docs.New(opts)
	require.NoError(t, err)

	return d
}

func serverOptions(t *testing.T, s *docsServer) *docs.Options {
	t.Helper()

	base, err := url.Parse(s.URL)
	require.NoError(t, err)

	return &docs.Options{BaseURL: base, Client: s.Client(), CacheDir: t.TempDir()}
}

func TestDocsHoverFetchesThePageInTheBackground(t *testing.T) {
	t.Parallel()

	s := newDocsServer(t, map[string]string{"/reference/hcl/functions/get_env.md": getEnvPage})
	opts := serverOptions(t, s)
	d := newDocs(t, opts)
	link := "\n\n[Terragrunt docs](" + s.URL + "/reference/hcl/functions/get_env/)"

	first, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.Equal(t, "Returns the value of an environment variable."+link+"\n\n"+docs.FetchingNote, first)

	d.Wait()

	second, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.Equal(t, getEnvSummary+link, second)

	cached, err := os.ReadFile(filepath.Join(opts.CacheDir, docs.CacheVersion, "functions", "get_env.md"))
	require.NoError(t, err)
	assert.Equal(t, getEnvPage, string(cached))

	d.Wait()
	assert.Equal(t, 1, s.count("/reference/hcl/functions/get_env.md"))
}

func TestDocsHoverReadsTheCacheWithoutFetching(t *testing.T) {
	t.Parallel()

	s := newDocsServer(t, map[string]string{"/reference/hcl/functions/get_env.md": getEnvPage})
	opts := serverOptions(t, s)

	warm := newDocs(t, opts)
	warm.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	warm.Wait()

	opts.Client = nil
	offline := newDocs(t, opts)

	hover, ok := offline.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(hover, getEnvSummary), hover)
}

func TestDocsHoverRefetchesAStalePage(t *testing.T) {
	t.Parallel()

	s := newDocsServer(t, map[string]string{"/reference/hcl/functions/get_env.md": getEnvPage})
	opts := serverOptions(t, s)

	name := filepath.Join(opts.CacheDir, docs.CacheVersion, "functions", "get_env.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
	require.NoError(t, os.WriteFile(name, []byte("# get_env\n\n> An old description.\n"), 0o644))
	ageCache(t, name)

	d := newDocs(t, opts)

	hover, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(hover, "An old description."), hover)

	d.Wait()

	assert.Equal(t, 1, s.count("/reference/hcl/functions/get_env.md"))

	hover, ok = d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(hover, getEnvSummary), hover)
}

func TestDocsHoverRevalidatesAnUnchangedPage(t *testing.T) {
	t.Parallel()

	const path = "/reference/hcl/functions/get_env.md"

	s := newDocsServer(t, map[string]string{path: getEnvPage})
	opts := serverOptions(t, s)
	name := filepath.Join(opts.CacheDir, docs.CacheVersion, "functions", "get_env.md")

	warm := newDocs(t, opts)
	warm.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	warm.Wait()

	etag, err := os.ReadFile(name + ".etag")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(getEnvPage))), string(etag))

	ageCache(t, name)

	d := newDocs(t, opts)
	d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	d.Wait()

	assert.Equal(t, 2, s.count(path))
	assert.Equal(t, 1, s.countNotModified(path))

	info, err := os.Stat(name)
	require.NoError(t, err)
	assert.Less(t, time.Since(info.ModTime()), docs.DefaultMaxAge)

	hover, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(hover, getEnvSummary), hover)

	d.Wait()
	assert.Equal(t, 2, s.count(path))
}

func TestDocsHoverDownloadsAChangedPage(t *testing.T) {
	t.Parallel()

	const path = "/reference/hcl/functions/get_env.md"

	s := newDocsServer(t, map[string]string{path: getEnvPage})
	opts := serverOptions(t, s)
	name := filepath.Join(opts.CacheDir, docs.CacheVersion, "functions", "get_env.md")

	warm := newDocs(t, opts)
	warm.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	warm.Wait()

	changed := "# get_env\n\n> Returns an environment variable, or a default.\n"
	s.setPage(path, changed)
	ageCache(t, name)

	d := newDocs(t, opts)
	d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	d.Wait()

	assert.Equal(t, 0, s.countNotModified(path))

	hover, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(hover, "Returns an environment variable, or a default."), hover)

	etag, err := os.ReadFile(name + ".etag")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(changed))), string(etag))
}

func TestDocsRefreshDropsTheETagOfAnUnreadableCachedIndex(t *testing.T) {
	t.Parallel()

	s := newDocsServer(t, map[string]string{"/reference/hcl/units.md": testutils.DocsUnitsIndex})
	opts := serverOptions(t, s)
	name := filepath.Join(opts.CacheDir, docs.CacheVersion, "units.md")

	// The cached copy doesn't parse, but its ETag says it's current.
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
	require.NoError(t, os.WriteFile(name, []byte("not an index"), 0o644))
	require.NoError(t, os.WriteFile(
		name+".etag",
		fmt.Appendf(nil, `"%x"`, sha256.Sum256([]byte(testutils.DocsUnitsIndex))),
		0o644,
	))
	ageCache(t, name)

	d := newDocs(t, opts)
	d.Refresh(t.Context())
	d.Wait()

	assert.Equal(t, 1, s.countNotModified("/reference/hcl/units.md"))
	assert.NoFileExists(t, name+".etag")
}

func TestDocsHoverWaitsAfterAFailedFetch(t *testing.T) {
	t.Parallel()

	s := newDocsServer(t, map[string]string{})
	opts := serverOptions(t, s)
	d := newDocs(t, opts)

	for i := range 3 {
		hover, ok := d.Hover(t.Context(), docs.ConfigUnit, docs.KindFunction, "get_env")
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(hover, "Returns the value of an environment variable.\n\n"), hover)
		assert.Equal(t, i == 0, strings.HasSuffix(hover, docs.FetchingNote), hover)

		d.Wait()
	}

	assert.Equal(t, 1, s.count("/reference/hcl/functions/get_env.md"))

	cached, err := os.ReadDir(opts.CacheDir)
	require.NoError(t, err)
	assert.Empty(t, cached)
}

func TestDocsHoverOnlyItemsValidInTheConfig(t *testing.T) {
	t.Parallel()

	d := testutils.NewDocs(t)

	_, ok := d.Hover(t.Context(), docs.ConfigStack, docs.KindAttribute, "inputs")
	assert.False(t, ok)

	_, ok = d.Hover(t.Context(), docs.ConfigUnit, docs.KindBlock, "inputs")
	assert.False(t, ok)

	_, ok = d.Hover(t.Context(), docs.ConfigUnit, docs.KindAttribute, "inputs")
	assert.True(t, ok)
}

func TestDocsRefresh(t *testing.T) {
	t.Parallel()

	units := testutils.DocsUnitsIndex +
		"| [`get_platform`](/reference/hcl/functions/get_platform/) | Returns the operating system. |\n"

	s := newDocsServer(t, map[string]string{
		"/reference/hcl/units.md":  units,
		"/reference/hcl/stacks.md": "<!doctype html>",
	})
	opts := serverOptions(t, s)
	d := newDocs(t, opts)

	d.Refresh(t.Context())
	d.Wait()

	item, ok := d.Lookup(docs.ConfigUnit, docs.KindFunction, "get_platform")
	require.True(t, ok)
	assert.Equal(t, "Returns the operating system.", item.Description)

	// The stacks page lists nothing, so the snapshot's index stays.
	_, ok = d.Lookup(docs.ConfigStack, docs.KindBlock, "unit")
	assert.True(t, ok)

	// A fresh cache needs no fetch, and a new instance starts from it.
	d.Refresh(t.Context())
	d.Wait()
	assert.Equal(t, 1, s.count("/reference/hcl/units.md"))

	opts.Client = nil
	offline := newDocs(t, opts)

	_, ok = offline.Lookup(docs.ConfigUnit, docs.KindFunction, "get_platform")
	assert.True(t, ok)
}
