package modstore

import (
	"astera"
	"astera/mock"
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmwalaszek/weakcache"
)

// RoundTripper mock
type mockRoundTripper func(req *http.Request) *http.Response

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req), nil
}

func TestQueryInvalid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repositoryMock := &mock.Repository{}
	vcsMock := &mock.VCS{}

	weakCache := weakcache.NewWeakCache[[]byte]()

	proxyCache := &ModuleStore{
		moduleRepository: repositoryMock,
		vcs:              vcsMock,
		weakCache:        weakCache,
	}

	var tt = []struct {
		query string
	}{
		{
			query: "github.com/@v",
		},
		{
			query: "github.com/slk/@v",
		},
		{
			query: "github.com/tmwalaszek/module1/v1.0.0",
		},
		{
			query: "github.com/tmwalaszek/module2/@latest/asf",
		},
		{
			query: "github.com/tmwalaszek/module3/@v/@latest",
		},
		{
			query: "github.com/tmwalaszek/module4/@v/@latest/asf",
		},
	}

	for _, tc := range tt {
		t.Run(tc.query, func(t *testing.T) {
			_, err := proxyCache.Query(ctx, tc.query)
			assert.Error(t, err)
		})
	}
}

func TestQueryInfoModZip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repositoryMock := &mock.Repository{}
	vcsMock := &mock.VCS{}

	weakCache := weakcache.NewWeakCache[[]byte]()

	proxyCache := &ModuleStore{
		goProxyClient:    &GoProxyClient{},
		moduleRepository: repositoryMock,
		vcs:              vcsMock,
		weakCache:        weakCache,
	}

	var tt = []struct {
		query        string
		zipResponse  []byte
		modResponse  []byte
		infoResponse []byte
		code         int
		err          error
		dbErr        error
	}{
		{
			query:       "github.com/tmwalaszek/module1/@v/v1.0.0.zip",
			zipResponse: []byte("zip content"),
		},
		{
			query:        "github.com/tmwalaszek/module1/@v/v1.0.0.info",
			infoResponse: []byte("info content"),
		},
		{
			query:       "github.com/tmwalaszek/module1/@v/v1.0.0.mod",
			modResponse: []byte("mod content"),
		},
		{
			query:        "github.com/tmwalaszek/module2/@v/v1.0.0.zip",
			dbErr:        astera.ErrModuleNotFound,
			zipResponse:  []byte("zip content"),
			modResponse:  []byte("mod response"),
			infoResponse: []byte(`{"Version":"v1.1.0","Time":"2025-09-12T21:00:38Z","Origin":{"VCS":"git","URL":"https://github.com/tmwalaszek/module2/@v/v1.0.0.zip","Hash":"91adcceaf87d787834a6afb185514e0cb59ff917","Ref":"refs/tags/v1.1.0"}}`),
			code:         http.StatusOK,
			err:          nil,
		},
		{
			query:        "github.com/tmwalaszek/module2/@v/v1.0.0.info",
			dbErr:        astera.ErrModuleNotFound,
			zipResponse:  []byte("zip content"),
			modResponse:  []byte("mod response"),
			infoResponse: []byte(`{"Version":"v1.1.0","Time":"2025-09-12T21:00:38Z","Origin":{"VCS":"git","URL":"https://github.com/tmwalaszek/module2/@v/v1.0.0.zip","Hash":"91adcceaf87d787834a6afb185514e0cb59ff917","Ref":"refs/tags/v1.1.0"}}`),
			code:         http.StatusOK,
			err:          nil,
		},
		{
			query:        "github.com/tmwalaszek/module2/@v/v1.0.0.mod",
			dbErr:        astera.ErrModuleNotFound,
			zipResponse:  []byte("zip content"),
			modResponse:  []byte("mod response"),
			infoResponse: []byte(`{"Version":"v1.1.0","Time":"2025-09-12T21:00:38Z","Origin":{"VCS":"git","URL":"https://github.com/tmwalaszek/module2/@v/v1.0.0.zip","Hash":"91adcceaf87d787834a6afb185514e0cb59ff917","Ref":"refs/tags/v1.1.0"}}`),
			code:         http.StatusOK,
			err:          nil,
		},
		{
			query: "github.com/tmwalaszek/module3/@v/v2.0.0.zip",
			dbErr: astera.ErrModuleNotFound,
			code:  http.StatusNotFound,
			err:   astera.ErrModuleNotFound,
		},
	}

	for _, tc := range tt {
		t.Run(tc.query, func(t *testing.T) {
			repositoryMock.GetModuleZipFn = func(name string, version string) ([]byte, error) {
				if tc.dbErr != nil {
					return nil, tc.dbErr
				}

				return tc.zipResponse, nil
			}

			repositoryMock.GetVersionInfoFn = func(name string, version string) ([]byte, error) {
				if tc.dbErr != nil {
					return nil, tc.dbErr
				}

				return tc.infoResponse, nil
			}

			repositoryMock.GetModFileFn = func(name string, version string) ([]byte, error) {
				if tc.dbErr != nil {
					return nil, tc.dbErr
				}

				return tc.modResponse, nil
			}

			repositoryMock.InsertModuleFn = func(m *astera.Module) error {
				return nil
			}

			repositoryMock.ModuleExistsFn = func(module string, version string) (bool, error) {
				return false, nil
			}

			mockTransport := mockRoundTripper(func(req *http.Request) *http.Response {
				if req.URL.Path == "/github.com/tmwalaszek/module2/@v/v1.1.0.zip" {
					return &http.Response{
						StatusCode: tc.code,
						Body:       io.NopCloser(bytes.NewBuffer(tc.zipResponse)),
						Header:     make(http.Header),
					}
				} else if req.URL.Path == "/github.com/tmwalaszek/module2/@v/v1.1.0.mod" {
					return &http.Response{
						StatusCode: tc.code,
						Body:       io.NopCloser(bytes.NewBuffer(tc.modResponse)),
						Header:     make(http.Header),
					}
				} else {
					return &http.Response{
						StatusCode: tc.code,
						Body:       io.NopCloser(bytes.NewBuffer(tc.infoResponse)),
						Header:     make(http.Header),
					}
				}
			})

			mockClient := &http.Client{Transport: mockTransport}

			proxyCache.goProxyClient.client = mockClient

			_, err := proxyCache.Query(ctx, tc.query)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}

			assert.NoError(t, err)
		})
	}

}

func TestQueryList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repositoryMock := &mock.Repository{}
	vcsMock := &mock.VCS{}

	weakCache := weakcache.NewWeakCache[[]byte]()

	proxyCache := &ModuleStore{
		moduleRepository: repositoryMock,
		vcs:              vcsMock,
		weakCache:        weakCache,
	}

	var tt = []struct {
		query    string
		response []string
		err      error
	}{
		{
			query:    "github.com/tmwalaszek/module1/@v/list",
			response: []string{"v1.0.0", "v1.0.1"},
		},
		{
			query:    "github.com/tmwalaszek/module3/@v/list",
			response: []string{},
			err:      astera.ErrModuleNotFound,
		},
	}

	for _, tc := range tt {
		t.Run(tc.query, func(t *testing.T) {
			repositoryMock.GetVersionListFn = func(module string) ([]string, error) {
				if tc.err != nil {
					return nil, tc.err
				}

				return tc.response, nil
			}

			_, err := proxyCache.Query(ctx, tc.query)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestQueryLatest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	repositoryMock := &mock.Repository{}
	vcsMock := &mock.VCS{}

	weakCache := weakcache.NewWeakCache[[]byte]()

	proxyCache := &ModuleStore{
		goProxyClient:    &GoProxyClient{},
		moduleRepository: repositoryMock,
		vcs:              vcsMock,
		weakCache:        weakCache,
	}

	var tt = []struct {
		query    string
		module   string
		response string
		code     int
	}{
		{
			query:    "github.com/tmwalaszek/module1/@latest",
			module:   "github.com/tmwalaszek/module1",
			response: `{"Version":"v0.0.1","Time":"2025-05-24T17:42:06Z"}%`,
			code:     http.StatusOK,
		},
		{
			query:  "github.com/tmwalaszek/module2/@latest",
			module: "github.com/tmwalaszek/module2",
			code:   http.StatusNotFound,
		},
		{
			query:  "github.com/tmwalaszek/module3/@latest",
			module: "github.com/tmwalaszek/module3",
			code:   http.StatusGone,
		},
		{
			query:  "github.com/tmwalaszek/module4/@latest",
			module: "github.com/tmwalaszek/module4",
			code:   http.StatusInternalServerError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.query, func(t *testing.T) {
			mockTransport := mockRoundTripper(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: tc.code,
					Body:       io.NopCloser(bytes.NewBufferString(tc.response)),
					Header:     make(http.Header),
				}
			})

			mockClient := &http.Client{Transport: mockTransport}

			proxyCache.goProxyClient.client = mockClient

			d, err := proxyCache.Query(ctx, tc.query)
			if tc.code != http.StatusOK {
				if tc.code == http.StatusNotFound || tc.code == http.StatusGone {
					assert.EqualError(t, err, astera.ErrModuleNotFound.Error())
					return
				}

				assert.EqualError(t, err, "request failed with status code 500")
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.response, string(d))
		})
	}
}
