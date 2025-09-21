package mock

import "astera"

type Repository struct {
	InsertModuleFn func(module *astera.Module) error

	GetVersionListFn func(name string) ([]string, error)
	GetVersionInfoFn func(name, version string) ([]byte, error)
	GetModFileFn     func(name, version string) ([]byte, error)
	GetModuleZipFn   func(name, version string) ([]byte, error)

	ModuleExistsFn func(name string, version string) (bool, error)
}

func (r *Repository) InsertModule(module *astera.Module) error {
	return r.InsertModuleFn(module)
}

func (r *Repository) GetVersionList(name string) ([]string, error) {
	return r.GetVersionListFn(name)
}

func (r *Repository) GetVersionInfo(name, version string) ([]byte, error) {
	return r.GetVersionInfoFn(name, version)
}

func (r *Repository) GetModFile(name, version string) ([]byte, error) {
	return r.GetModFileFn(name, version)
}

func (r *Repository) GetModuleZip(name, version string) ([]byte, error) {
	return r.GetModuleZipFn(name, version)
}

func (r *Repository) ModuleExists(name string, version string) (bool, error) {
	return r.ModuleExistsFn(name, version)
}
