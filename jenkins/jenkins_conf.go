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
const UpdateJobURI = "UpdateJobURI"
const ScriptURI = "ScriptURI"
const DeleteBuildURI = "DeleteBuildURI"
const GetCrumbURI = "GetCrumbURI"
const JenkinsCrumbHeader = "JenkinsCrumbHeader"
const JenkinsCrumb = "JenkinsCrumb"
const JenkinsJobBuildURI = "JenkinsJobBuildURI"
const JenkinsJobInfoURI = "JenkinsJobInfoURI"
const JenkinsBuildInfoURI = "JenkinsBuildInfoURI"
const JenkinsBuildLogURI = "JenkinsBuildLogURI"
const JenkinsJobBuildWithParamsURI = "JenkinsJobBuildWithParamsURI"
const JenkinsTemlpateFolder = "JenkinsTemlpateFolder"

//const JenkinsBaseWorkspacePath = "JenkinsBaseWorkspacePath"
const BuildJobStageConfigFile = "build_stage_example.xml"

//TODO add master
const ScriptSkel = `import hudson.util.RemotingDiagnostics; 
node = "%s"
script = "%s"
cmd = 'def proc = "'+script+'".execute(); proc.waitFor(); println proc.in.text';
for (slave in hudson.model.Hudson.instance.slaves) {
  if(slave.name==node){
	println RemotingDiagnostics.executeGroovy(cmd, slave.getChannel());
  }
}
//on master
if(node == "master"){
	def proc = script.execute(); proc.waitFor(); println proc.in.text
}
`

const GetActiveNodesScript = `for (slave in hudson.model.Hudson.instance.slaves) {
  if (!slave.getComputer().isOffline()){
	    println slave.name;
  }
}
println "master"
`

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
	UpdateJobURI:                 "/job/%s/config.xml",
	DeleteBuildURI:               "/job/%s/lastBuild/doDelete", //purge-job-history/doPurge",
	GetCrumbURI:                  "/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)",
	JenkinsJobBuildURI:           "/job/%s/build",
	JenkinsJobBuildWithParamsURI: "/job/%s/buildWithParameters",
	JenkinsJobInfoURI:            "/job/%s/api/json",
	JenkinsBuildInfoURI:          "/job/%s/lastBuild/api/json",
	JenkinsBuildLogURI:           "/job/%s/lastBuild/timestamps/?elapsed=HH'h'mm'm'ss's'S'ms'&appendLog",
	ScriptURI:                    "/scriptText",
}
