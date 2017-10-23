package pipeline

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/pipeline/util"
	"github.com/robfig/cron"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	Latest           = "latest"
	PipelineFileName = "pipeline.yml"
)

var ErrTemplatePathNotVaild = errors.New("TemplateBasePath is not a vaild directory path")
var ErrPipelineNotFound = errors.New("Pipeline Not found")

type PipelineContext struct {
	Provider PipelineProvider
}

func (p *PipelineContext) GetPipelineById(id string) *Pipeline {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	if len(goCollection.Data) == 0 {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil
	}
	data := goCollection.Data[0]
	ppl := Pipeline{}
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), &ppl)
	logrus.Debugf("get pipeline:%v", ppl)
	return &ppl
}

func BuildPipelineContext(provider PipelineProvider) *PipelineContext {
	r := PipelineContext{
		Provider: provider,
	}

	return &r
}

func (p *PipelineContext) CreatePipeline(pipeline *Pipeline) error {
	b, err := json.Marshal(*pipeline)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	_, err = apiClient.GenericObject.Create(&client.GenericObject{
		Name:         pipeline.Name,
		Key:          pipeline.Id,
		ResourceData: resourceData,
		Kind:         "pipeline",
	})
	logrus.Debugf("created pipeline:%v", pipeline)

	return err
}
func (p *PipelineContext) UpdatePipeline(pipeline *Pipeline) error {
	b, err := json.Marshal(*pipeline)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	filters := make(map[string]interface{})
	filters["key"] = pipeline.Id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return err
	}
	if len(goCollection.Data) == 0 {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return err
	}
	existing := goCollection.Data[0]
	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Name:         pipeline.Name,
		Key:          pipeline.Id,
		ResourceData: resourceData,
		Kind:         "pipeline",
	})
	if err != nil {
		return err
	}
	logrus.Debugf("updated pipeline")
	return nil
}

func (p *PipelineContext) DeletePipeline(id string) (*Pipeline, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return &Pipeline{}, err
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return &Pipeline{}, err
	}
	if len(goCollection.Data) == 0 {
		return &Pipeline{}, errors.New("cannot find pipeline to delete")
	}
	existing := goCollection.Data[0]
	ppl := Pipeline{}
	err = json.Unmarshal([]byte(existing.ResourceData["data"].(string)), &ppl)
	if err != nil {
		return &ppl, err
	}
	err = apiClient.GenericObject.Delete(&existing)
	if err != nil {
		return &Pipeline{}, err
	}

	return &ppl, nil
}

func toPipeline(pipelineBasePath, version string) *Pipeline {
	targetPath := path.Join(pipelineBasePath, version, PipelineFileName)
	if _, err := os.Stat(targetPath); err == os.ErrNotExist {
		return nil
	}
	data, err := ioutil.ReadFile(targetPath)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	r := Pipeline{}
	err = yaml.Unmarshal(data, &r)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	r.File = string(data)
	r.VersionSequence = version
	return &r
}

//get all pipelines from GenericObject
func (p *PipelineContext) ListPipelines() []*Pipeline {
	geObjList, err := PaginateGenericObjects("pipeline")
	if err != nil {
		logrus.Errorf("fail to list pipeline,err:%v", err)
		return nil
	}
	var pipelines []*Pipeline
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &Pipeline{}
		json.Unmarshal(b, a)
		pipelines = append(pipelines, a)
	}
	return pipelines
}

func getLatestVersionPipelineFile(pipelinePath string) *Pipeline {
	if _, er := os.Stat(pipelinePath); er == os.ErrNotExist {
		logrus.Error(errors.Wrapf(er, "pipeline <%s> not found", pipelinePath))
		return nil
	}
	fi, _ := os.OpenFile(pipelinePath, os.O_RDONLY, 0755)
	defer fi.Close()
	versions, _ := fi.Readdir(0)
	max := 0
	for _, vfi := range versions {
		if versionNum, err := strconv.Atoi(vfi.Name()); err == nil {
			if max <= versionNum {
				max = versionNum
			}
		}
	}
	return toPipeline(pipelinePath, strconv.Itoa(max))
}

func (p *PipelineContext) RunPipeline(id string, triggerType string) (*Activity, error) {
	pp := p.GetPipelineById(id)
	if pp == nil {
		return &Activity{}, ErrPipelineNotFound
	}

	activity, err := p.Provider.RunPipeline(pp, triggerType)
	if err != nil {
		return &Activity{}, err
	}

	pp.RunCount = activity.RunSequence
	pp.LastRunId = activity.Id
	pp.LastRunStatus = activity.Status
	pp.LastRunTime = activity.StartTS
	pp.NextRunTime = GetNextRunTime(pp)
	p.UpdatePipeline(pp)
	return activity, nil
}

func (p *PipelineContext) RerunActivity(activity *Activity) error {
	err := p.Provider.RerunActivity(activity)
	if err != nil {
		return err
	}
	//TODO whether update last run of pipeline or not
	return nil
}

//ResetActivity delete previous build info and reset activity status
func (p *PipelineContext) ResetActivity(activity *Activity) error {
	err := p.Provider.DeleteFormerBuild(activity)
	if err != nil {
		return err
	}
	resetActivityStatus(activity)
	return nil

}

//resetActivityStatus reset status and timestamp
func resetActivityStatus(activity *Activity) {
	activity.Status = ActivityWaiting
	activity.PendingStage = 0
	activity.StartTS = 0
	activity.StopTS = 0
	for _, stage := range activity.ActivityStages {
		stage.Duration = 0
		stage.StartTS = 0
		stage.Status = ActivityStageWaiting
		for _, step := range stage.ActivitySteps {
			step.Duration = 0
			step.StartTS = 0
			step.Status = ActivityStepWaiting
		}
	}
}

func (p *PipelineContext) ApproveActivity(activity *Activity) error {
	if activity == nil {
		return errors.New("nil activity!")
	}
	if activity.Status != ActivityPending {
		return errors.New("activity not pending for approval!")
	}
	err := p.Provider.RunStage(activity, activity.PendingStage)
	return err
}

func (p *PipelineContext) DenyActivity(activity *Activity) error {
	if activity == nil {
		return errors.New("nil activity!")
	}
	if activity.Status != ActivityPending {
		return errors.New("activity not pending for deny!")
	}
	if activity.PendingStage < len(activity.ActivityStages) {
		activity.ActivityStages[activity.PendingStage].Status = ActivityStageDenied
		activity.Status = ActivityDenied
	}
	return nil

}
func GetNextRunTime(pipeline *Pipeline) int64 {
	nextRunTime := int64(0)
	if !pipeline.IsActivate {
		return nextRunTime
	}
	trigger := pipeline.CronTrigger
	if trigger == nil {
		return nextRunTime
	}
	spec := trigger.Spec
	if spec == "" {
		return nextRunTime
	}
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		logrus.Errorf("error parse cron exp,%v,%v", spec, err)
		return nextRunTime
	}
	nextRunTime = schedule.Next(time.Now()).UnixNano() / int64(time.Millisecond)

	return nextRunTime
}

//get updated activity from provider
func (p *PipelineContext) SyncActivity(activity *Activity) error {
	//its done, no need to sync
	//return nil

	if activity.Status == ActivityFail || activity.Status == ActivitySuccess {
		return nil
	}
	return p.Provider.SyncActivity(activity)
}

//GetServices gets run services before the step
func GetServices(activity *Activity, stageOrdinal int, stepOrdinal int) []*CIService {
	services := []*CIService{}
	for i := 0; i <= stageOrdinal; i++ {
		for j := 0; j < len(activity.Pipeline.Stages[i].Steps); j++ {
			if i == stageOrdinal && j >= stepOrdinal {
				break
			}
			step := activity.Pipeline.Stages[i].Steps[j]
			if step.IsService && step.Type == StepTypeTask {
				service := &CIService{
					ContainerName: activity.Id + step.Alias,
					Name:          step.Alias,
					Image:         step.Image,
				}
				services = append(services, service)
			}
		}
	}
	return services
}

//GetAllServices gets all run services of the activity
func GetAllServices(activity *Activity) []*CIService {
	lastStageOrdinal := len(activity.ActivityStages) - 1
	if lastStageOrdinal < 0 {
		lastStageOrdinal = 0
	}
	lastStepOrdinal := len(activity.ActivityStages[lastStageOrdinal].ActivitySteps) - 1
	if lastStepOrdinal < 0 {
		lastStepOrdinal = 0
	}
	return GetServices(activity, lastStageOrdinal, lastStepOrdinal)
}

func HasStepCondition(s *Step) bool {
	//return s.Condition != ""
	return s.Conditions != nil && (len(s.Conditions.All) > 0 || len(s.Conditions.Any) > 0)
}

func HasStageCondition(s *Stage) bool {
	//return s.Condition != ""
	return s.Conditions != nil && (len(s.Conditions.All) > 0 || len(s.Conditions.Any) > 0)
}

func PaginateGenericObjects(kind string) ([]client.GenericObject, error) {
	result := []client.GenericObject{}
	limit := "1000"
	marker := ""
	var pageData []client.GenericObject
	var err error
	for {
		logrus.Debugf("paging got:%v,%v,%v", kind, limit, marker)
		pageData, marker, err = getGenericObjects(kind, limit, marker)
		if err != nil {
			logrus.Debugf("get genericobject err:%v", err)
			return nil, err
		}
		result = append(result, pageData...)
		if marker == "" {
			break
		}
	}
	return result, nil
}

func getGenericObjects(kind string, limit string, marker string) ([]client.GenericObject, string, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		logrus.Errorf("fail to get client:%v", err)
		return nil, "", err
	}
	filters := make(map[string]interface{})
	filters["kind"] = kind
	filters["limit"] = limit
	filters["marker"] = marker
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("fail querying generic objects, error:%v", err)
		return nil, "", err
	}
	//get next marker
	nextMarker := ""
	if goCollection.Pagination != nil && goCollection.Pagination.Next != "" {
		r, err := url.Parse(goCollection.Pagination.Next)
		if err != nil {
			logrus.Errorf("fail parsing next url, error:%v", err)
			return nil, "", err
		}
		nextMarker = r.Query().Get("marker")
	}
	return goCollection.Data, nextMarker, err

}
