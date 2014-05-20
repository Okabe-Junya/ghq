package main

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/motemen/ghq/utils"
)

type LocalRepository struct {
	FullPath  string
	RelPath   string
	PathParts []string
}

func LocalRepositoryFromFullPath(fullPath string) (*LocalRepository, error) {
	var relPath string

	for _, root := range localRepositoryRoots() {
		if strings.HasPrefix(fullPath, root) == false {
			continue
		}

		var err error
		relPath, err = filepath.Rel(root, fullPath)
		if err == nil {
			break
		}
	}

	if relPath == "" {
		return nil, fmt.Errorf("No local repository found for: %s", fullPath)
	}

	pathParts := strings.Split(relPath, string(filepath.Separator))
	if len(pathParts) != 3 { // host, user, project
		return nil, nil
	}

	return &LocalRepository{fullPath, relPath, pathParts}, nil
}

func LocalRepositoryFromPathParts(pathParts []string) *LocalRepository {
	relPath := path.Join(pathParts...)
	return &LocalRepository{
		path.Join(primaryLocalRepositoryRoot(), relPath),
		relPath,
		pathParts,
	}
}

// List of tail parts of relative path from the root directory (shortest first)
// for example, {"ghq", "motemen/ghq", "github.com/motemen/ghq"} for $root/github.com/motemen/ghq.
func (repo *LocalRepository) Subpaths() []string {
	tails := make([]string, len(repo.PathParts))

	for i, _ := range repo.PathParts {
		tails[i] = strings.Join(repo.PathParts[len(repo.PathParts)-(i+1):], "/")
	}

	return tails
}

func (repo *LocalRepository) NonHostPath() string {
	return strings.Join(repo.PathParts[1:], "/")
}

func (repo *LocalRepository) IsUnderPrimaryRoot() bool {
	return strings.HasPrefix(repo.FullPath, primaryLocalRepositoryRoot())
}

// Checks if any subpath of the local repository equals the query.
func (repo *LocalRepository) Matches(pathQuery string) bool {
	for _, p := range repo.Subpaths() {
		if p == pathQuery {
			return true
		}
	}

	return false
}

// TODO return err
func (repo *LocalRepository) VCS() *VCSBackend {
	var (
		fi  os.FileInfo
		err error
	)

	fi, err = os.Stat(filepath.Join(repo.FullPath, ".git"))
	if err == nil && fi.IsDir() {
		return GitBackend
	}

	fi, err = os.Stat(filepath.Join(repo.FullPath, ".hg"))
	if err == nil && fi.IsDir() {
		return MercurialBackend
	}

	return nil
}

func walkLocalRepositories(callback func(*LocalRepository)) {
	for _, root := range localRepositoryRoots() {
		filepath.Walk(root, func(path string, fileInfo os.FileInfo, err error) error {
			repo, err := LocalRepositoryFromFullPath(path)
			if err != nil {
				return nil
			}

			if repo != nil {
				callback(repo)
				return filepath.SkipDir
			} else {
				return nil
			}
		})
	}

	return
}

var _localRepositoryRoots []string

// Returns local cloned repositories' root.
// Uses the value of `git config ghq.root` or defaults to ~/.ghq.
func localRepositoryRoots() []string {
	if len(_localRepositoryRoots) != 0 {
		return _localRepositoryRoots
	}

	var err error
	_localRepositoryRoots, err = GitConfigAll("ghq.root")
	utils.PanicIf(err)

	if len(_localRepositoryRoots) == 0 {
		usr, err := user.Current()
		utils.PanicIf(err)

		_localRepositoryRoots = []string{path.Join(usr.HomeDir, ".ghq")}
	}

	return _localRepositoryRoots
}

func primaryLocalRepositoryRoot() string {
	return localRepositoryRoots()[0]
}