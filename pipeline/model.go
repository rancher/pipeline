package pipeline

import (
	"time"

	"github.com/rancher/go-rancher/client"
)

const StepTypeTask = "task"
const StepTypeCatalog = "catalog"
const StepTypeDeploy = "deploy"
const StepTypeSCM = "scm"
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
	RunCount        int      `json:"runCount,omitempty" yaml:"runCount,omitempty"`
	LastRunId       string   `json:"lastRunId,omitempty" yaml:"lastRunId,omitempty"`
	LastRunStatus   string   `json:"lastRunStatus,omitempty" yaml:"lastRunStatus,omitempty"`
	CommitInfo      string   `json:"commitInfo,omitempty" yaml:"commitInfo,omitempty"`
	Repository      string   `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch          string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	TargetImage     string   `json:"targetImage,omitempty" yaml:"target-image,omitempty"`
	File            string   `json:"file"`
	Stages          []*Stage `json:"stages,omitempty" yaml:"stages,omitempty"`
}

type Stage struct {
	Name        string  `json:"name,omitempty" yaml:"name,omitempty"`
	Ordinal     int     `json:"ordinal,omitempty" yaml:"ordinal,omitempty"`
	NeedApprove bool    `json:"needApprove,omitempty" yaml:"need-approve,omitempty"`
	Steps       []*Step `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Step struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string `json:"type,omitempty" yaml:"type,omitempty"`
	Ordinal int    `json:"ordinal,omitempty" yaml:"ordinal,omitempty"`
	//---SCM step
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string `json:"branch,omitempty" yaml:"branch,omitempty"`
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
	RunPipeline(*Pipeline) (*Activity, error)
	RunBuild(*Stage, string) error
	RunStage(*Activity, int) error
	SyncActivity(*Activity) (bool, error)
	GetStepLog(*Activity, int, int) (string, error)
}

type Activity struct {
	client.Resource
	Id              string          `json:"id,omitempty"`
	Pipeline        Pipeline        `json:"pipeline,omitempty"`
	PipelineName    string          `json:"pipelineName,omitempty"`
	PipelineVersion string          `json:"pipelineVersion,omitempty"`
	RunSequence     int             `json:"runSequence,omitempty"`
	CommitInfo      string          `json:"commitInfo,omitempty"`
	Status          string          `json:"status,omitempty"`
	StartTS         int64           `json:"start_ts,omitempty"`
	StopTS          int64           `json:"stop_ts,omitempty"`
	ActivityStages  []ActivityStage `json:"activity_stages,omitempty"`
}

type ActivityStage struct {
	ActivityId    string         `json:"activity_id,omitempty"`
	Name          string         `json:"name,omitempty"`
	NeedApproval  bool           `json:"need_approval,omitempty"`
	ActivitySteps []ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64          `json:"start_ts,omitempty"`
	Duration      int64          `json:"duration,omitempty"`
	Status        string         `json:"status,omitempty"`
	RawOutput     string         `json:"rawOutput,omitempty"`
}

type ActivityStep struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
	StartTS int64  `json:"start_ts,omitempty"`
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
