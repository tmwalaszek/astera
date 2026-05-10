package mock

import (
	"astera"
	"context"
)

type GoProxyCache struct {
	ImportCachedModulesFn func(dir string) error
	QueryFn               func(ctx context.Context, query string, opts ...astera.QueryOption) ([]byte, error)
}

func (c *GoProxyCache) ImportCachedModules(dir string) error {
	return c.ImportCachedModulesFn(dir)
}

func (c *GoProxyCache) Query(ctx context.Context, query string, opts ...astera.QueryOption) ([]byte, error) {
	return c.QueryFn(ctx, query, opts...)
}
