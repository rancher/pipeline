package service

import (
	"fmt"
	"strings"

	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/scm"
	"github.com/rancher/pipeline/util"
)

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
