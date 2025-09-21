package astera

import (
	"context"
	"errors"
)

var (
	ErrModuleAlreadyExists = errors.New("module already exists")
	ErrModuleNotFound      = errors.New("module not found")
	ErrInvalidResource     = errors.New("invalid resource")
)

type Module struct {
	Name string

	Version string
	ZipHash string

	Info []byte
	Mod  []byte
	Zip  []byte
}

type Info struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
	Origin  Origin `json:"Origin"`
}

type Origin struct {
	VCS  string `json:"VCS"`
	URL  string `json:"URL"`
	Hash string `json:"Hash"`
	Ref  string `json:"Ref"`
}

type ModuleRepository interface {
	InsertModule(module *Module) error

	GetVersionList(name string) ([]string, error)
	GetVersionInfo(name, version string) ([]byte, error)
	GetModFile(name, version string) ([]byte, error)
	GetModuleZip(name, version string) ([]byte, error)

	ModuleExists(name string, version string) (bool, error)
}

type GoProxyService interface {
	ImportCachedModules(dir string) error
	Query(context.Context, string) ([]byte, error)
}

type VCS interface {
	Clone(repo string, tag string) (*Module, error)
	FetchTags(repo string) ([]string, error)
}
