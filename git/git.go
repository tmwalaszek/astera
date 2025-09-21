package git

import (
	"astera"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/mod/zip"
)

type Git struct {
	GtiBinary string

	tempDir string
}

func New() *Git {
	return &Git{GtiBinary: "git"}
}

func (g *Git) FetchTags(repo string) ([]string, error) {
	repoURL := g.AddPrefixToRepo(repo)

	cmd := exec.Command(
		g.GtiBinary,
		"ls-remote",
		"--tags",
		repoURL,
	)

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w\n%s", cmdErr, out)
	}

	outLines := bytes.Split(out, []byte("\n"))
	tags := make([]string, 0, len(outLines))
	for _, line := range outLines {
		if len(line) == 0 {
			continue
		}

		parts := bytes.Split(line, []byte("\t"))
		if len(parts) != 2 {
			continue
		}

		refParts := bytes.Split(parts[1], []byte("/"))
		if len(refParts) == 0 {
			continue
		}

		tags = append(tags, string(refParts[len(refParts)-1]))
	}

	return tags, nil
}

func (g *Git) Clone(repo, tag string) (*astera.Module, error) {
	repoURL := g.AddPrefixToRepo(repo)

	tempDir, err := os.MkdirTemp(g.tempDir, "module-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	defer os.RemoveAll(tempDir)

	cmd := exec.Command(
		g.GtiBinary,
		"clone",
		"--depth", "1",
		"--branch", tag,
		repoURL,
		tempDir,
	)

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to clone repo: %w\n%s", cmdErr, out)
	}

	cmd = exec.Command(
		g.GtiBinary,
		"-C",
		tempDir,
		"--no-pager",
		"show",
		"-s",
		"--format=%cI",
		tag,
	)

	timeOutput, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to get commit time: %w\n%s", cmdErr, timeOutput)
	}

	timeOutput = bytes.TrimSuffix(timeOutput, []byte("\n"))

	cmd = exec.Command(
		g.GtiBinary,
		"-C",
		tempDir,
		"--no-pager",
		"rev-parse",
		"--symbolic-full-name",
		tag)

	refsNameOutput, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to get refs name: %w\n%s", cmdErr, refsNameOutput)
	}

	refsNameOutput = bytes.TrimSuffix(refsNameOutput, []byte("\n"))

	buf := bytes.NewBuffer(nil)
	err = zip.CreateFromDir(buf, module.Version{Path: repo, Version: tag}, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to zip module: %w", err)
	}

	zipped := buf.Bytes()

	modByte, err := os.ReadFile(path.Join(tempDir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	zipHash, err := dirhash.HashDir(tempDir, repo+"@"+tag, dirhash.DefaultHash)
	if err != nil {
		return nil, fmt.Errorf("failed to hash module: %w", err)
	}

	info := &astera.Info{
		Version: tag,
		Time:    string(timeOutput),
		Origin: astera.Origin{
			VCS:  "git",
			URL:  repoURL,
			Hash: zipHash,
			Ref:  string(refsNameOutput),
		},
	}

	infoBody, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal info: %w", err)
	}

	return &astera.Module{
		Name:    repo,
		Version: tag,
		Info:    infoBody,
		Zip:     zipped,
		Mod:     modByte,
		ZipHash: zipHash,
	}, nil
}

func stripModuleMajorSuffix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	parts := strings.Split(p, "/")
	last := parts[len(parts)-1]
	if len(last) > 1 && last[0] == 'v' {
		if n, err := strconv.Atoi(last[1:]); err == nil && n >= 2 {
			return strings.Join(parts[:len(parts)-1], "/")
		}
	}
	return p
}

func (g *Git) AddPrefixToRepo(repo string) string {
	repo = stripModuleMajorSuffix(repo)
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		return repo
	}

	return fmt.Sprintf("https://%s", repo)
}
