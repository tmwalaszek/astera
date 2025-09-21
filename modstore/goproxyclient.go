package modstore

import (
	"astera"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	proxyGolangURL = "https://proxy.golang.org"
)

type GoProxyClient struct {
	client *http.Client
}

func NewGoProxyClient() *GoProxyClient {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 1 * time.Minute,
	}

	return &GoProxyClient{
		client: c,
	}
}

func (c *GoProxyClient) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, astera.ErrModuleNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *GoProxyClient) FetchLatest(ctx context.Context, module string) ([]byte, error) {
	u, err := url.JoinPath(proxyGolangURL, module, "@latest")
	if err != nil {
		return nil, err
	}

	return c.fetch(ctx, u)
}

func (c *GoProxyClient) FetchModuleMod(ctx context.Context, module, version string) ([]byte, error) {
	u, err := url.JoinPath(proxyGolangURL, module, "@v", version+".mod")
	if err != nil {
		return nil, err
	}

	return c.fetch(ctx, u)
}

func (c *GoProxyClient) FetchModuleZip(ctx context.Context, module, version string) ([]byte, error) {
	u, err := url.JoinPath(proxyGolangURL, module, "@v", version+".zip")
	if err != nil {
		return nil, err
	}

	return c.fetch(ctx, u)
}

func (c *GoProxyClient) FetchModuleInfo(ctx context.Context, module, version string) ([]byte, error) {
	u, err := url.JoinPath(proxyGolangURL, module, "@v", version+".info")
	if err != nil {
		return nil, err
	}

	return c.fetch(ctx, u)
}
