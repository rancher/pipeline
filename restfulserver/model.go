package restfulserver

import (
	"time"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

const pipelineFileExample = `---
stage_zero:
    name: stage zero
	need_approve: false
	steps:
	  - name: build step
	    image: test/build:v0.1
		command: echo 'i am turkey'
`
const ActivityStepWaitting = "waitting"
const ActivityStepRunning = "running"
const ActivityStepFinished = "finished"
const ActivityStepError = "error"

const ActivityStageRunning = "running"
const ActivityStageWaitting = "waitting"
const ActivityStageError = "error"
const ActivityStageFinished = "finished"

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}
	schemas.AddType("error", client.ApiError{})
	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	pipelineSchema(schemas.AddType("pipeline", Pipeline{}))
	acitvitySchema(schemas.AddType("activity", Activity{}))
	return schemas
}

type Pipeline struct {
	client.Resource
	Name            string     `json:"name,omitempty"`
	Repository      string     `json:"repository,omitempty"`
	Branch          string     `json:"branch,omitempty"`
	Version         string     `json:"version,omitempty"`
	Status          string     `json:"status,omitempty"`
	PipelineContent string     `json:"pipeline_content,omitempty"`
	Activities      []Activity `json:"activities,omitempty"`
}

type Activity struct {
	client.Resource
	Id             string          `json:"id,omitempty"`
	FromPipeline   *Pipeline       `json:"from_pipeline,omitempty"`
	Status         string          `json:"status,omitempty"`
	Result         string          `json:"result,omitempty"`
	StartTS        int64           `json:"start_ts,omitempty"`
	StopTS         int64           `json:"stop_ts,omitempty"`
	ActivityStages []ActivityStage `json:"activity_stages,omitempty"`
}

type ActivityStage struct {
	Name          string         `json:"name,omitstage"`
	NeedApproval  bool           `json:"need_approval,omitempty"`
	AcitvitySteps []ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64          `json:"start_ts,omitempty"`
	Status        string         `json:"status,omitempty"`
}

type ActivityStep struct {
	Name    string `json:"name,omitstage"`
	Image   string `json:"image"`
	Command string `json:"command"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

func pipelineSchema(pipeline *client.Schema) {
	pipeline.CollectionMethods = []string{"GET"}
	pipelineName := pipeline.ResourceFields["name"]
	pipelineName.Create = true
	pipelineName.Required = true
	pipelineName.Unique = true
	pipeline.ResourceFields["name"] = pipelineName

	pipelineRepository := pipeline.ResourceFields["repository"]
	pipelineRepository.Create = true
	pipelineRepository.Required = true
	pipeline.ResourceFields["repository"] = pipelineRepository

	pipelineBranch := pipeline.ResourceFields["branch"]
	pipelineBranch.Create = true
	pipelineBranch.Required = true
	pipeline.ResourceFields["branch"] = pipelineBranch

	//todo others

}

func acitvitySchema(activity *client.Schema) {
	activity.ResourceFields["from_pipeline"] = client.Field{
		Type:     "struct",
		Nullable: true,
	}
}

func toPipelineResource(apiContext *api.ApiContext) *Pipeline {
	r := Pipeline{
		Resource: client.Resource{
			Id:      "example",
			Type:    "pipeline",
			Actions: map[string]string{},
			Links:   map[string]string{
			//"activities": apiContext.UrlBuilder.ReferenceLink(nil),
			},
		},
		Name:            "example",
		Status:          "active",
		Repository:      "github.com/orangedeng/ui",
		Branch:          "master",
		PipelineContent: pipelineFileExample,
		Activities:      []Activity{*toActivityResource(apiContext)},
	}
	return &r
}

func toActivityResource(apiContext *api.ApiContext) *Activity {
	pipeline := toPipelineResourceWithoutActivities(apiContext)
	r := Activity{
		Resource: client.Resource{
			Id:      "example#1",
			Type:    "activity",
			Actions: map[string]string{},
			Links: map[string]string{
				"pipeline": apiContext.UrlBuilder.ReferenceLink(pipeline.Resource),
			},
		},
		Id:     "example#1",
		Status: "Finished",
		ActivityStages: []ActivityStage{
			ActivityStage{
				Name:         "stage zeor",
				NeedApproval: false,
				AcitvitySteps: []ActivityStep{
					ActivityStep{
						Name:    "build step",
						Image:   "test/build:v0.1",
						Command: "echo 'i am turkey'",
						Message: "build success",
						Status:  ActivityStepFinished,
					},
				},
				Status:  ActivityStageFinished,
				StartTS: time.Now().Unix() - 30,
			},
		},
		FromPipeline: pipeline,
	}
	return &r
}

func toPipelineResourceWithoutActivities(apiContext *api.ApiContext) *Pipeline {
	r := Pipeline{
		Resource: client.Resource{
			Id:      "example",
			Type:    "pipeline",
			Actions: map[string]string{},
			Links:   map[string]string{},
		},
		Name:            "example",
		Status:          "active",
		Repository:      "github.com/orangedeng.ui",
		Branch:          "master",
		PipelineContent: pipelineFileExample,
		//Activities:      []Activity{*toActivityResource(apiContext)},
	}
	return &r
}
