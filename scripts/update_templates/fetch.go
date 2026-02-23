package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

func fetchComposeFile(client *http.Client, githubToken string, repoSpec string) (io.Reader, error) {
	repo, ref := parseRepoSpec(repoSpec)

	base, err := url.Parse("https://api.github.com")
	if err != nil {
		return nil, err
	}

	base.Path = path.Join("repos", repo, "contents", "compose.yaml")

	if ref != "" {
		q := base.Query()
		q.Set("ref", ref)
		base.RawQuery = q.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "topo-template-update")
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := client.Do(req) // #nosec
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("compose.yaml not found (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	yamlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	yamlReader := bytes.NewReader(yamlBytes)

	return yamlReader, nil
}
