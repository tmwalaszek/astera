package mock

import "context"

type GoProxyCache struct {
	ImportCachedModulesFn func(dir string) error
	QueryFn               func(ctx context.Context, query string) ([]byte, error)
}

func (c *GoProxyCache) ImportCachedModules(dir string) error {
	return c.ImportCachedModulesFn(dir)
}

func (c *GoProxyCache) Query(ctx context.Context, query string) ([]byte, error) {
	return c.QueryFn(ctx, query)
}
