package docs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
)

const (
	// MaxPageBytes is the largest page [Fetch] reads.
	MaxPageBytes = 1 << 20
	// MaxETagBytes is the longest ETag [Fetch] returns. It drops a longer one.
	MaxETagBytes = 256
)

var (
	// ErrPageTooLarge is returned by [Fetch] for a page larger than [MaxPageBytes].
	ErrPageTooLarge = fmt.Errorf("docs page is larger than %d bytes", MaxPageBytes)
	// ErrNotMarkdown is returned by [Fetch] for a response that isn't
	// text/markdown, such as an HTML error page.
	ErrNotMarkdown = errors.New("docs page is not Markdown")
	// ErrNotModified is returned by [Fetch] when the page still has the ETag
	// it was given.
	ErrNotModified = errors.New("docs page is not modified")
)

// StatusError is returned by [Fetch] for a response with a status other than
// 200 OK or 304 Not Modified.
type StatusError struct {
	URL        string
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("fetching %s: %s", e.URL, http.StatusText(e.StatusCode))
}

// Page is a page that [Fetch] returns.
type Page struct {
	// ETag identifies this version of the page, or is empty when the
	// response had none or one longer than [MaxETagBytes] or with characters
	// an ETag can't hold.
	ETag string
	Body []byte
}

// Fetch returns the Markdown page at path on the docs site at base, e.g.
// `/reference/hcl/units.md`. With a non-empty etag, it asks for the page only
// if it no longer has that ETag.
//
// Returns [ErrNotModified] when the page still has etag, a [*StatusError] for
// any other status but 200 OK, [ErrNotMarkdown] for a response that isn't
// text/markdown, and [ErrPageTooLarge] for a body larger than [MaxPageBytes].
func Fetch(ctx context.Context, c *http.Client, base *url.URL, path, etag string) (page Page, err error) {
	target := base.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return Page{}, err
	}

	req.Header.Set("Accept", "text/markdown")

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.Do(req)
	if err != nil {
		return Page{}, redactURLError(err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			page, err = Page{}, closeErr
		}
	}()

	if resp.StatusCode == http.StatusNotModified && etag != "" {
		return Page{}, ErrNotModified
	}

	if resp.StatusCode != http.StatusOK {
		return Page{}, &StatusError{URL: target.Redacted(), StatusCode: resp.StatusCode}
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/markdown" {
		return Page{}, ErrNotMarkdown
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxPageBytes+1))
	if err != nil {
		return Page{}, err
	}

	if len(body) > MaxPageBytes {
		return Page{}, ErrPageTooLarge
	}

	return Page{Body: body, ETag: validETag(resp.Header.Get("ETag"))}, nil
}

// validETag returns etag when it is short enough to cache and holds only the
// visible ASCII characters an ETag can, and an empty string otherwise.
func validETag(etag string) string {
	if len(etag) > MaxETagBytes {
		return ""
	}

	for i := range len(etag) {
		if etag[i] < '!' || etag[i] > '~' {
			return ""
		}
	}

	return etag
}

// redactURLError removes credentials from the URL of a *url.Error.
func redactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}

	if u, parseErr := url.Parse(urlErr.URL); parseErr == nil {
		urlErr.URL = u.Redacted()
	}

	return urlErr
}
