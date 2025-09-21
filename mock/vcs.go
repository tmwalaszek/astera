package mock

import "astera"

type VCS struct {
	CloneFn     func(repo string, tag string) (*astera.Module, error)
	FetchTagsFn func(repo string) ([]string, error)
}

func (v *VCS) Clone(repo string, tag string) (*astera.Module, error) {
	return v.CloneFn(repo, tag)
}

func (v *VCS) FetchTags(repo string) ([]string, error) {
	return v.FetchTagsFn(repo)
}
