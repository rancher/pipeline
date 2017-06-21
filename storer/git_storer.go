package storer

import (
	"os"
	"path/filepath"

	git "github.com/rancher/pipeline/git"
)

const (
	GitStorerName = "git"
)

type GitStorer struct {
	LocalStorer
	RepoURL string
	Branch  string
}

func (g *GitStorer) GetName() string {
	return GitStorerName
}

func InitializeGitStorer(repo string, branch string) *GitStorer {
	return &GitStorer{
		RepoURL: repo,
		Branch:  branch,
	}
}

func (g *GitStorer) getRepo(pipelinePath string) error {
	path := filepath.Join(BasePipelinePath, pipelinePath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
		err = git.Init(path, g.RepoURL)
		if err != nil {
			return err
		}
	}
	return nil
}

//SavePipelineFile save pipeline file with the content to a new version folder in the path
func (g *GitStorer) SavePipelineFile(pipelinePath string, content string) error {
	err := g.getRepo(pipelinePath)
	if err != nil {
		return err
	}
	err = g.LocalStorer.SavePipelineFile(pipelinePath, content)
	if err != nil {
		return err
	}

	path := filepath.Join(BasePipelinePath, pipelinePath)
	err = git.LazyPush(path, g.RepoURL, g.Branch)

	return err

}

//ReadPipelineFile read pipeline file in the path with specific version
func (g *GitStorer) ReadPipelineFile(pipelinePath string, version string) (string, error) {
	return g.LocalStorer.ReadPipelineFile(pipelinePath, version)
}

//ReadPipelineFile read pipeline file in the path with specific version
func (g *GitStorer) ReadLatestPipelineFile(pipelinePath string) (string, error) {
	return g.LocalStorer.ReadLatestPipelineFile(pipelinePath)
}

//SaveLogFile saves step log file in "stagename_stepname.log" under pipeline_folder/logs
func (g *GitStorer) SaveLogFile(pipelinePath string, version string, stageName string, stepName string, content string) error {
	err := g.LocalStorer.SaveLogFile(pipelinePath, version, stageName, stepName, content)
	if err != nil {
		return err
	}
	path := filepath.Join(BasePipelinePath, pipelinePath)
	err = git.LazyPush(path, g.RepoURL, g.Branch)

	return err
}

//ReadLogFile reads log file from pipeline path
func (g *GitStorer) ReadLogFile(pipelinePath string, version string, stageName string, stepName string) (string, error) {
	return g.LocalStorer.ReadLogFile(pipelinePath, version, stageName, stepName)
}

//GetLatestVersion gets latest pipeline file version in the pipeline path, return -1 if non valid version exists
func (g *GitStorer) GetLatestVersion(pipelinePath string) int {
	return g.LocalStorer.GetLatestVersion(pipelinePath)
}
