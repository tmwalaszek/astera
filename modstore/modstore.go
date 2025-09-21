package modstore

import (
	"astera"
	"astera/git"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tmwalaszek/weakcache"
	xmod "golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	infoSuffix    = ".info"
	modSuffix     = ".mod"
	zipSuffix     = ".zip"
	zipHashSuffix = ".ziphash"
)

type ModuleStore struct {
	moduleRepository astera.ModuleRepository
	vcs              astera.VCS

	goProxyClient *GoProxyClient

	goPrivate string

	weakCache *weakcache.WeakCache[[]byte]
}

func NewModuleStore(moduleRepository astera.ModuleRepository) astera.GoProxyService {
	goProxyClient := NewGoProxyClient()
	newWeakCache := weakcache.NewWeakCache[[]byte]()
	vcs := git.New()

	goPrivate := os.Getenv("GOPRIVATE")

	return &ModuleStore{moduleRepository: moduleRepository,
		weakCache:     newWeakCache,
		goProxyClient: goProxyClient,
		goPrivate:     goPrivate,
		vcs:           vcs,
	}
}

func (c *ModuleStore) ImportCachedModules(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.New("module cache directory does not exist")
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && info.Name() == "@v" {
			err = c.createModuleCache(dir, path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		return nil
	})

	return err
}

func (c *ModuleStore) Query(ctx context.Context, query string) ([]byte, error) {
	var resource, module string
	var err error

	var responseBody []byte

	query = strings.TrimPrefix(query, "/")

	if strings.HasSuffix(query, "/@latest") {
		resource = "@latest"
		module = strings.TrimSuffix(query, "/@latest")
	} else if strings.HasSuffix(query, "/list") {
		resource = "list"
		module = strings.TrimSuffix(query, "/@v/list")
	} else {
		split := strings.Split(query, "/@v/")
		if len(split) != 2 {
			return nil, fmt.Errorf("%w: query %s", astera.ErrInvalidResource, query)
		}
		module = split[0]
		resource = split[1]
	}

	_, err = xmod.UnescapePath(module)
	if err != nil {
		return nil, err
	}

	switch {
	case resource == "list":
		var versionLists []string
		versionLists, err = c.queryVersionsList(module)

		responseBody = []byte(strings.Join(versionLists, "\n"))
	case resource == "@latest":
		var latest string
		latest, err = c.queryLatest(ctx, module)

		responseBody = []byte(latest)
	case strings.HasSuffix(resource, infoSuffix):
		ver := strings.TrimSuffix(resource, infoSuffix)
		_, err = xmod.UnescapeVersion(ver)
		if err != nil {
			return nil, err
		}

		responseBody, err = c.queryModuleInfo(ctx, module, ver)
	case strings.HasSuffix(resource, modSuffix):
		ver := strings.TrimSuffix(resource, modSuffix)
		_, err = xmod.UnescapeVersion(ver)
		if err != nil {
			return nil, err
		}

		responseBody, err = c.queryModuleMod(ctx, module, ver)
	case strings.HasSuffix(resource, zipSuffix):
		ver := strings.TrimSuffix(resource, zipSuffix)
		_, err = xmod.UnescapeVersion(ver)
		if err != nil {
			return nil, err
		}

		responseBody, err = c.queryModuleZip(ctx, module, ver)
	default:
		return nil, fmt.Errorf("%w: query %s", astera.ErrInvalidResource, query)
	}

	if err != nil {
		return nil, err
	}

	return responseBody, nil
}

func (c *ModuleStore) createModuleCache(dir, modulePath string) error {
	module := strings.TrimPrefix(modulePath, dir+"/")
	module = strings.TrimSuffix(module, "/@v")

	listBody, err := os.ReadFile(path.Join(modulePath, "list"))
	if err != nil {
		return err
	}

	list := strings.Split(string(listBody), "\n")
	for _, v := range list {
		if len(v) == 0 {
			continue
		}

		version := strings.TrimSpace(v)
		var infoFile, zip, zipHash []byte

		modFilePath := path.Join(modulePath, version+modSuffix)
		modFile, err := os.ReadFile(modFilePath)
		if err != nil {
			return err
		}

		infoFilePath := path.Join(modulePath, version+infoSuffix)
		_, err = os.Stat(infoFilePath)
		if !os.IsNotExist(err) {
			infoFile, err = os.ReadFile(infoFilePath)
			if err != nil {
				return err
			}
		}

		zipFilePath := path.Join(modulePath, version+zipSuffix)
		_, err = os.Stat(zipFilePath)
		if !os.IsNotExist(err) {
			zip, err = os.ReadFile(zipFilePath)
			if err != nil {
				return err
			}
		}

		zipHashFilePath := path.Join(modulePath, version+zipHashSuffix)
		_, err = os.Stat(zipHashFilePath)
		if !os.IsNotExist(err) {
			zipHash, err = os.ReadFile(zipHashFilePath)
			if err != nil {
				return err
			}
		}

		m := &astera.Module{
			Name:    module,
			Version: version,
			Info:    infoFile,
			Mod:     modFile,
			ZipHash: string(zipHash),
			Zip:     zip,
		}

		err = c.moduleRepository.InsertModule(m)
		if err != nil && !errors.Is(err, astera.ErrModuleAlreadyExists) {
			return err
		}
	}

	return nil
}

func (c *ModuleStore) queryVersionsList(module string) ([]string, error) {
	var versionList []string
	var err error

	if xmod.MatchPrefixPatterns(c.goPrivate, module) {
		module, err = xmod.UnescapePath(module)
		if err != nil {
			return nil, err
		}

		versionList, err = c.vcs.FetchTags(module)
		if err != nil {
			return nil, err
		}
	} else {
		versionList, err = c.moduleRepository.GetVersionList(module)
		if err != nil {
			return nil, err
		}
	}

	return versionList, nil
}

func (c *ModuleStore) queryLatest(ctx context.Context, module string) (string, error) {
	var latest string

	if xmod.MatchPrefixPatterns(c.goPrivate, module) {
		module, err := xmod.UnescapePath(module)
		if err != nil {
			return "", err
		}

		tagLists, err := c.vcs.FetchTags(module)
		if err != nil {
			return "", err
		}

		semver.Sort(tagLists)
		latest = tagLists[len(tagLists)-1]
	} else {
		tag, err := c.goProxyClient.FetchLatest(ctx, module)
		if err != nil {
			return "", err
		}

		latest = string(tag)
	}

	return latest, nil
}

func (c *ModuleStore) queryModuleInfo(ctx context.Context, module, version string) ([]byte, error) {
	result, err := c.queryModuleInfoCache(module, version)
	if err == nil {
		return result, nil
	}

	if errors.Is(err, astera.ErrModuleNotFound) {
		err := c.fetchAndSetModule(ctx, module, version)
		if err != nil {
			return nil, err
		}

		return c.queryModuleInfoCache(module, version)

	}

	return nil, err
}

func (c *ModuleStore) queryModuleMod(ctx context.Context, module, version string) ([]byte, error) {
	result, err := c.queryModuleModCache(module, version)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, astera.ErrModuleNotFound) {
		err := c.fetchAndSetModule(ctx, module, version)
		if err != nil {
			return nil, err
		}

		return c.queryModuleModCache(module, version)
	}

	return nil, err
}

func (c *ModuleStore) queryModuleZip(ctx context.Context, module, version string) ([]byte, error) {
	result, err := c.queryModuleZipCache(module, version)
	if err == nil {
		return result, nil
	}

	if errors.Is(err, astera.ErrModuleNotFound) {
		err := c.fetchAndSetModule(ctx, module, version)
		if err != nil {
			return nil, err
		}

		return c.queryModuleZipCache(module, version)
	}

	return nil, err
}

func cacheKey(module, version, suffix string) string {
	return fmt.Sprintf("%s-%s%s", module, version, suffix)
}

func (c *ModuleStore) fetchAndCache(ctx context.Context, module, version, suffix string,
	fetchFn func(context.Context, string, string) ([]byte, error),
) ([]byte, error) {
	key := cacheKey(module, version, suffix)
	return c.weakCache.Do(key, func() ([]byte, error) {
		return fetchFn(ctx, module, version)
	})
}

// resource is version + {info,mod,zip}
func (c *ModuleStore) fetchModule(ctx context.Context, module, version string) (*astera.Module, error) {
	info, err := c.fetchAndCache(ctx, module, version, infoSuffix, c.goProxyClient.FetchModuleInfo)
	if err != nil {
		return nil, err
	}

	mod, err := c.fetchAndCache(ctx, module, version, modSuffix, c.goProxyClient.FetchModuleMod)
	if err != nil {
		return nil, err
	}

	zip, err := c.fetchAndCache(ctx, module, version, zipSuffix, c.goProxyClient.FetchModuleZip)
	if err != nil {
		return nil, err
	}

	var infoResponse astera.Info
	err = json.Unmarshal(info, &infoResponse)
	if err != nil {
		return nil, err
	}

	return &astera.Module{
		Name:    module,
		Version: version,
		Info:    info,
		Mod:     mod,
		Zip:     zip,
		ZipHash: infoResponse.Origin.Hash,
	}, nil
}

// fetchModule
func (c *ModuleStore) fetchAndSetModule(ctx context.Context, module, version string) error {
	moduleExists, err := c.moduleRepository.ModuleExists(module, version)
	if err != nil {
		return err
	}

	if moduleExists {
		return nil
	}

	var m *astera.Module
	if xmod.MatchPrefixPatterns(c.goPrivate, module) {
		module, err := xmod.UnescapePath(module)
		if err != nil {
			return err
		}

		version, err = xmod.UnescapeVersion(version)
		if err != nil {
			return err
		}

		m, err = c.vcs.Clone(module, version)
		if err != nil {
			return err
		}
	} else {
		m, err = c.fetchModule(ctx, module, version)
		if err != nil {
			return err
		}
	}

	err = c.moduleRepository.InsertModule(m)
	if err != nil {
		return err
	}

	return nil
}

func (c *ModuleStore) queryRepository(module, version string, modulePostfix string, repositoryGetFn func(string, string) ([]byte, error)) ([]byte, error) {
	result, err := repositoryGetFn(module, version)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ModuleStore) queryModuleInfoCache(module, version string) ([]byte, error) {
	result, err := c.weakCache.Do(fmt.Sprintf("%s-%s%s", module, version, infoSuffix),
		func() ([]byte, error) {
			return c.queryRepository(module, version, "info", c.moduleRepository.GetVersionInfo)
		})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ModuleStore) queryModuleModCache(module, version string) ([]byte, error) {
	result, err := c.weakCache.Do(fmt.Sprintf("%s-%s%s", module, version, modSuffix),
		func() ([]byte, error) {
			return c.queryRepository(module, version, "mod", c.moduleRepository.GetModFile)
		})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ModuleStore) queryModuleZipCache(module, version string) ([]byte, error) {
	result, err := c.weakCache.Do(fmt.Sprintf("%s-%s%s", module, version, zipSuffix),
		func() ([]byte, error) {
			return c.queryRepository(module, version, "zip", c.moduleRepository.GetModuleZip)
		})
	if err != nil {
		return nil, err
	}

	return result, nil
}
