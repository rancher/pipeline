package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	v1client "github.com/rancher/go-rancher/client"
	"github.com/rancher/pipeline/model"

	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

//List All Activities
func (s *Server) ListActivities(rw http.ResponseWriter, req *http.Request) error {

	apiContext := api.GetApiContext(req)
	geObjList, err := service.PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity,err:%v", err)
		return err
	}
	var activities []*model.Activity
	uid, err := util.GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Errorf("cannot get currentUser,%v,%v", uid, err)
	}

	for _, gobj := range geObjList {
		b := []byte(gobj.ResourceData["data"].(string))
		a := &model.Activity{}
		json.Unmarshal(b, a)
		model.ToActivityResource(apiContext, a)
		if a.CanApprove(uid) {
			//add approve action
			a.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=approve"
			a.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=deny"
		}
		activities = append(activities, a)
	}

	datalist := priorityPendingActivity(activities)
	apiContext.Write(&v1client.GenericCollection{
		Data: datalist,
	})

	return nil

}

func (s *Server) CleanActivities(rw http.ResponseWriter, req *http.Request) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	geObjList, err := service.PaginateGenericObjects("activity")
	if err != nil {
		logrus.Errorf("fail to list activity,err:%v", err)
		return err
	}
	for _, gobj := range geObjList {
		apiClient.GenericObject.Delete(&gobj)
	}
	return nil

}

func (s *Server) CleanPipelines(rw http.ResponseWriter, req *http.Request) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	geObjList, err := service.PaginateGenericObjects("pipeline")
	if err != nil {
		logrus.Errorf("fail to list pipeline,err:%v", err)
		return err
	}
	for _, gobj := range geObjList {
		apiClient.GenericObject.Delete(&gobj)
	}
	return nil
}

//CreateActivity Handler
func (s *Server) CreateActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	activity := &model.Activity{}

	if err := json.Unmarshal(requestBytes, activity); err != nil {
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, activity.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", activity.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = service.CreateActivity(activity); err != nil {
		return err
	}
	model.ToActivityResource(apiContext, activity)
	apiContext.Write(activity)
	return nil

}

func (s *Server) RerunActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := service.GetActivity(id)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//TODO validate activity
	//validate git account access
	if !service.ValidAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = service.ResetActivity(s.Provider, r); err != nil {
		logrus.Errorf("reset activity error:%v", err)
		return err
	}

	if err = service.RerunActivity(s.Provider, r); err != nil {
		logrus.Errorf("rerun activity error:%v", err)
		return err
	}
	if err = service.UpdateActivity(r); err != nil {
		logrus.Errorf("update activity error:%v", err)
		return err
	}
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *r,
	}
	model.ToActivityResource(apiContext, r)
	apiContext.Write(r)
	return nil
}

func (s *Server) ApproveActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)
	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := service.GetActivity(id)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = service.ApproveActivity(s.Provider, r); err != nil {
		logrus.Errorf("fail approve activity:%v", err)
		return err
	}
	r.Status = model.ActivityWaiting
	r.ActivityStages[r.PendingStage].Status = model.ActivityStageWaiting
	r.PendingStage = 0
	if err = service.UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}
	s.UpdateLastActivity(r)
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *r,
	}
	model.ToActivityResource(apiContext, r)
	apiContext.Write(r)
	return nil

}

func (s *Server) DenyActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := service.GetActivity(id)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = service.DenyActivity(r); err != nil {
		logrus.Errorf("fail denyActivity:%v", err)
		return err
	}
	if err = service.UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *r,
	}
	s.UpdateLastActivity(r)
	model.ToActivityResource(apiContext, r)
	apiContext.Write(r)
	return nil

}

func (s *Server) StopActivity(rw http.ResponseWriter, req *http.Request) error {
	id := mux.Vars(req)["id"]
	apiContext := api.GetApiContext(req)

	mutex := MyAgent.getActivityLock(id)
	mutex.Lock()
	defer mutex.Unlock()

	r, err := service.GetActivity(id)
	if err != nil {
		logrus.Errorf("fail getting activity with id:%v", id)
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}

	if err = service.StopActivity(s.Provider, r); err != nil {
		logrus.Errorf("fail stop activity:%v", err)
		return err
	}
	if err = service.UpdateActivity(r); err != nil {
		logrus.Errorf("fail update activity:%v", err)
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         *r,
	}
	s.UpdateLastActivity(r)
	model.ToActivityResource(apiContext, r)
	apiContext.Write(r)
	return nil

}

func (s *Server) DeleteActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]
	r, err := service.GetActivity(id)
	if err != nil {
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, r.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", r.Pipeline.Stages[0].Steps[0].GitUser)
	}
	err = service.DeleteActivity(id)
	if err != nil {
		return err
	}
	apiContext.Write(model.ToActivityResource(apiContext, r))
	return nil
}

func (s *Server) UpdateActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	requestBytes, err := ioutil.ReadAll(req.Body)
	activity := &model.Activity{}

	if err := json.Unmarshal(requestBytes, activity); err != nil {
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, activity.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", activity.Pipeline.Stages[0].Steps[0].GitUser)
	}
	err = service.UpdateActivity(activity)
	if err != nil {
		return err
	}
	model.ToActivityResource(apiContext, activity)
	apiContext.Write(activity)
	return nil
}

func (s *Server) GetActivity(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	a, err := service.GetActivity(id)
	if err != nil {
		return err
	}
	//validate git account access
	if !service.ValidAccountAccess(req, a.Pipeline.Stages[0].Steps[0].GitUser) {
		return fmt.Errorf("no access to '%s' git account", a.Pipeline.Stages[0].Steps[0].GitUser)
	}

	model.ToActivityResource(apiContext, a)
	uid, err := util.GetCurrentUser(req.Cookies())
	if err != nil || uid == "" {
		logrus.Errorf("get currentUser fail,%v,%v", uid, err)
	}
	if a.CanApprove(uid) {
		//add approve action
		a.Actions["approve"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=approve"
		a.Actions["deny"] = apiContext.UrlBuilder.ReferenceLink(a.Resource) + "?action=deny"
	}
	return apiContext.WriteResource(a)
}

func priorityPendingActivity(activities []*model.Activity) []interface{} {
	var actilist []interface{}
	var pendinglist []interface{}
	for _, a := range activities {
		if a.Status == model.ActivityPending {
			pendinglist = append(pendinglist, a)
		} else {
			actilist = append(actilist, a)
		}
	}
	actilist = append(pendinglist, actilist...)
	return actilist
}

//update last activity info in the pipeline on activity changes
func (s *Server) UpdateLastActivity(activity *model.Activity) {
	logrus.Debugf("begin UpdateLastActivity")
	pId := activity.Pipeline.Id
	p := service.GetPipelineById(pId)
	if p == nil || p.LastRunId == "" {
		return
	}
	if activity.Id != p.LastRunId {
		return
	}
	p.LastRunStatus = activity.Status
	p.CommitInfo = activity.CommitInfo
	p.NextRunTime = service.GetNextRunTime(p)

	if err := service.UpdatePipeline(p); err != nil {
		logrus.Errorf("fail update pipeline last run status,%v", err)
	}
	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "pipeline",
		Time:         time.Now(),
		Data:         *p,
	}
}
