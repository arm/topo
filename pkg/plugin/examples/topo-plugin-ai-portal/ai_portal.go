package main

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type ModelQuery struct {
	ProjectName  string
	Task         string
	Format       string
	Runtime      string
	Backend      string
	PeakMemoryMB int
}

type BrowserClient struct {
	portalURL   string
	openBrowser func(context.Context, string) error
}

func NewBrowserClient(portalURL string) BrowserClient {
	return BrowserClient{portalURL: portalURL, openBrowser: openBrowser}
}

func (c BrowserClient) OpenModelSelectionPage(ctx context.Context, callbackURL string, query ModelQuery) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	address, err := buildPortalURL(c.portalURL, callbackURL, query)
	if err != nil {
		return err
	}
	if err := c.openBrowser(ctx, address); err != nil {
		return fmt.Errorf("open AI Portal browser: %w", err)
	}
	return nil
}

func buildPortalURL(portalURL, callbackURL string, filters ModelQuery) (string, error) {
	if filters.PeakMemoryMB < 0 {
		return "", fmt.Errorf("peak memory cutoff must not be negative")
	}
	address, err := url.Parse(portalURL)
	if err != nil {
		return "", fmt.Errorf("parse Portal URL: %w", err)
	}
	if (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" ||
		address.User != nil || address.Fragment != "" || address.Opaque != "" {
		return "", fmt.Errorf("Portal URL must be an absolute HTTP(S) URL without credentials or a fragment")
	}
	query, err := url.ParseQuery(address.RawQuery)
	if err != nil {
		return "", fmt.Errorf("parse Portal URL query: %w", err)
	}
	for key, value := range map[string]string{
		"task": filters.Task, "format": filters.Format, "runtime": filters.Runtime,
		"runtimeBackend":  filters.Backend,
		"topoProjectName": strings.TrimSpace(filters.ProjectName),
	} {
		query.Del(key)
		if value != "" {
			query.Set(key, value)
		}
	}
	query.Del("peakMemoryMb")
	if filters.PeakMemoryMB > 0 {
		query.Set("peakMemoryMb", strconv.Itoa(filters.PeakMemoryMB))
	}
	query.Set("topoCallback", callbackURL)
	address.RawQuery = query.Encode()
	return address.String(), nil
}

func openBrowser(ctx context.Context, address string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "open", address).Run()
	case "windows":
		return exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", address).Run()
	default:
		return exec.CommandContext(ctx, "xdg-open", address).Run()
	}
}
