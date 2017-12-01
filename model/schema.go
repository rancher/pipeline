package model

import (
	"net/http"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}
	schemas.AddType("error", Error{})
	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	pipelineSchema(schemas.AddType("pipeline", Pipeline{}))
	acitvitySchema(schemas.AddType("activity", Activity{}))
	pipelineSettingSchema(schemas.AddType("setting", PipelineSetting{}))
	scmSettingSchema(schemas.AddType("scmSetting", SCMSetting{}))
	accountSchema(schemas.AddType("gitaccount", GitAccount{}))
	repositorySchema(schemas.AddType("gitrepository", GitRepository{}))
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

	activity.ResourceActions = map[string]client.Action{
		"rerun": client.Action{
			Output: "activity",
		},
		"update": client.Action{
			Output: "activity",
		},
		"approve": client.Action{
			Output: "activity",
		},
		"deny": client.Action{
			Output: "activity",
		},
		"remove": client.Action{
			Output: "activity",
		},
		"stop": client.Action{
			Output: "activity",
		},
	}
}

func pipelineSettingSchema(setting *client.Schema) {
	setting.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	setting.ResourceActions = map[string]client.Action{
		"update": client.Action{
			Output: "setting",
		},
	}
}

func scmSettingSchema(setting *client.Schema) {
	setting.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	setting.ResourceActions = map[string]client.Action{
		"update": client.Action{
			Output: "scmSetting",
		},
		"remove": client.Action{
			Output: "scmSetting",
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

func repositorySchema(repository *client.Schema) {
	repository.CollectionMethods = []string{http.MethodGet, http.MethodPost}
	repository.PluralName = "gitrepositories"
}

func ToPipelineCollections(apiContext *api.ApiContext, pipelines []*Pipeline) []interface{} {
	var r []interface{}
	for _, p := range pipelines {
		r = append(r, ToPipelineResource(apiContext, p))
	}
	return r
}

func ToPipelineResource(apiContext *api.ApiContext, pipeline *Pipeline) *Pipeline {
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
	filterPipeline(pipeline)
	return pipeline
}

func ToActivityResource(apiContext *api.ApiContext, a *Activity) *Activity {
	a.Resource = client.Resource{
		Id:      a.Id,
		Type:    "activity",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	a.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=update"
	a.Actions["remove"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=remove"
	//TODO if a.Iscomplete()
	if a.Status != ActivityWaiting &&
		a.Status != ActivityBuilding &&
		a.Status != ActivityPending {
		a.Actions["rerun"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=rerun"
	} else {
		a.Actions["stop"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=stop"
	}

	filterActivity(a)
	return a
}

func ToAccountResource(apiContext *api.ApiContext, account *GitAccount) *GitAccount {
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
	filterAccount(account)
	return account
}

func ToRepositoryResource(apiContext *api.ApiContext, repository *GitRepository) *GitRepository {
	repository.Resource = client.Resource{
		Id:      repository.Id,
		Type:    "gitrepository",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	return repository
}

func ToPipelineSettingResource(apiContext *api.ApiContext, setting *PipelineSetting) *PipelineSetting {
	setting.Resource = client.Resource{
		Type:    "setting",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	setting.Actions["update"] = apiContext.UrlBuilder.Current() + "?action=update" //apiContext.UrlBuilder.ReferenceLink(setting.Resource) + "?action=update"
	setting.Actions["oauth"] = apiContext.UrlBuilder.Current() + "?action=oauth"
	setting.Actions["reset"] = apiContext.UrlBuilder.Current() + "?action=reset"

	setting.Links["scmsettings"] = apiContext.UrlBuilder.Current() + "/scmsettings"
	return setting
}

func ToSCMSettingResource(apiContext *api.ApiContext, setting *SCMSetting) *SCMSetting {
	setting.Resource = client.Resource{
		Id:      setting.Id,
		Type:    "scmSetting",
		Actions: map[string]string{},
		Links:   map[string]string{},
	}
	setting.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(setting.Resource) + "?action=update"
	setting.Actions["remove"] = apiContext.UrlBuilder.ReferenceLink(setting.Resource) + "?action=remove"
	filterSCMSetting(setting)
	return setting
}

func filterPipeline(pipeline *Pipeline) {
	pipeline.WebHookToken = ""
}

func filterActivity(activity *Activity) {
	//remove pipeline reference
	activity.Pipeline.Type = ""
}

func filterAccount(account *GitAccount) {
	account.AccessToken = ""
}

func filterSCMSetting(setting *SCMSetting) {
	setting.ClientSecret = ""
}
