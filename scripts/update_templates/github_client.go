package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

type GitHubClient struct {
	httpClient *http.Client
	token      string
}

func NewGitHubClient(token string) GitHubClient {
	return GitHubClient{
		httpClient: http.DefaultClient,
		token:      token,
	}
}

func (c GitHubClient) FetchFile(source GitHubSource, repoFilePath string) ([]byte, error) {
	// #nosec G704 -- URL is constructed from static GitHub source metadata.
	req, err := http.NewRequest(http.MethodGet, c.fileURL(source, repoFilePath), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "topo-template-update")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	// #nosec G704 -- URL is constructed from static GitHub source metadata.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s not found (status %d)", repoFilePath, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return content, nil
}

func (c GitHubClient) fileURL(source GitHubSource, repoFilePath string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   path.Join("repos", source.Repo, "contents", repoFilePath),
	}
	q := u.Query()
	q.Set("ref", source.SHA)
	u.RawQuery = q.Encode()
	return u.String()
}
