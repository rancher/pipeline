package restfulserver

import (
	"net/http"

	"github.com/rancher/pipeline/scm"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/pipeline"
)

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}
	schemas.AddType("error", Error{})
	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	pipelineSchema(schemas.AddType("pipeline", pipeline.Pipeline{}))
	acitvitySchema(schemas.AddType("activity", pipeline.Activity{}))
	pipelineSettingSchema(schemas.AddType("setting", pipeline.PipelineSetting{}))
	accountSchema(schemas.AddType("gitaccount", scm.Account{}))
	return schemas
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
		"update": client.Action{
			Output: "pipeline",
		},
		"activate": client.Action{
			Output: "pipeline",
		},
		"deactivate": client.Action{
			Output: "pipeline",
		},
		"remove": client.Action{
			Output: "pipeline",
		},
		"export": client.Action{
			Output: "pipeline",
		},
	}

	pipeline.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	pipeline.IncludeableLinks = []string{"activities"}
}

func acitvitySchema(activity *client.Schema) {
	activity.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	activity.PluralName = "activities"
}

func pipelineSettingSchema(setting *client.Schema) {
	setting.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	setting.ResourceActions = map[string]client.Action{
		"update": client.Action{
			Output: "setting",
		},
	}
}

func accountSchema(account *client.Schema) {
	account.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	account.ResourceActions = map[string]client.Action{
		"update": client.Action{
			Output: "gitaccount",
		},
	}
}

func toPipelineCollections(apiContext *api.ApiContext, pipelines []*pipeline.Pipeline) []interface{} {
	var r []interface{}
	for _, p := range pipelines {
		r = append(r, toPipelineResource(apiContext, p))
	}
	return r
}

func toPipelineResource(apiContext *api.ApiContext, pipeline *pipeline.Pipeline) *pipeline.Pipeline {
	pipeline.Resource = client.Resource{
		Id:      pipeline.Id,
		Type:    "pipeline",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	pipeline.Actions["run"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=run"
	pipeline.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=update"
	pipeline.Actions["remove"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=remove"
	pipeline.Actions["activate"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=activate"
	pipeline.Actions["deactivate"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=deactivate"
	pipeline.Actions["export"] = apiContext.UrlBuilder.ReferenceLink(pipeline.Resource) + "?action=export"

	pipeline.Links["activities"] = apiContext.UrlBuilder.Link(pipeline.Resource, "activities")
	pipeline.Links["exportConfig"] = apiContext.UrlBuilder.Link(pipeline.Resource, "exportConfig")
	return pipeline
}

func toActivityResource(apiContext *api.ApiContext, a *pipeline.Activity) *pipeline.Activity {
	a.Resource = client.Resource{
		Id:      a.Id,
		Type:    "activity",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	a.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=update"
	a.Actions["remove"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=remove"
	//TODO if a.Iscomplete()
	if a.Status != pipeline.ActivityWaiting &&
		a.Status != pipeline.ActivityBuilding &&
		a.Status != pipeline.ActivityPending {
		a.Actions["rerun"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=rerun"
	} else {
		a.Actions["stop"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=stop"
	}

	//remove pipeline reference
	a.Pipeline.Type = ""
	return a
}

func toAccountResource(apiContext *api.ApiContext, account *scm.Account) *scm.Account {
	account.Resource = client.Resource{
		Id:      account.Id,
		Type:    "gitaccount",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	if account.Private {
		account.Actions["share"] = apiContext.UrlBuilder.ReferenceLink(account.Resource) + "?action=share"
	} else {
		account.Actions["unshare"] = apiContext.UrlBuilder.ReferenceLink(account.Resource) + "?action=unshare"
	}
	account.Actions["refreshrepos"] = apiContext.UrlBuilder.ReferenceLink(account.Resource) + "?action=refreshrepos"
	account.Actions["remove"] = apiContext.UrlBuilder.ReferenceLink(account.Resource) + "?action=remove"
	account.Links["repos"] = apiContext.UrlBuilder.ReferenceLink(account.Resource) + "/repos"
	return account
}

func toPipelineSettingResource(apiContext *api.ApiContext, setting *pipeline.PipelineSetting) *pipeline.PipelineSetting {
	setting.Resource = client.Resource{
		Type:    "setting",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	setting.Actions["update"] = apiContext.UrlBuilder.Current() + "?action=update" //apiContext.UrlBuilder.ReferenceLink(setting.Resource) + "?action=update"
	setting.Actions["githuboauth"] = apiContext.UrlBuilder.Current() + "?action=githuboauth"
	setting.Actions["getrepos"] = apiContext.UrlBuilder.Current() + "?action=getrepos"

	return setting
}

func initActivityResource(a *pipeline.Activity) {
	a.Resource = client.Resource{
		Id:      a.Id,
		Type:    "activity",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
}
