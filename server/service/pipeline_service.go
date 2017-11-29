package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/util"
	"github.com/robfig/cron"
)

func GetPipelineById(id string) (*model.Pipeline, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil, err
	}
	if len(goCollection.Data) == 0 {
		return nil, fmt.Errorf("pipeline '%s' is not found", id)
	}
	data := goCollection.Data[0]
	ppl := &model.Pipeline{}
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), ppl)
	return ppl, nil
}

func CreatePipeline(pipeline *model.Pipeline) error {
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

func UpdatePipeline(pipeline *model.Pipeline) error {
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

func DeletePipeline(id string) (*model.Pipeline, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "pipeline"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		logrus.Errorf("Error %v filtering genericObjects by key", err)
		return nil, err
	}
	if len(goCollection.Data) == 0 {
		return nil, errors.New("cannot find pipeline to delete")
	}
	existing := goCollection.Data[0]
	ppl := &model.Pipeline{}
	if err = json.Unmarshal([]byte(existing.ResourceData["data"].(string)), ppl); err != nil {
		return ppl, err
	}
	if err = apiClient.GenericObject.Delete(&existing); err != nil {
		return nil, err
	}

	return ppl, nil
}

//get all pipelines from GenericObject
func ListPipelines() []*model.Pipeline {
	geObjList, err := PaginateGenericObjects("pipeline")
	if err != nil {
		logrus.Errorf("fail to list pipeline,err:%v", err)
		return nil
	}
	var pipelines []*model.Pipeline
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Pipeline{}
		json.Unmarshal(b, a)
		pipelines = append(pipelines, a)
	}
	return pipelines
}

func RunPipeline(provider model.PipelineProvider, id string, triggerType string) (*model.Activity, error) {
	pp, err := GetPipelineById(id)
	if err != nil {
		return nil, fmt.Errorf("fail to get pipeline: %v", err)
	}

	activity, err := provider.RunPipeline(pp, triggerType)
	if err != nil {
		return nil, err
	}

	pp.RunCount = activity.RunSequence
	pp.LastRunId = activity.Id
	pp.LastRunStatus = activity.Status
	pp.LastRunTime = activity.StartTS
	pp.NextRunTime = GetNextRunTime(pp)
	UpdatePipeline(pp)
	return activity, nil
}

func HasStepCondition(s *model.Step) bool {
	return s.Conditions != nil && (len(s.Conditions.All) > 0 || len(s.Conditions.Any) > 0)
}

func HasStageCondition(s *model.Stage) bool {
	return s.Conditions != nil && (len(s.Conditions.All) > 0 || len(s.Conditions.Any) > 0)
}

func GetNextRunTime(pipeline *model.Pipeline) int64 {
	nextRunTime := int64(0)
	if !pipeline.IsActivate {
		return nextRunTime
	}
	trigger := pipeline.CronTrigger
	spec := trigger.Spec
	timezone := trigger.Timezone
	if spec == "" {
		return nextRunTime
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logrus.Errorf("fail get timezone '%s',err:%v", timezone, err)
		return nextRunTime
	}
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		logrus.Errorf("error parse cron exp,%v,%v", spec, err)
		return nextRunTime
	}
	nextRunTime = schedule.Next(time.Now().In(loc)).UnixNano() / int64(time.Millisecond)

	return nextRunTime
}

func PaginateGenericObjects(kind string) ([]client.GenericObject, error) {
	result := []client.GenericObject{}
	limit := "1000"
	marker := ""
	var pageData []client.GenericObject
	var err error
	for {
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
