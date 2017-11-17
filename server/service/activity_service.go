package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/util"
)

func ListActivities() ([]*model.Activity, error) {
	geObjList, err := PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity, err:%v", err)
		return nil, err
	}
	var activities []*model.Activity
	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Activity{}
		json.Unmarshal(b, a)
		activities = append(activities, a)
	}

	return activities, nil
}

//Get Activity From GenericObjects By Id
func GetActivity(id string) (*model.Activity, error) {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return nil, err
	}
	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "activity"
	goCollection, err := apiClient.GenericObject.List(&client.ListOpts{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("Error %v filtering genericObjects by key", err)
	}
	if len(goCollection.Data) == 0 {
		return nil, fmt.Errorf("Requested activity not found")
	}
	data := goCollection.Data[0]
	activity := &model.Activity{}
	json.Unmarshal([]byte(data.ResourceData["data"].(string)), activity)

	return activity, nil
}

func CreateActivity(activity *model.Activity) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	//activity.Id = uuid.Rand().Hex()
	b, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	resourceData := map[string]interface{}{
		"data": string(b),
	}

	if _, err := apiClient.GenericObject.Create(&client.GenericObject{
		Name:         activity.Id,
		Key:          activity.Id,
		ResourceData: resourceData,
		Kind:         "activity",
	}); err != nil {
		return fmt.Errorf("Failed to save activity: %v", err)
	}
	return nil
}

func UpdateActivity(activity *model.Activity) error {
	logrus.Debugf("updating activity %v.", activity.Id)
	logrus.Debugf("activity stages:%v", activity.ActivityStages)
	b, err := json.Marshal(activity)
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
	filters["key"] = activity.Id
	filters["kind"] = "activity"
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
	existing := goCollection.Data[0]
	_, err = apiClient.GenericObject.Update(&existing, &client.GenericObject{
		Name:         activity.Id,
		Key:          activity.Id,
		ResourceData: resourceData,
		Kind:         "activity",
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteActivity(id string) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}

	filters := make(map[string]interface{})
	filters["key"] = id
	filters["kind"] = "activity"
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
	existing := goCollection.Data[0]
	err = apiClient.GenericObject.Delete(&existing)
	if err != nil {
		return err
	}
	return nil
}

func RerunActivity(provider model.PipelineProvider, activity *model.Activity) error {

	if err := provider.RerunActivity(activity); err != nil {
		return err
	}
	//TODO whether update last run of pipeline or not
	return nil
}

//ResetActivity delete previous build info and reset activity status
func ResetActivity(provider model.PipelineProvider, activity *model.Activity) error {
	if err := provider.DeleteFormerBuild(activity); err != nil {
		return err
	}
	resetActivityStatus(activity)
	return nil

}

//resetActivityStatus reset status and timestamp
func resetActivityStatus(activity *model.Activity) {
	activity.Status = model.ActivityWaiting
	activity.PendingStage = 0
	activity.StartTS = 0
	activity.StopTS = 0
	for _, stage := range activity.ActivityStages {
		stage.Duration = 0
		stage.StartTS = 0
		stage.Status = model.ActivityStageWaiting
		for _, step := range stage.ActivitySteps {
			step.Duration = 0
			step.StartTS = 0
			step.Status = model.ActivityStepWaiting
		}
	}
}

func ApproveActivity(provider model.PipelineProvider, activity *model.Activity) error {
	if activity == nil {
		return errors.New("nil activity")
	}
	if activity.Status != model.ActivityPending {
		return errors.New("activity not pending for approval")
	}
	return provider.RunStage(activity, activity.PendingStage)
}

func DenyActivity(activity *model.Activity) error {
	if activity == nil {
		return errors.New("nil activity")
	}
	if activity.Status != model.ActivityPending {
		return errors.New("activity not pending for deny")
	}
	if activity.PendingStage < len(activity.ActivityStages) {
		activity.ActivityStages[activity.PendingStage].Status = model.ActivityStageDenied
		activity.Status = model.ActivityDenied
	}
	return nil

}

func StopActivity(provider model.PipelineProvider, activity *model.Activity) error {
	if activity == nil {
		return errors.New("nil activity")
	}
	if activity.Status != model.ActivityBuilding && activity.Status != model.ActivityWaiting {
		return errors.New("Not a running activity for stop")
	}

	return provider.StopActivity(activity)

}

//get updated activity from provider
func SyncActivity(provider model.PipelineProvider, activity *model.Activity) error {
	//its done, no need to sync
	if activity.Status == model.ActivityFail || activity.Status == model.ActivitySuccess ||
		activity.Status == model.ActivityDenied || activity.Status == model.ActivityAbort {
		return nil
	}
	return provider.SyncActivity(activity)
}

//GetServices gets run services before the step
func GetServices(activity *model.Activity, stageOrdinal int, stepOrdinal int) []*model.CIService {
	services := []*model.CIService{}
	for i := 0; i <= stageOrdinal; i++ {
		for j := 0; j < len(activity.Pipeline.Stages[i].Steps); j++ {
			if i == stageOrdinal && j >= stepOrdinal {
				break
			}
			step := activity.Pipeline.Stages[i].Steps[j]
			if step.IsService && step.Type == model.StepTypeTask {
				service := &model.CIService{
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
func GetAllServices(activity *model.Activity) []*model.CIService {
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

func IsStageSuccess(stage *model.ActivityStage) bool {
	if stage == nil {
		return false
	}

	if stage.Status == model.ActivityStageFail || stage.Status == model.ActivityStageDenied {
		return false
	}
	successSteps := 0
	for _, step := range stage.ActivitySteps {
		if step.Status == model.ActivityStepSuccess || step.Status == model.ActivityStepSkip {
			successSteps++
		}
	}
	return successSteps == len(stage.ActivitySteps)
}

func StartStep(activity *model.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.StartTS = curTime
	step.Status = model.ActivityStepBuilding
	stage.Status = model.ActivityStageBuilding
	activity.Status = model.ActivityBuilding
	if stepOrdinal == 0 {
		stage.StartTS = curTime
	}
}

func FailStep(activity *model.Activity, stageOrdinal int, stepOrdinal int) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.Status = model.ActivityStepFail
	step.Duration = now - step.StartTS
	stage.Status = model.ActivityStageFail
	stage.Duration = now - stage.StartTS
	activity.Status = model.ActivityFail
	activity.StopTS = now
	activity.FailMessage = fmt.Sprintf("Execution fail in '%v' stage, step %v", stage.Name, stepOrdinal+1)
}

func SuccessStep(activity *model.Activity, stageOrdinal int, stepOrdinal int) {
	curTime := time.Now().UnixNano() / int64(time.Millisecond)
	stage := activity.ActivityStages[stageOrdinal]
	step := stage.ActivitySteps[stepOrdinal]
	step.Status = model.ActivityStepSuccess
	step.Duration = curTime - step.StartTS
	if stage.Status == model.ActivityStageFail {
		return
	}

	if IsStageSuccess(stage) {
		stage.Status = model.ActivityStageSuccess
		stage.Duration = curTime - stage.StartTS
		if stageOrdinal == len(activity.ActivityStages)-1 {
			activity.Status = model.ActivitySuccess
			activity.StopTS = curTime
		} else {
			nextStage := activity.ActivityStages[stageOrdinal+1]
			if nextStage.NeedApproval {
				nextStage.Status = model.ActivityStagePending
				activity.Status = model.ActivityPending
				activity.PendingStage = stageOrdinal + 1
			}
		}
	}

}

func Triggernext(activity *model.Activity, stageOrdinal int, stepOrdinal int, provider model.PipelineProvider) {
	logrus.Debugf("triggering next:%d,%d", stageOrdinal, stepOrdinal)
	if activity.Status == model.ActivitySuccess ||
		activity.Status == model.ActivityFail ||
		activity.Status == model.ActivityPending ||
		activity.Status == model.ActivityDenied ||
		activity.Status == model.ActivityAbort {
		return
	}
	stage := activity.ActivityStages[stageOrdinal]
	if IsStageSuccess(stage) && stageOrdinal+1 < len(activity.ActivityStages) {
		nextStage := activity.ActivityStages[stageOrdinal+1]
		if err := provider.RunStage(activity, stageOrdinal+1); err != nil {
			logrus.Errorf("trigger next stage '%s' got error:%v", nextStage.Name, err)
			//activity.Status = Error
			activity.FailMessage = fmt.Sprintf("trigger next stage '%s' got error:%v", nextStage.Name, err)
		}
		return
	}

	if !activity.Pipeline.Stages[stageOrdinal].Parallel {
		if err := provider.RunStep(activity, stageOrdinal, stepOrdinal+1); err != nil {
			logrus.Errorf("trigger step #%d of '%s' got error:%v", stepOrdinal+2, stage.Name, err)
			//activity.Status = Error
			activity.FailMessage = fmt.Sprintf("trigger step #%d of '%s' got error:%v", stepOrdinal+2, stage.Name, err)
		}
	}
}
