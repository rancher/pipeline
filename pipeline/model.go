package pipeline

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/client"
)

const StepTypeTask = "task"
const StepTypeCatalog = "catalog"
const StepTypeDeploy = "deploy"
const (
	ActivityStepWaitting = "Waitting"
	ActivityStepBuilding = "Building"
	ActivityStepSuccess  = "Success"
	ActivityStepFail     = "Fail"

	ActivityStageWaitting = "Waitting"
	ActivityStageBuilding = "Building"
	ActivityStageSuccess  = "Success"
	ActivityStageFail     = "Fail"

	ActivityWaitting = "Waitting"
	ActivityBuilding = "Building"
	ActivitySuccess  = "Success"
	ActivityFail     = "Fail"
)

type Pipeline struct {
	client.Resource
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
	//---task step
	Command    string   `json:"command,omitempty" yaml:"command,omitempty"`
	Image      string   `json:"image,omitempty" yaml:"image,omitempty"`
	Parameters []string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	//---catalog step
	DockerCompose  string `json:"dockerCompose,omitempty" yaml:"docker-compose,omitempty"`
	RancherCompose string `json:"rancherCompose,omitempty" yaml:"rancher-compose,omitempty"`
	Environment    string `json:"environment,omitempty" yaml:"environment,omitempty"`
	//---deploy step
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

type Activity struct {
	client.Resource
	Id              string          `json:"id,omitempty"`
	PipelineName    string          `json:"pipelineName,omitempty"`
	PipelineVersion string          `json:"pipelineVersion,omitempty"`
	Status          string          `json:"status,omitempty"`
	StartTS         int64           `json:"start_ts,omitempty"`
	StopTS          int64           `json:"stop_ts,omitempty"`
	ActivityStages  []ActivityStage `json:"activity_stages,omitempty"`
}

type ActivityStage struct {
	Name          string         `json:"name,omitempty"`
	NeedApproval  bool           `json:"need_approval,omitempty"`
	ActivitySteps []ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64          `json:"start_ts,omitempty"`
	Status        string         `json:"status,omitempty"`
	RawOutput     string         `json:"rawOutput,omitempty"`
}

type ActivityStep struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
	StartTS int64  `json:"start_ts,omitempty"`
}

func (p *Pipeline) RunPipeline(provider PipelineProvider) {
	provider.Init(p)
	if len(p.Stages) > 0 {
		logrus.Info("building")
		if err := provider.RunBuild(p.Stages[0]); err != nil {
			logrus.Error(errors.Wrap(err, "build stage fail"))
			return
		}
	}
	logrus.Info("running other test")
	for i := 1; i < len(p.Stages); i++ {
		if err := provider.RunStage(p.Stages[i]); err != nil {
			logrus.Error(errors.Wrapf(err, "stage <%s> fail", p.Stages[i].Name))
			return
		}
	}
}

func ToDemoActivity() *Activity {
	startTS := (time.Now().Unix() - 30) * 1000
	stopTS := time.Now().Unix()
	r := Activity{
		Id:              "test",
		PipelineName:    "test1",
		PipelineVersion: "0",
		Status:          ActivitySuccess,
		StartTS:         startTS,
		StopTS:          stopTS,
		ActivityStages: []ActivityStage{
			ActivityStage{
				Name:         "build",
				NeedApproval: false,
				StartTS:      startTS,
				Status:       ActivityStageSuccess,
				RawOutput:    "",
				ActivitySteps: []ActivityStep{
					ActivityStep{
						Name:    "build",
						Message: "",
						Status:  ActivityStageSuccess,
					},
				},
			},
		},
	}
	return &r
}
