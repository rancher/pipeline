package jenkins

import (
	"errors"
	"sync"
)

type jenkinsConfig map[string]string

const JenkinsServerAddress = "JenkinsServerAddress"
const JenkinsUser = "JenkinsUser"
const JenkinsToken = "JenkinsToken"
const CreateJobURI = "CreateJobURI"
const GetCrumbURI = "GetCrumbURI"
const JenkinsCrumbHeader = "JenkinsCrumbHeader"
const JenkinsCrumb = "JenkinsCrumb"
const JenkinsJobBuildURI = "JenkinsJobBuildURI"
const JenkinsJobBuildWithParamsURI = "JenkinsJobBuildWithParamsURI"
const JenkinsTemlpateFolder = "JenkinsTemlpateFolder"
const JenkinsBaseWorkspacePath = "JenkinsBaseWorkspacePath"

var ErrConfigItemNotFound = errors.New("Jenkins configuration not fount")
var ErrJenkinsTemplateNotVaild = errors.New("Jenkins template folder path is not vaild")
var jenkinsConfLock = &sync.RWMutex{}

func (j jenkinsConfig) Set(key, value string) {
	jenkinsConfLock.Lock()
	defer jenkinsConfLock.Unlock()
	j[key] = value
}

func (j jenkinsConfig) Get(key string) (string, error) {
	jenkinsConfLock.RLock()
	defer jenkinsConfLock.RUnlock()
	if value, ok := j[key]; ok {
		return value, nil
	}
	return "", ErrConfigItemNotFound
}

var JenkinsConfig = jenkinsConfig{
	CreateJobURI:                 "/createItem",
	GetCrumbURI:                  "/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)",
	JenkinsJobBuildURI:           "/job/%s/build",
	JenkinsJobBuildWithParamsURI: "/job/%s/buildWithParameters",
}
