package storer

const (
	GitStorerName = "git"
)

type GitStorer struct {
	LocalStorer
}

func (g *GitStorer) GetName() string {
	return GitStorerName
}

//SavePipelineFile save pipeline file with the content to a new version folder in the path
func (g *GitStorer) SavePipelineFile(pipelinePath string, content string) error {
	return g.LocalStorer.SavePipelineFile(pipelinePath, content)
	//TODO
	//git operations
}

//ReadPipelineFile read pipeline file in the path with specific version
func (g *GitStorer) ReadPipelineFile(pipelinePath string, version string) (string, error) {
	return g.LocalStorer.ReadPipelineFile(pipelinePath, version)
	//TODO
	//git operations
}

//ReadPipelineFile read pipeline file in the path with specific version
func (g *GitStorer) ReadLatestPipelineFile(pipelinePath string) (string, error) {
	return g.LocalStorer.ReadLatestPipelineFile(pipelinePath)
	//TODO
	//git operations
}

//SaveLogFile saves step log file in "stagename_stepname.log" under pipeline_folder/logs
func (g *GitStorer) SaveLogFile(pipelinePath string, version string, stageName string, stepName string, content string) error {
	return g.LocalStorer.SaveLogFile(pipelinePath, version, stageName, stepName, content)
	//TODO
	//git operations
}

//ReadLogFile reads log file from pipeline path
func (g *GitStorer) ReadLogFile(pipelinePath string, version string, stageName string, stepName string) (string, error) {
	return g.LocalStorer.ReadLogFile(pipelinePath, version, stageName, stepName)
	//TODO
	//git operations
}

//GetLatestVersion gets latest pipeline file version in the pipeline path, return -1 if non valid version exists
func (g *GitStorer) GetLatestVersion(pipelinePath string) int {
	return g.LocalStorer.GetLatestVersion(pipelinePath)
	//TODO
	//git operations
}
