package restfulserver

import "github.com/rancher/go-rancher/client"

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}
	schemas.AddType("error", client.ApiError{})
	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("pipeline", Pipeline{})
	schemas.AddType("schema", client.Schema{})
	return schemas
}

type Pipeline struct {
	client.Resource
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func pipelineSchema(pipeline *client.Schema) {
	pipeline.CollectionMethods = []string{"GET"}
	pipelineName := pipeline.ResourceFields["name"]
	pipelineName.Create = true
	pipelineName.Required = true
	pipelineName.Unique = true
	pipeline.ResourceFields["name"] = pipelineName
}

func toPipelineResource() *Pipeline {
	r := Pipeline{
		Resource: client.Resource{
			Id:      "hello",
			Type:    "pipeline",
			Actions: map[string]string{},
			Links:   map[string]string{},
		},
		Name:   "hello",
		Status: "active",
	}
	return &r
}
