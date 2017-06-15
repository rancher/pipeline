package pipeline

import (
	"github.com/Sirupsen/logrus"
)

const StepTypeTask = "task"
const StepTypeCatalog = "catalog"
const StepTypeDeploy = "deploy"

type Pipeline struct {
	Name            string   `json:"name,omitempty" yaml:"name,omitempty"`
	VersionSequence string   `json:"-" yaml:"-"`
	Repository      string   `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch          string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	TargetImage     string   `json:"targetImage,omitempty" yaml:"target-image,omitempty"`
	File            string   `json:"file"`
	Stages          []*Stage `json:"stages,omitempty" yaml:"stages,omitempty"`
}

type Stage struct {
	Name        string  `json:"name,omitempty" yaml:"name,omitempty"`
	NeedApprove bool    `json:"needApprove,omitempty" yaml:"need-approve,omitempty"`
	Steps       []*Step `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Step struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	TaskStep
	CatalogStep
	DeployStep
}

type TaskStep struct {
	Command    string   `json:"command,omitempty" yaml:"command,omitempty"`
	Image      string   `json:"image,omitempty" yaml:"image,omitempty"`
	Parameters []string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type CatalogStep struct {
	DockerCompose  string `json:"dockerCompose,omitempty" yaml:"docker-compose,omitempty"`
	RancherCompose string `json:"rancherCompose,omitempty" yaml:"rancher-compose,omitempty"`
	Environment    string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

type DeployStep struct {
	DeployName        string `json:"deployName,omitempty" yaml:"deploy-name,omitempty"`
	DeployEnvironment string `json:"deployEnvironment,omitempty" yaml:"deploy-environment,omitempty"`
	Count             int    `json:"count,omitempty" yaml:"count,omitempty"`
}

type BuildStep struct {
	Repository string `json:"-" yaml:"-"`
	Branch     string `json:"-" yaml:"-"`
}

type PipelineProvider interface {
	Init(*Pipeline) error
	RunBuild(*Stage) error
	RunStage(*Stage) error
}

func (p *Pipeline) RunPipeline(provider PipelineProvider) {
	provider.Init(p)
	if len(p.Stages) > 0 {
		logrus.Info("building")
		provider.RunBuild(p.Stages[0])
	}
	logrus.Info("running other test")
	for i := 1; i < len(p.Stages); i++ {
		println(p.Stages[i].Name)
		provider.RunStage(p.Stages[i])
	}
}
