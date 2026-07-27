package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const timeout = 2 * time.Minute

// Get retrieves the contents of an HTTP or HTTPS URL.
func Get(ctx context.Context, rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// #nosec G704 -- callers explicitly provide the URL and its scheme is validated above.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		statusErr := fmt.Errorf("request failed: HTTP %d (%s)", response.StatusCode, response.Status)
		return nil, errors.Join(statusErr, response.Body.Close())
	}

	data, readErr := io.ReadAll(response.Body)
	if err := errors.Join(readErr, response.Body.Close()); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return data, nil
}
