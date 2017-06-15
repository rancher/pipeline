package restfulserver

import (
	"net/http"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/pipeline"
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

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}
	schemas.AddType("error", Error{})
	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	pipelineSchema(schemas.AddType("pipeline", Pipeline{}))
	acitvitySchema(schemas.AddType("activity", Activity{}))
	return schemas
}

type Pipeline struct {
	client.Resource
	pipeline.Pipeline
	Status     []string   `json:"status,omitempty"`
	Activities []Activity `json:"activities,omitempty"`
}

type Empty struct {
	client.Resource
}

type Error struct {
	client.Resource
	Status   int    `json:"status"`
	Code     string `json:"code"`
	Msg      string `json:"message"`
	Detail   string `json:"detail"`
	BaseType string `json:"baseType"`
}

type Activity struct {
	client.Resource
	Id             string             `json:"id,omitempty"`
	FromPipeline   *pipeline.Pipeline `json:"from_pipeline,omitempty"`
	Status         string             `json:"status,omitempty"`
	Result         string             `json:"result,omitempty"`
	StartTS        int64              `json:"start_ts,omitempty"`
	StopTS         int64              `json:"stop_ts,omitempty"`
	ActivityStages []ActivityStage    `json:"activity_stages,omitempty"`
}

type ActivityStage struct {
	Name          string         `json:"name,omitstage"`
	NeedApproval  bool           `json:"need_approval,omitempty"`
	AcitvitySteps []ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64          `json:"start_ts,omitempty"`
	Status        string         `json:"status,omitempty"`
}

type ActivityStep struct {
	Name    string `json:"name,omitempty"`
	Image   string `json:"image,omitempty"`
	Command string `json:"command,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
	StartTS int64  `json:"start_ts,omitempty"`
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
	pipeline.ResourceActions = map[string]client.Action{
		"run": client.Action{
			Output: "activity",
		},
	}

	pipeline.CollectionMethods = []string{http.MethodGet, http.MethodPost}
}

func acitvitySchema(activity *client.Schema) {
	activity.ResourceFields["from_pipeline"] = client.Field{
		Type:     "struct",
		Nullable: true,
	}
}

func toPipelineCollections(apiContext *api.ApiContext, pipelines []*pipeline.Pipeline) []interface{} {
	var r []interface{}
	for _, p := range pipelines {
		r = append(r, toPipelineResourceWithoutActivities(apiContext, p))
	}
	return r
}

func toPipelineResourceWithoutActivities(apiContext *api.ApiContext, pipeline *pipeline.Pipeline) *Pipeline {
	r := Pipeline{
		Resource: client.Resource{
			Id:      pipeline.Name,
			Type:    "pipeline",
			Actions: map[string]string{},
			Links:   map[string]string{
			//"activities": apiContext.UrlBuilder.ReferenceLink(nil),
			},
		},
		Pipeline: *pipeline,
		//Activities: []Activity{*toActivityResource(apiContext)},
	}
	r.Actions["run"] = apiContext.UrlBuilder.ReferenceLink(r.Resource) + "?action=run"
	return &r
}
