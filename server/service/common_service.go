package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/scm"
	"github.com/rancher/pipeline/util"
)

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

func GetSCManager(scmType string) (model.SCManager, error) {
	s, err := GetSCMSetting(scmType)
	if err != nil {
		return nil, err
	}
	var manager model.SCManager
	switch s.ScmType {
	case "github":
		manager = &scm.GithubManager{}
	case "gitlab":
		manager = &scm.GitlabManager{}
	}
	manager = manager.Config(s)
	return manager, nil
}

func GetSCManagerFromSetting(s *model.SCMSetting) (model.SCManager, error) {
	if s == nil {
		return nil, fmt.Errorf("null setting")
	}

	var manager model.SCManager
	switch s.ScmType {
	case "github":
		manager = &scm.GithubManager{}
	case "gitlab":
		manager = &scm.GitlabManager{}
	}
	manager = manager.Config(s)
	return manager, nil
}

func GetSCManagerFromUserID(userId string) (model.SCManager, error) {
	splits := strings.Split(userId, ":")
	if len(splits) != 2 {
		return nil, fmt.Errorf("invalid userId '%s'", userId)
	}
	scmType := splits[0]
	return GetSCManager(scmType)
}

func Reset() error {
	if err := cleanGO("activity"); err != nil {
		return err
	}
	if err := cleanGO("pipeline"); err != nil {
		return err
	}
	if err := cleanGO("pipelineSetting"); err != nil {
		return err
	}
	if err := cleanGO("scmSetting"); err != nil {
		return err
	}
	if err := cleanGO("gitaccount"); err != nil {
		return err
	}
	if err := cleanGO("repocache"); err != nil {
		return err
	}
	if err := cleanGO("pipelineCred"); err != nil {
		return err
	}
	return nil
}

func cleanGO(kind string) error {
	apiClient, err := util.GetRancherClient()
	if err != nil {
		return err
	}
	data, err := PaginateGenericObjects(kind)
	if err != nil {
		return err
	}
	for _, gobj := range data {
		if err := apiClient.GenericObject.Delete(&gobj); err != nil {
			return err
		}
	}
	return nil

}
