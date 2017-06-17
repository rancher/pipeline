package storer

//Storer interface defines methods to manipulate ci-files including pipeline files and step logs
type Storer interface {
	//Get Storer name
	GetName() string
	//Get lastest version of the pipeline
	GetLatestVersion(pipelinepath string) int
	//Save pipeline file, it will generate a newest version of the pipeline
	SavePipelineFile(pipelinePath string, content string) error
	//read a specific version pipeline file
	ReadPipelineFile(pipelinePath string, version string) (string, error)
	//read the latest pipeline file
	ReadLatestPipelineFile(pipePath string) (string, error)
	//save log file
	SaveLogFile(pipelinePath string, version string, stage string, step string, content string) error
	//read log file
	ReadLogFile(pipelinePath string, version string, stage string, step string) (string, error)
}

const (
	BasePipelinePath = "/var/tmp/pipelines" //"/var/lib/rancher/pipelines"
)
