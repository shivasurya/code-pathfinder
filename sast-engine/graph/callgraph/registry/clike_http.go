package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// httpFetchTimeout is the upper bound on a single HTTP request. The 30-second
// value matches the Go stdlib loader; CDN responses are typically <100ms but
// flaky networks need slack to avoid spurious cache misses.
//
// The constant lives at package scope (not on the loader struct) so test
// servers and benchmarks can rely on a deterministic value.

// fetchURL performs a single HTTP GET and returns the response body. Non-200
// responses are turned into errors so callers can branch on a single failure
// path. The caller-supplied client carries timeout + retry policy.
//
// Defined at package scope so both the C and C++ remote loaders share the
// same wire-format behavior — the only thing that differs between them is the
// URL path (`/c/v1/` vs `/cpp/v1/`).
func fetchURL(client *http.Client, url string) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("fetchURL: nil HTTP client")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchURL: building request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchURL: GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchURL: reading body from %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchURL: GET %s: HTTP %d", url, resp.StatusCode)
	}
	return body, nil
}

// verifyChecksum confirms that data hashes to the expected sha256 digest.
// expected is the manifest's `checksum` field, expected to be of the form
// "sha256:<hex>". An empty expected string disables checksum verification —
// useful for manifests generated before the checksum field was populated, and
// for tests that don't want to compute hashes by hand.
//
// Returns an error tagged with both expected + actual so log lines stay
// useful for diagnosing CDN tampering or stale-cache + new-checksum drift.
func verifyChecksum(data []byte, expected string) error {
	if expected == "" {
		return nil
	}
	if !strings.HasPrefix(expected, "sha256:") {
		return fmt.Errorf("verifyChecksum: unsupported checksum format %q (want sha256:<hex>)", expected)
	}
	want := strings.TrimPrefix(expected, "sha256:")
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("verifyChecksum: digest mismatch (want %s, got %s)", want, got)
	}
	return nil
}

// joinURL concatenates a base URL with one or more path segments using a
// single forward slash as separator. Stripping leading/trailing slashes on
// the input avoids the duplicate-slash artifacts a naive join would emit
// (e.g. "https://x/" + "/foo" → "https://x//foo").
func joinURL(base string, segments ...string) string {
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, strings.TrimRight(base, "/"))
	for _, s := range segments {
		s = strings.Trim(s, "/")
		if s == "" {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "/")
}
