package storer

//Storer interface defines methods to manipulate ci-files including pipeline files and step logs
type Storer interface {
	GetName() string
	GetLatestVersion(pipelinepath string) int
	SavePipelineFile(pipelinePath string, content string) error
	ReadLatestPipelineFile(pipePath string) (string, error)
	SaveLogFile(pipelinePath string, stage string, step string, content string) error
	ReadLogFile(pipelinePath string, stage string, step string) (string, error)
}

const (
	BasePipelinePath = "/var/tmp/pipelines" //"/var/lib/rancher/pipelines"
)
