package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func fetchComposeFile(client *http.Client, githubToken string, repoSpec string) (io.Reader, error) {
	repo, ref := parseRepoSpec(repoSpec)

	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/compose.yaml", orgName, repo)

	if ref != "" {
		v := url.Values{}
		v.Set("ref", ref)
		u = u + "?" + v.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "topo-cli-template-update")
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := client.Do(req)
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
